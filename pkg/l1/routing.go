package l1

import (
	"crypto/ed25519"
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"net"
	"sync"
	"time"

	"ipvn7/pkg/l0"
)

const (
	NumRings     = 12
	PeersPerRing = 10
	MaxPeers     = NumRings * PeersPerRing // 120 pares máximo según especificación
)

// Estados de la Máquina de Estados Finitos (FSM) de salud de pares
const (
	HealthStateHealthy     = 0 // Flujo óptimo y pacing respetado
	HealthStateDegraded    = 1 // Incremento de RTT o ligera varianza de Jitter
	HealthStateUnstable    = 2 // Pérdida moderada de paquetes; activa Make-Before-Break
	HealthStateUnreachable = 3 // Ausencia total de ACKs o pérdida crítica
	HealthStateQuarantined = 4 // Anomalía maliciosa o aislamiento de vecindario
)

// PeerLocator representa la dirección física efímera de un par
type PeerLocator struct {
	PhysicalAddr *net.UDPAddr
	LastSeen     time.Time
	LatencyMs    float64
}

// PeerNode representa un par en la red en malla
type PeerNode struct {
	DID             string
	PublicKey       ed25519.PublicKey
	Locator         PeerLocator
	RingIndex       int
	HealthState     int
	JitterMs        float64
	LossRate        float64
	HealthScore     float64
	QuarantineUntil time.Time
}

// KleinbergRouter implementa el enrutamiento geométrico de Mundo Pequeño
type KleinbergRouter struct {
	mu          sync.RWMutex
	LocalDID    string
	LocalPub    ed25519.PublicKey
	rings       [NumRings][]*PeerNode
	peerIndex   map[string]*PeerNode
	TotalRoutes int
}

// NewKleinbergRouter inicializa el enrutador con la identidad local
func NewKleinbergRouter(localID *l0.Identity) *KleinbergRouter {
	return &KleinbergRouter{
		LocalDID:  localID.DID(),
		LocalPub:  localID.PublicKey,
		peerIndex: make(map[string]*PeerNode),
	}
}

// XORKeyDistance calcula la distancia XOR de 256 bits y determina el anillo logarítmico (0 a 11)
func XORKeyDistance(keyA, keyB []byte) int {
	if len(keyA) != 32 || len(keyB) != 32 {
		return NumRings - 1
	}

	// Contar ceros iniciales en XOR (Leading Zeros)
	lz := 0
	for i := 0; i < 4; i++ {
		chunkA := binary.BigEndian.Uint64(keyA[i*8 : (i+1)*8])
		chunkB := binary.BigEndian.Uint64(keyB[i*8 : (i+1)*8])
		xor := chunkA ^ chunkB
		if xor == 0 {
			lz += 64
		} else {
			lz += bits.LeadingZeros64(xor)
			break
		}
	}

	// Mapear los 256 bits a 12 anillos logarítmicos
	// Mayor cantidad de ceros iniciales = nodos más cercanos = Anillo 0
	ring := (256 - lz) / (256 / NumRings)
	if ring >= NumRings {
		ring = NumRings - 1
	}
	if ring < 0 {
		ring = 0
	}
	return ring
}

