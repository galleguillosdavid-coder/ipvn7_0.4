// Package l1 implementa el Motor de Descubrimiento Autónomo de Malla Soberana.
// Permite que nodos aislados, desconectados de la LAN o detrás de NAT/CGNAT,
// se descubran y emparejen automáticamente mediante balizas ciegas EBRA y STUN reflexivo.
package l1

import (
	"crypto/rand"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"ipvn7/pkg/l0"
)

// KuzuPeerUpserter abstrae la sincronización topológica con KùzuDB
type KuzuPeerUpserter interface {
	UpsertPeer(did, alias, peerType string, isCurrent bool)
	UpsertXORLink(sourceDID, targetDID, relation string, latencyMs float64, hopDistance int8, iface string)
}

// AutonomousDiscoveryEngine orquesta el descubrimiento continuo y autónomo de pares
type AutonomousDiscoveryEngine struct {
	mu         sync.Mutex
	Identity   *l0.Identity
	Router     *KleinbergRouter
	Rendezvous *BlindRendezvousManager
	Firewall   *ZTNAFirewall
	Kuzu       KuzuPeerUpserter
	PacketConn net.PacketConn
	ListenPort int
	running    bool
	stopChan   chan struct{}
	discovered map[string]time.Time
}

// NewAutonomousDiscoveryEngine inicializa el motor de descubrimiento
func NewAutonomousDiscoveryEngine(
	id *l0.Identity,
	router *KleinbergRouter,
	rz *BlindRendezvousManager,
	fw *ZTNAFirewall,
	kuzu KuzuPeerUpserter,
	conn net.PacketConn,
	port int,
) *AutonomousDiscoveryEngine {
	return &AutonomousDiscoveryEngine{
		Identity:   id,
		Router:     router,
		Rendezvous: rz,
		Firewall:   fw,
		Kuzu:       kuzu,
		PacketConn: conn,
		ListenPort: port,
		discovered: make(map[string]time.Time),
		stopChan:   make(chan struct{}),
	}
}

// Start inicia el bucle de descubrimiento autónomo en segundo plano
func (e *AutonomousDiscoveryEngine) Start() {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true
	e.mu.Unlock()

	go e.discoveryLoop()
}

// Stop detiene el bucle de descubrimiento
func (e *AutonomousDiscoveryEngine) Stop() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !e.running {
		return
	}
	e.running = false
	close(e.stopChan)
}

func (e *AutonomousDiscoveryEngine) discoveryLoop() {
	// Pulso inicial inmediato
	e.executeCycle()

	ticker := time.NewTicker(12 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			e.executeCycle()
		}
	}
}

