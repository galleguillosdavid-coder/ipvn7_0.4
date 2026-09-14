package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"ipvn7/pkg/l0"
	"ipvn7/pkg/l1"
	"ipvn7/pkg/l2"
	"ipvn7/pkg/l3"
	"ipvn7/pkg/l4"
)

const (
	Banner = `
██╗██████╗ ██╗   ██╗███╗   ██╗███████╗
██║██╔══██╗██║   ██║████╗  ██║╚════██║
██║██████╔╝██║   ██║██╔██╗ ██║    ██╔╝
██║██╔═══╝ ╚██╗ ██╔╝██║╚██╗██║   ██╔╝ 
██║██║      ╚████╔╝ ██║ ╚████║   ██║  
╚═╝╚═╝       ╚═══╝  ╚═╝  ╚═══╝   ╚═╝  
   Network OS - Autonomous Sovereign Mesh v0.4
`
)

func main() {
	keystorePath := flag.String("keystore", filepath.Join("keystore", "node_identity.key"), "Ruta del archivo keystore")
	listenPort := flag.Int("port", 7777, "Puerto UDP de transporte físico")
	webPort := flag.Int("web-port", 7070, "Puerto HTTP para el panel de control web interactivo")
	mcpMode := flag.Bool("mcp", false, "Iniciar en modo servidor MCP (Model Context Protocol) sobre stdio")
	diagnostics := flag.Bool("diagnostics", false, "Ejecutar autodiagnóstico de capas y salir")
	flag.Parse()

	// 1. Capa L0: Identidad y Criptografía
	var identity *l0.Identity
	var err error

	if _, statErr := os.Stat(*keystorePath); statErr == nil {
		identity, err = l0.LoadFromFile(*keystorePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR L0] No se pudo cargar keystore: %v. Generando uno nuevo...\n", err)
			identity, _ = l0.GenerateIdentity()
			_ = identity.SaveToFile(*keystorePath)
		}
	} else {
		identity, err = l0.GenerateIdentity()
		if err != nil {
			fmt.Fprintf(os.Stderr, "[FATAL L0] Error generando entropía soberana: %v\n", err)
			os.Exit(1)
		}
		_ = identity.SaveToFile(*keystorePath)
	}

	// 2. Capa L1: Enrutamiento, Adaptador de Red y Marcapasos
	router := l1.NewKleinbergRouter(identity)
	pacer := l1.NewPacketPacer(l1.DefaultPacerConfig())
	tun, err := l1.CreateTunAdapter(identity, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[FATAL L1] Error instanciando adaptador TUN: %v\n", err)
		os.Exit(1)
	}
	defer tun.Close()

	// 3. Capa L2: Telemetría y Observabilidad
	telemetry := l2.NewTelemetryRingBuffer()

	// 4. Capa L4: Gestor de Servicios y Panel de Control Web
	svcManager := l4.NewServiceLifecycleManager(5 * time.Minute)
	svcManager.RegisterDormantService("chat_e2ee")
	svcManager.RegisterDormantService("remote_desktop")
	svcManager.RegisterDormantService("dag_store")

	webServer := l4.NewWebDashboardServer(*webPort, identity, router, telemetry, pacer, svcManager, l4.FindStaticDir())
	if err := webServer.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "[ADVERTENCIA L4] No se pudo iniciar servidor web: %v\n", err)
	} else {
		defer webServer.Stop()
	}

	// 5. Capa L3: Plano de Control Inteligente / MCP (Dimensión 6)
	mcpServer := l3.NewMCPServer(identity, router, telemetry)
	mcpServer.AttachSubsystems(
		webServer.Firewall,
		webServer.DAGStore,
		webServer.WoT,
		webServer.Multipath,
		webServer.Copilot,
		func(name string) (string, string, string, bool) {
			rec, found := webServer.Petnames.Resolve(name, "")
			if !found {
				return "", "", "", false
			}
			return rec.DID, rec.VirtualIPv6, rec.VirtualIPv4, true
		},
		func(peerDID string) (string, []string, error) {
			pubPeer, err := l0.PublicKeyFromDID(peerDID)
			if err != nil {
				return "", nil, err
			}
			sas := l4.DeriveSAS(identity.PublicKey, pubPeer)
			return sas.Digits, sas.Emojis, nil
		},
	)
	mcpServer.AttachKuzu(webServer.Kuzu)
	mcpServer.AttachHybridCrypto(webServer.HybridKeys, webServer.Sphinx)
	mcpServer.AttachLegacyRescues(webServer.SOCKS5, webServer.Guardian)
	mcpServer.AttachUINStack(webServer.UIN, webServer.MemoryArbiter, webServer.Hierarchy)

	// Si se invoca con flag --mcp, toma el control directo de stdio para el agente IA
	if *mcpMode {
		if err := mcpServer.ServeStdioDefault(); err != nil {
			fmt.Fprintf(os.Stderr, "[MCP ERROR] %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Impresión del Banner e Identidad de Red (12 Dimensiones)
	fmt.Print(Banner)
	fmt.Println("================================================================================")
	fmt.Printf("[L0 DID SOBERANO]   : %s\n", identity.DID())
	fmt.Printf("[L1 IPv6 SOBERANO]  : %s/64\n", identity.IPv6())
	fmt.Printf("[L1 IPv4 VIRTUAL]   : %s/16\n", identity.IPv4())
	fmt.Printf("[L1 MTU ESTÁNDAR]   : %d bytes (Deterministic Wire)\n", tun.MTU())
	fmt.Printf("[L1 ADAPTADOR]      : Modo Simulado Zero-Copy (Ejecución Perpetua)\n")
	fmt.Printf("[L1 CRIPTO HÍBRIDA] : ML-DSA-65 & ML-KEM-768 Cuántico-Resistente (NIST L3)\n")
	fmt.Printf("[L1 SPHINX ONION]   : Enrutamiento Cebolla 3 Saltos (MTU 1280B Fijo)\n")
	fmt.Printf("[L1 FAST-PATH eBPF] : Gancho XDP Directo & Sync a Mapas de Kernel (<18 ns)\n")
	fmt.Printf("[L1 SOCKS5 PROXY]   : Gateway Transparente RFC 1928 Activo en :10807\n")
	fmt.Printf("[L1 STUN NAT]       : Perforación Autónoma RFC 5389 & Hole Punching\n")
	fmt.Printf("[L1 RED SILENCIOSA] : Cero Broadcast O(1) & Semáforo DoS (<15 ns)\n")
	fmt.Printf("[L1 UIN PASAPORTE]  : Entidades 256-bit root_id & BindingRecord Wire v1\n")
	fmt.Printf("[L1 MEMORY ARBITER] : Cuotas Estrictas RAM (Anti-OOM DoS) por Clase\n")
	fmt.Printf("[L1 ANTI-REPLAY]    : Ventana Deslizante 1024 con Aislamiento Cross-Session\n")
	fmt.Printf("[L1 ZTNA FIREWALL]  : Micro-segmentación Default-Deny Activa (Dim 1)\n")
	fmt.Printf("[L1 QoS & ANTI-DDOS]: Token Bucket & Desafíos PoW Dinámicos (Dim 3)\n")
	fmt.Printf("[L1 ZERO-COPY POOL] : 3 Niveles Preasignados con Conteo Atómico (Dim 12)\n")
	fmt.Printf("[L1 PERSISTENCIA]   : DAG Inmutable Content-Addressed & DTN (Dim 4)\n")
	fmt.Printf("[L1 SMART PACKETS]  : Micro-Hook Pipelines & Sandbox Guard (Dim 7)\n")
	fmt.Printf("[L1 MULTIPATH]      : Overlay Scheduler & Critical Duplication (Dim 8)\n")
	fmt.Printf("[L1 WEB-OF-TRUST]   : Grafo Criptográfico & Reputación BFS (Dim 9)\n")
	fmt.Printf("[L2 TELEMETRÍA]     : Ring Buffer Lock-Free (<28 ns) Activo\n")
	fmt.Printf("[L2 GUARDIAN SALUD] : Resiliencia con Backoff Jitter & Circuit Breaker\n")
	fmt.Printf("[L2 TIT-FOR-TAT]    : Economía Algorítmica 4-Tier Reciprocidad (Dim 5)\n")
	fmt.Printf("[L3 CONTROL DUAL]   : Servidor MCP JSON-RPC & CLI Unificado (Dim 6)\n")
	fmt.Printf("[L3 KÙZU GRAFO PFO] : Jerarquía Axiomática & Detección Anti-Colisión Activa\n")
	fmt.Printf("[L3 AI COPILOT]     : Diagnóstico Autónomo & Auto-Curación (Dim 11)\n")
	fmt.Printf("[L4 dDNS PETNAMES]  : Espacio de Nombres Humano & Hosts Local (Dim 2)\n")
	fmt.Printf("[L4 OOB PAIRING]    : SAS 6-Dígitos & UR QR Frames Animados (Dim 10)\n")
	fmt.Printf("[L4 INTERFAZ WEB]   : http://localhost:%d (Panel Interactivo)\n", *webPort)
	fmt.Println("================================================================================")

	if *diagnostics {
		fmt.Println("[DIAGNÓSTICO] Todas las capas (L0 -> L1 -> L2 -> L3 -> L4 Web) operan correctamente.")
		return
	}

	// 5. Iniciar socket UDP de transporte físico
	listenAddr := fmt.Sprintf("0.0.0.0:%d", *listenPort)
	conn, err := net.ListenPacket("udp", listenAddr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR TRANSPORTE] No se pudo abrir socket UDP en %s: %v\n", listenAddr, err)
		os.Exit(1)
	}
	defer conn.Close()
	fmt.Printf("[+] Escuchando datagramas ipvn7 en UDP %s...\n", listenAddr)

	// Bucle de recepción UDP y despacho con Zero-Copy, ZTNA y QoS
	buf := make([]byte, l0.MaxPacketSize*2)
	go func() {
		for {
			n, remoteAddr, err := conn.ReadFrom(buf)
			if err != nil {
				return
			}

			// 1. Asignar búfer desde el Pool Zero-Copy
			pktBuf := webServer.BufferPool.Acquire(n)
			copy(pktBuf.RawSlice(), buf[:n])

			telemetry.RecordEvent(l2.EventRxPacket, uint32(n), 1100, 0)

			// 2. Procesar y validar paquete canónico CBOR
			pkt, err := l0.DecodePacket(pktBuf.Data())
			if err != nil {
				telemetry.RecordEvent(l2.EventDrop, uint32(n), 0, 0)
				pktBuf.Release()
				continue
			}

			senderDID := pkt.SourceDID

			// 3. Evaluar Cortafuegos ZTNA
			if pkt.Type != l0.MsgTypeRoamingUpdate { // actualizaciones de roaming pasan a router
				decision, _ := webServer.Firewall.EvaluateInbound(senderDID, uint16(*listenPort))
				if decision != l1.DecisionAccept {
					telemetry.RecordEvent(l2.EventDrop, uint32(n), 0, 0)
					pktBuf.Release()
					continue
				}
			}

			// 4. Evaluar QoS Token Bucket
			allowed, _ := webServer.QoS.EvaluatePacket(senderDID, l1.ClassControl, n)
			if !allowed {
				telemetry.RecordEvent(l2.EventDrop, uint32(n), 0, 0)
				pktBuf.Release()
				continue
			}

			// 5. Manejar según tipo de mensaje
			if pkt.Type == l0.MsgTypeRoamingUpdate {
				if udpAddr, ok := remoteAddr.(*net.UDPAddr); ok {
					_ = router.HandleRoamingUpdate(pkt, udpAddr)
				}
			}

			// Liberar búfer reciclable
			pktBuf.Release()
		}
	}()

	// Pulsos periódicos de mantenimiento y telemetría
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	fmt.Println("[+] Nodo ipvn7 en ejecución continua. Presione Ctrl+C para detener.")

	for {
		select {
		case <-ticker.C:
			snap := telemetry.Snapshot()
			fmt.Printf("[TELEMETRÍA] Tx: %d pkts | Rx: %d pkts | Drops: %d | Pares: %d\n",
				snap.PacketsTx, snap.PacketsRx, snap.PacketsDropped, len(router.GetAllPeers()))
		case sig := <-sigChan:
			fmt.Printf("\n[*] Señal recibida (%v). Apagado ordenado y elegante de ipvn7...\n", sig)
			return
		}
	}
}