// AddOrUpdatePeer inserta o actualiza un par en la tabla de 120 slots
func (r *KleinbergRouter) AddOrUpdatePeer(did string, addr *net.UDPAddr, latencyMs float64) error {
	pub, err := l0.PublicKeyFromDID(did)
	if err != nil {
		return fmt.Errorf("did inválido: %w", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Si el par ya existe, actualizamos su localizador (IP Roaming sin caída)
	if existing, found := r.peerIndex[did]; found {
		existing.Locator.PhysicalAddr = addr
		existing.Locator.LastSeen = time.Now()
		existing.Locator.LatencyMs = latencyMs
		return nil
	}

	ring := XORKeyDistance(r.LocalPub, pub)

	// Comprobar límite de 10 pares por anillo
	if len(r.rings[ring]) >= PeersPerRing {
		// Reemplazar el par con peor latencia o más inactivo
		worstIdx := 0
		worstLatency := r.rings[ring][0].Locator.LatencyMs
		for i, p := range r.rings[ring] {
			if p.Locator.LatencyMs > worstLatency {
				worstLatency = p.Locator.LatencyMs
				worstIdx = i
			}
		}

		if latencyMs < worstLatency {
			oldPeer := r.rings[ring][worstIdx]
			delete(r.peerIndex, oldPeer.DID)

			newPeer := &PeerNode{
				DID:       did,
				PublicKey: pub,
				Locator: PeerLocator{
					PhysicalAddr: addr,
					LastSeen:     time.Now(),
					LatencyMs:    latencyMs,
				},
				RingIndex:   ring,
				HealthState: HealthStateHealthy,
				HealthScore: 100.0,
			}
			r.rings[ring][worstIdx] = newPeer
			r.peerIndex[did] = newPeer
		}
		return nil
	}

	newPeer := &PeerNode{
		DID:       did,
		PublicKey: pub,
		Locator: PeerLocator{
			PhysicalAddr: addr,
			LastSeen:     time.Now(),
			LatencyMs:    latencyMs,
		},
		RingIndex:   ring,
		HealthState: HealthStateHealthy,
		HealthScore: 100.0,
	}

	r.rings[ring] = append(r.rings[ring], newPeer)
	r.peerIndex[did] = newPeer
	r.TotalRoutes++
	return nil
}

// CalculateHealthScore calcula el puntaje de mérito de 0 a 100 y el estado FSM
func CalculateHealthScore(pacingCompliance, jitterMs, lossRate float64) (float64, int) {
	jitterPen := (jitterMs / 50.0) * 30.0
	if jitterPen > 30.0 {
		jitterPen = 30.0
	}
	lossPen := lossRate * 40.0
	if lossPen > 40.0 {
		lossPen = 40.0
	}

	score := (pacingCompliance * 30.0) + (30.0 - jitterPen) + (40.0 - lossPen)
	if score < 0 {
		score = 0
	} else if score > 100 {
		score = 100
	}

	state := HealthStateHealthy
	if score < 25.0 || lossRate >= 0.4 {
		state = HealthStateUnreachable
	} else if score < 50.0 || lossRate >= 0.20 || jitterMs > 60.0 {
		state = HealthStateUnstable
	} else if score < 75.0 || lossRate >= 0.05 || jitterMs > 25.0 {
		state = HealthStateDegraded
	}
	return score, state
}

// UpdatePeerHealth actualiza las métricas empíricas y transiciona la FSM del par
func (r *KleinbergRouter) UpdatePeerHealth(did string, pacingCompliance, jitterMs, lossRate float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	peer, found := r.peerIndex[did]
	if !found {
		return
	}

	score, state := CalculateHealthScore(pacingCompliance, jitterMs, lossRate)
	peer.JitterMs = jitterMs
	peer.LossRate = lossRate
	peer.HealthScore = score
	peer.HealthState = state
}

// FindNextHop implementa el reenvío voraz hacia la clave destino con conmutación proactiva Make-Before-Break
func (r *KleinbergRouter) FindNextHop(destDID string) (*PeerNode, error) {
	pubDest, err := l0.PublicKeyFromDID(destDID)
	if err != nil {
		return nil, fmt.Errorf("did destino inválido: %w", err)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	// 1. Si está en índice directo y saludable, entrega directa
	if direct, found := r.peerIndex[destDID]; found {
		if direct.HealthState == HealthStateHealthy {
			return direct, nil
		}
		// Si está degradado o inestable (Make-Before-Break), intentamos buscar un intermediario saludable
		// pero si no hay mejor, usamos el directo
	}

	// 2. Búsqueda voraz del par con menor distancia XOR al destino y mayor salud
	var bestPeer *PeerNode
	bestRing := NumRings
	bestScore := -1.0

	for _, peer := range r.peerIndex {
		// Descartar temporalmente nodos inaccesibles o en cuarentena
		if peer.HealthState == HealthStateQuarantined || peer.HealthState == HealthStateUnreachable {
			continue
		}

		distRing := XORKeyDistance(peer.PublicKey, pubDest)
		if distRing < bestRing || (distRing == bestRing && peer.HealthScore > bestScore) {
			bestRing = distRing
			bestScore = peer.HealthScore
			bestPeer = peer
		}
	}

	if bestPeer == nil {
		// Fallback: si todos los pares están degradados, retornar el mejor disponible
		for _, peer := range r.peerIndex {
			if bestPeer == nil || peer.HealthScore > bestPeer.HealthScore {
				bestPeer = peer
			}
		}
	}

	if bestPeer == nil {
		return nil, errors.New("no hay rutas disponibles hacia el destino")
	}

	return bestPeer, nil
}

// GetAllPeers retorna una instantánea de los pares registrados
func (r *KleinbergRouter) GetAllPeers() []*PeerNode {
	r.mu.RLock()
	defer r.mu.RUnlock()

	res := make([]*PeerNode, 0, len(r.peerIndex))
	for _, p := range r.peerIndex {
		pCopy := *p
		res = append(res, &pCopy)
	}
	return res
}

// HandleRoamingUpdate procesa un paquete de roaming y actualiza el localizador
func (r *KleinbergRouter) HandleRoamingUpdate(pkt *l0.Packet, newAddr *net.UDPAddr) error {
	valid, err := pkt.VerifyPacketSignature()
	if err != nil || !valid {
		return errors.New("firma de paquete de roaming inválida")
	}

	return r.AddOrUpdatePeer(pkt.SourceDID, newAddr, 1.1) // 1.1ms benchmark de conmutación
}