func (e *AutonomousDiscoveryEngine) executeCycle() {
	currentPeers := len(e.Router.GetAllPeers())

	// Si ya alcanzamos al menos 2 pares directos y el circuit breaker se disparó, reposar
	if currentPeers >= MinPeerTripThreshold && e.Rendezvous.IsCircuitBroken() {
		return
	}

	// 1. Recolectar interfaces locales y sonda reflexiva STUN RFC 5389
	var candidateEndpoints []string

	stunRes, _ := ProbeSTUN(nil, 1500*time.Millisecond)
	if stunRes != nil && stunRes.PublicIP != "" {
		candidateEndpoints = append(candidateEndpoints, fmt.Sprintf("%s:%d", stunRes.PublicIP, e.ListenPort))
	}

	localIPs := getLocalNonLoopbackIPs()
	for _, ip := range localIPs {
		candidateEndpoints = append(candidateEndpoints, fmt.Sprintf("%s:%d", ip, e.ListenPort))
	}

	if len(candidateEndpoints) == 0 {
		candidateEndpoints = append(candidateEndpoints, fmt.Sprintf("127.0.0.1:%d", e.ListenPort))
	}

	endpointPayload := strings.Join(candidateEndpoints, ",")

	// 2. Publicar Baliza Ciega Efímera EBRA
	if !e.Rendezvous.IsCircuitBroken() {
		_, _ = e.Rendezvous.PublishBlindBeacon(endpointPayload, 0)
	}

	// 3. Enviar datagrama broadcast LAN de cortesía (por si ambos nodos comparten LAN)
	e.broadcastLocalBeacon()

	// 4. Descubrir balizas de pares remotos en el tópico EBRA
	beacons, err := e.Rendezvous.DiscoverAllPeers(0)
	if err != nil || len(beacons) == 0 {
		return
	}

	for _, peerBeacon := range beacons {
		if peerBeacon.DID == e.Identity.DID() {
			continue
		}

		endpoints := strings.Split(peerBeacon.Endpoint, ",")
		for _, ep := range endpoints {
			ep = strings.TrimSpace(ep)
			if ep == "" {
				continue
			}

			udpAddr, err := net.ResolveUDPAddr("udp", ep)
			if err != nil {
				continue
			}

			// Autorizar en Cortafuegos ZTNA
			if e.Firewall != nil {
				e.Firewall.AuthorizeDID(&DIDPolicy{
					DID:           peerBeacon.DID,
					AllowInbound:  true,
					AllowOutbound: true,
					AllowRelay:    true,
				})
			}

			// Registrar en Kleinberg Router
			_ = e.Router.AddOrUpdatePeer(peerBeacon.DID, udpAddr, 2.5)

			// Sincronizar en Grafo KùzuDB
			if e.Kuzu != nil {
				e.Kuzu.UpsertPeer(peerBeacon.DID, "", "kleinberg_peer", false)
				e.Kuzu.UpsertXORLink(e.Identity.DID(), peerBeacon.DID, "xor_metric", 2.5, 0, "EBRA-WAN")
			}

			// Enviar datagrama UDP firmado de RoamingUpdate para perforar NAT (Hole Punching)
			e.sendHandshakePacket(peerBeacon.DID, udpAddr)

			e.mu.Lock()
			last, exists := e.discovered[peerBeacon.DID]
			if !exists || time.Since(last) > 30*time.Second {
				e.discovered[peerBeacon.DID] = time.Now()
				fmt.Printf("[DESCUBRIMIENTO AUTÓNOMO EBRA] ¡Par descubierto!: %s en %s\n", peerBeacon.DID[:24]+"...", ep)
			}
			e.mu.Unlock()

			e.Rendezvous.NotifyDirectPeerConnected()
		}
	}
}

func (e *AutonomousDiscoveryEngine) broadcastLocalBeacon() {
	if e.PacketConn == nil {
		return
	}

	nonce := make([]byte, 16)
	_, _ = rand.Read(nonce)

	pkt := l0.NewPacket(
		l0.MsgTypeRoamingUpdate,
		e.Identity.DID(),
		"",
		1,
		nonce,
		[]byte(fmt.Sprintf("%d", e.ListenPort)),
	)
	_ = pkt.SignPacket(e.Identity)
	encoded, err := pkt.Encode()
	if err != nil {
		return
	}

	targetPorts := []int{7777, 7778, 7001, 8080}
	for _, p := range targetPorts {
		dst := &net.UDPAddr{IP: net.IPv4bcast, Port: p}
		_, _ = e.PacketConn.WriteTo(encoded, dst)
	}
}

func (e *AutonomousDiscoveryEngine) sendHandshakePacket(targetDID string, targetAddr *net.UDPAddr) {
	if e.PacketConn == nil {
		return
	}

	nonce := make([]byte, 16)
	_, _ = rand.Read(nonce)

	pkt := l0.NewPacket(
		l0.MsgTypeRoamingUpdate,
		e.Identity.DID(),
		targetDID,
		uint64(time.Now().UnixNano()),
		nonce,
		[]byte(fmt.Sprintf("endpoint:%d", e.ListenPort)),
	)
	_ = pkt.SignPacket(e.Identity)
	encoded, err := pkt.Encode()
	if err != nil {
		return
	}

	_, _ = e.PacketConn.WriteTo(encoded, targetAddr)
}

func getLocalNonLoopbackIPs() []string {
	var ips []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil && !ip.IsLoopback() && ip.To4() != nil {
				ips = append(ips, ip.String())
			}
		}
	}
	return ips
}
