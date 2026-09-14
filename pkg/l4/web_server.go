package l4

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"ipvn7/pkg/l0"
	"ipvn7/pkg/l1"
	"ipvn7/pkg/l2"
	"ipvn7/pkg/l3"
)

// WebDashboardServer orquesta la interfaz gráfica web y los endpoints REST
type WebDashboardServer struct {
	Port       int
	Identity   *l0.Identity
	Router     *l1.KleinbergRouter
	Telemetry  *l2.TelemetryRingBuffer
	Pacer      *l1.PacketPacer
	SvcManager *ServiceLifecycleManager
	Firewall   *l1.ZTNAFirewall
	BufferPool *l1.BufferPool
	QoS        *l1.QoSManager
	DAGStore     *l1.DAGStore
	Accounting   *l2.TransitAccounting
	WoT          *l1.WebOfTrust
	Petnames     *PetnameResolver
	SmartPackets *l1.SmartPacketPipeline
	Multipath    *l1.MultipathScheduler
	Copilot      *l3.AICopilotEngine
	WDRR         *l1.WDRRScheduler
	AuditLog     *l1.MerkleAuditLog
	Semantic     *l1.SemanticCatalog
	DarkNode     *l1.DarkNodeMacroOrchestrator
	Fingerprinter *l2.AnomalyFingerprinter
	Kuzu         *l3.KuzuGraphEngine
	HybridKeys   *l1.HybridKeyPair
	Sphinx       *l1.SphinxRouter
	FastPath     *l1.XDPFastPathEngine
	SOCKS5        *l1.SOCKS5Gateway
	Silent        *l1.SilentDispatcher
	Guardian      *l2.ResilienceGuardian
	MemoryArbiter *l1.GlobalMemoryArbiter
	UIN           *l1.UINIdentityManager
	AntiReplay    *l1.AntiReplayFilter
	Hierarchy     *l1.NodeHierarchyManager
	Chat          *ChatManager
	Multicast     *l1.CascadeMulticastEngine
	RemoteDesktop *RemoteDesktopManager
	CorporateVPN  *l1.CorporateVPNManager
	LiveStream    *LiveStreamManager
	AudioRadio    *AudioRadioManager
	Probe         *l1.NetworkProbeEngine
	Blackout      *l1.BlackoutRecoveryManager
	CopilotMode   string
	StartTime     time.Time
	StaticDir     string
	server        *http.Server
}

// NewWebDashboardServer inicializa el servidor web
func NewWebDashboardServer(
	port int,
	id *l0.Identity,
	router *l1.KleinbergRouter,
	telemetry *l2.TelemetryRingBuffer,
	pacer *l1.PacketPacer,
	svcMgr *ServiceLifecycleManager,
	staticDir string,
) *WebDashboardServer {
	fw := l1.NewZTNAFirewall(true)        // Default-Deny por diseño fundacional (Dimensión 1)
	mp := l1.NewMultipathScheduler()       // Multipath Overlay Scheduler (Dimensión 8)
	copilot := l3.NewAICopilotEngine(fw, mp, telemetry) // AI Copilot & Auto-Healing (Dimensión 11)
	wdrr := l1.NewWDRRScheduler(1500, 512)              // Fair Queuing WDRR (1.md Sección 11)
	auditLog := l1.NewMerkleAuditLog()                   // Merkle Audit Log Inalterable (1.md Sección 22)
	semantic := l1.NewSemanticCatalog()                 // Descubrimiento Semántico Pull (1.md Sección 7 y 26)
	darkNode := l1.NewDarkNodeOrchestrator(fw, router, pacer) // Macro-Acción Atómica Modo Invisible (2.md)
	fingerprinter := l2.NewAnomalyFingerprinter()             // Telemetría Agregada por Huella Digital (2.md y genesis.md)
	kuzuEngine := l3.NewKuzuGraphEngine()                     // Motor y Esquema Unificado KùzuDB PFO (genesis.md L178 y 2.md Sec 2)
	hybridKeys, _ := l1.GenerateHybridKeyPair(id.DID())       // Criptografía Híbrida PQC ML-DSA / ML-KEM (Fase 2)
	sphinxRouter := l1.NewSphinxRouter(hybridKeys)            // Enrutador Cebolla Sphinx 3-hop 1280B (Fase 2)
	fastPath := l1.NewXDPFastPathEngine("eth0")                // Aceleración eBPF/XDP Fast-Path en Kernel (Fase 4)
	socks5Gateway := l1.NewSOCKS5Gateway("0.0.0.0:10807", sphinxRouter) // Gateway Universal SOCKS5 RFC 1928 (D:\David)
	go func() {
		_ = socks5Gateway.Start()
	}()
	silentDisp := l1.NewSilentDispatcher(id.DID())            // Red Silenciosa Unicast Cero Broadcast (IPv7-HRS)
	silentDisp.RegisterPeer(id.DID(), "127.0.0.1:7777", true)
	guardian := l2.NewResilienceGuardian(l2.DefaultGuardianConfig()) // Guardián de Resiliencia ASA Nexus (D:\David)

	// Subsistemas Rescatados de G:\Mi unidad\03_Programacion (UIN, Memory Arbiter, Anti-Replay, Hierarchy)
	memArbiter := l1.NewGlobalMemoryArbiter(l1.DefaultTotalMemoryLimit)
	uinManager, _ := l1.NewUINIdentityManager(l1.IdentityModeHybrid)
	antiReplay := l1.NewAntiReplayFilter(memArbiter)
	hierarchy := l1.NewNodeHierarchyManager(id.DID(), l1.NodeClassMeshNode)

	// Delegar un BindingRecord de demostración para el agente de IA local
	_, _ = uinManager.IssueBinding(hybridKeys.ClassicalSignPub, l1.BindingAlgoEd25519, l1.BindingScopeAIAgent, 720*time.Hour, [16]byte{})

	kuzuEngine.UpsertPeer(id.DID(), hybridKeys.MLDSAPubHex, "router_local", false)
	fastPath.SyncRouteFromKuzu(id.DID(), "127.0.0.1", 0, 0.05, 1)
	_, _ = semantic.SignAndRegister(id, []string{"geo/lan", "role/router", "service/ipvn7"}, "Nodo Base ipvn7", 100.0)

	dagStore := l1.NewDAGStore(id)

	return &WebDashboardServer{
		Port:          port,
		Identity:      id,
		Router:        router,
		Telemetry:     telemetry,
		Pacer:         pacer,
		SvcManager:    svcMgr,
		Firewall:      fw,
		BufferPool:    l1.NewBufferPool(),       // Zero-Copy 3-tier pool (Dimensión 12)
		QoS:           l1.NewQoSManager(),       // Token Bucket & Anti-DDoS PoW (Dimensión 3)
		DAGStore:      dagStore,                 // Almacén DAG y DTN Store-and-Forward (Dimensión 4)
		Accounting:    l2.NewTransitAccounting(), // Economía Tit-for-Tat (Dimensión 5)
		WoT:           l1.NewWebOfTrust(),       // Web-of-Trust y Reputación Atenuada (Dimensión 9)
		Petnames:      NewPetnameResolver(),     // dDNS Petnames locales (Dimensión 2)
		SmartPackets:  l1.NewSmartPacketPipeline(), // Smart Packets con sandbox Wazero (Dimensión 7)
		Multipath:     mp,
		Copilot:       copilot,
		WDRR:          wdrr,
		AuditLog:      auditLog,
		Semantic:      semantic,
		DarkNode:      darkNode,
		Fingerprinter: fingerprinter,
		Kuzu:          kuzuEngine,
		HybridKeys:    hybridKeys,
		Sphinx:        sphinxRouter,
		FastPath:      fastPath,
		SOCKS5:        socks5Gateway,
		Silent:        silentDisp,
		Guardian:      guardian,
		MemoryArbiter: memArbiter,
		UIN:           uinManager,
		AntiReplay:    antiReplay,
		Hierarchy:     hierarchy,
		Chat:          NewChatManager(id, dagStore),
		Multicast:     l1.NewCascadeMulticastEngine(id, router),
		RemoteDesktop: NewRemoteDesktopManager(id, fw),
		CorporateVPN:  l1.NewCorporateVPNManager(id, l1.DefaultCorporateVPNConfig(), nil, socks5Gateway),
		LiveStream:    NewLiveStreamManager(id),
		AudioRadio:    NewAudioRadioManager(id),
		Probe:         l1.NewNetworkProbeEngine(id, router),
		Blackout:      l1.NewBlackoutRecoveryManager(id, l1.NewBlindRendezvousManager(id, l1.NewHybridBlindBeaconStore(""), "ipvn7-sovereign-v0.4")),
		CopilotMode:   "off",
		StartTime:     time.Now(),
		StaticDir:     staticDir,
	}
}

// Start inicia el servidor HTTP en segundo plano
func (ws *WebDashboardServer) Start() error {
	mux := http.NewServeMux()

	// 1. API Endpoints
	mux.HandleFunc("/api/status", ws.handleStatus)
	mux.HandleFunc("/api/services/toggle", ws.handleServiceToggle)
	mux.HandleFunc("/api/alias", ws.handleAlias)
	mux.HandleFunc("/api/peers/add", ws.handleAddPeer)
	mux.HandleFunc("/api/tasks/distributed", ws.handleDistributedTask)

	// Endpoints Fase A: ZTNA Firewall, QoS y BufferPool
	mux.HandleFunc("/api/firewall", ws.handleFirewall)
	mux.HandleFunc("/api/firewall/rules", ws.handleFirewallRules)
	mux.HandleFunc("/api/firewall/toggle", ws.handleFirewallToggle)
	mux.HandleFunc("/api/qos", ws.handleQoS)
	mux.HandleFunc("/api/qos/pow/solve", ws.handlePoWSolve)
	mux.HandleFunc("/api/pool", ws.handlePool)

	// Endpoints Fase B: DAG Store, Tit-for-Tat Accounting y Web-of-Trust
	mux.HandleFunc("/api/dag/blocks", ws.handleDAGBlocks)
	mux.HandleFunc("/api/dag/pending", ws.handleDAGPending)
	mux.HandleFunc("/api/dag/ingest", ws.handleDAGIngest)
	mux.HandleFunc("/api/accounting", ws.handleAccounting)
	mux.HandleFunc("/api/wot/reputation", ws.handleWoTReputation)
	mux.HandleFunc("/api/wot/vouch", ws.handleWoTVouch)
	mux.HandleFunc("/api/wot/vouches", ws.handleWoTVouches)

	// Endpoints Fase C: dDNS Petnames, Smart Packets, Multipath y OOB Pairing
	mux.HandleFunc("/api/ddns/names", ws.handlePetnames)
	mux.HandleFunc("/api/ddns/resolve", ws.handlePetnameResolve)
	mux.HandleFunc("/api/ddns/hosts", ws.handleHostsExport)
	mux.HandleFunc("/api/smart/stats", ws.handleSmartStats)
	mux.HandleFunc("/api/smart/inspect", ws.handleSmartInspect)
	mux.HandleFunc("/api/multipath/interfaces", ws.handleMultipathInterfaces)
	mux.HandleFunc("/api/multipath/select", ws.handleMultipathSelect)
	mux.HandleFunc("/api/oob/sas", ws.handleOOBSAS)
	mux.HandleFunc("/api/oob/profile", ws.handleOOBProfile)

	// Endpoints Fase D: AI Copilot & Autonomous Self-Healing (Dimensión 11)
	mux.HandleFunc("/api/copilot/diagnose", ws.handleCopilotDiagnose)
	mux.HandleFunc("/api/copilot/history", ws.handleCopilotHistory)
	mux.HandleFunc("/api/copilot/stats", ws.handleCopilotStats)

	// Endpoints de Próxima Generación (1.md): WDRR Fair Queuing, Merkle Audit Log & Semantic Catalog
	mux.HandleFunc("/api/wdrr/queues", ws.handleWDRRQueues)
	mux.HandleFunc("/api/audit/logs", ws.handleAuditLogs)
	mux.HandleFunc("/api/semantic/query", ws.handleSemanticQuery)
	mux.HandleFunc("/api/semantic/register", ws.handleSemanticRegister)

	// Endpoints Experiencia de Usuario & Macro-Acciones Atómicas (2.md)
	mux.HandleFunc("/api/macro/dark-node", ws.handleMacroDarkNode)
	mux.HandleFunc("/api/macro/mode", ws.handleMacroMode)
	mux.HandleFunc("/api/telemetry/anomalies/summary", ws.handleAnomaliesSummary)
	mux.HandleFunc("/api/telemetry/anomalies/simulate", ws.handleAnomaliesSimulate)

	// Endpoints KùzuDB Graph Engine & Pipeline Axiomático PFO (Fase 1)
	mux.HandleFunc("/api/kuzu/query", ws.handleKuzuQuery)
	mux.HandleFunc("/api/kuzu/pfo", ws.handleKuzuPFO)
	mux.HandleFunc("/api/kuzu/audit", ws.handleKuzuAudit)

	// Endpoints Criptografía Híbrida PQC & Enrutamiento Cebolla Sphinx (Fase 2)
	mux.HandleFunc("/api/crypto/hybrid/info", ws.handleHybridCryptoInfo)
	mux.HandleFunc("/api/crypto/hybrid/sign-verify", ws.handleHybridSignVerify)
	mux.HandleFunc("/api/onion/circuit/build", ws.handleOnionCircuitBuild)
	mux.HandleFunc("/api/onion/circuit/peel", ws.handleOnionCircuitPeel)

	// Endpoints eBPF/XDP Fast-Path & Sincronización de Kernel (Fase 4)
	mux.HandleFunc("/api/ebpf/status", ws.handleEBPFStatus)
	mux.HandleFunc("/api/ebpf/sync", ws.handleEBPFSync)

	// Endpoints Integración D:\David (SOCKS5, STUN NAT, Silent Network, Resilience Guardian)
	mux.HandleFunc("/api/proxy/socks5", ws.handleSOCKS5Status)
	mux.HandleFunc("/api/nat/stun", ws.handleNATStun)
	mux.HandleFunc("/api/guardian/status", ws.handleGuardianStatus)
	mux.HandleFunc("/api/silent/status", ws.handleSilentStatus)
	mux.HandleFunc("/api/silent/call", ws.handleSilentCall)

	// Endpoints Integración G:\Mi unidad\03_Programacion (UIN, Memory Arbiter, Anti-Replay, Hierarchy)
	mux.HandleFunc("/api/uin/passport", ws.handleUINPassport)
	mux.HandleFunc("/api/memory/arbiter", ws.handleMemoryArbiter)
	mux.HandleFunc("/api/uin/binding/create", ws.handleUINBindingCreate)
	mux.HandleFunc("/api/antireplay/verify", ws.handleAntiReplayVerify)
	mux.HandleFunc("/api/hierarchy/snapshot", ws.handleHierarchySnapshot)

	// Endpoints Aplicaciones Soberanas (Chat E2EE, Remote Desktop, DAG Download, Multicast)
	mux.HandleFunc("/api/chat/send", ws.handleChatSend)
	mux.HandleFunc("/api/chat/history", ws.handleChatHistory)
	mux.HandleFunc("/api/chat/contacts", ws.handleChatContacts)
	mux.HandleFunc("/api/dag/download", ws.handleDAGDownload)
	mux.HandleFunc("/api/remote/status", ws.handleRemoteStatus)
	mux.HandleFunc("/api/remote/input", ws.handleRemoteInput)
	mux.HandleFunc("/api/multicast/broadcast", ws.handleMulticastBroadcast)
	mux.HandleFunc("/api/multicast/tree", ws.handleMulticastTree)

	// Endpoints VPN Corporativa Fricción Cero para Multinacionales
	mux.HandleFunc("/api/vpn/corporate/status", ws.handleCorporateVPNStatus)
	mux.HandleFunc("/api/vpn/corporate/toggle", ws.handleCorporateVPNToggle)
	mux.HandleFunc("/api/vpn/corporate/mode", ws.handleCorporateVPNMode)
	mux.HandleFunc("/api/vpn/corporate/egress", ws.handleCorporateVPNEgress)
	mux.HandleFunc("/api/vpn/corporate/ztna/evaluate", ws.handleCorporateVPNZTNAEvaluate)

	// Endpoints Streaming Real en Vivo (WHIP/WHEP P2P)
	mux.HandleFunc("/api/stream/channels", ws.handleStreamChannels)
	mux.HandleFunc("/api/stream/publish", ws.handleStreamPublish)
	mux.HandleFunc("/api/stream/live", ws.handleStreamLive)

	// Endpoints Música & Radio Soberana sin Video
	mux.HandleFunc("/api/audio/stations", ws.handleAudioStations)
	mux.HandleFunc("/api/audio/stream", ws.handleAudioStream)
	mux.HandleFunc("/api/audio/broadcast", ws.handleAudioBroadcast)

	// Endpoints Sonda Voluntaria de Red & Detección de Censura
	mux.HandleFunc("/api/probe/report", ws.handleProbeReport)
	mux.HandleFunc("/api/probe/toggle", ws.handleProbeToggle)

	// Endpoint Protocolo de Recuperación en Cascada Post-Apagón (5 Fases Reales)
	mux.HandleFunc("/api/mesh/blackout/cascade", ws.handleBlackoutCascade)

	// Endpoint Control de Gobernanza de IA Opcional (Zero Token Drain)
	mux.HandleFunc("/api/copilot/mode", ws.handleCopilotMode)

	// 2. Archivos estáticos de interfaz gráfica
	fs := http.FileServer(http.Dir(ws.StaticDir))
	mux.Handle("/", fs)

	addr := fmt.Sprintf("0.0.0.0:%d", ws.Port)
	ws.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		_ = ws.server.ListenAndServe()
	}()

	return nil
}

// Stop apaga el servidor HTTP
func (ws *WebDashboardServer) Stop() error {
	if ws.SOCKS5 != nil {
		ws.SOCKS5.Stop()
	}
	if ws.server != nil {
		return ws.server.Close()
	}
	return nil
}

func (ws *WebDashboardServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	peers := ws.Router.GetAllPeers()
	type peerDTO struct {
		DID         string  `json:"did"`
		Ring        int     `json:"ring"`
		LatencyMs   float64 `json:"latency_ms"`
		HealthState int     `json:"health_state"`
		HealthScore float64 `json:"health_score"`
	}
	pList := make([]peerDTO, 0, len(peers))
	for _, p := range peers {
		pList = append(pList, peerDTO{
			DID:         p.DID,
			Ring:        p.RingIndex,
			LatencyMs:   p.Locator.LatencyMs,
			HealthState: p.HealthState,
			HealthScore: p.HealthScore,
		})
	}

	resp := map[string]interface{}{
		"did":          ws.Identity.DID(),
		"sovereign_v6": ws.Identity.IPv6().String(),
		"virtual_v4":   ws.Identity.IPv4().String(),
		"uptime_sec":   time.Since(ws.StartTime).Seconds(),
		"telemetry":    ws.Telemetry.Snapshot(),
		"pacer":        ws.Pacer.Stats(),
		"firewall":     ws.Firewall.Stats(),
		"qos":          ws.QoS.Stats(),
		"pool":         ws.BufferPool.Stats(),
		"dag":          ws.DAGStore.Stats(),
		"accounting":   ws.Accounting.Stats(),
		"wot":          ws.WoT.Stats(),
		"ddns":         ws.Petnames.Stats(),
		"smart":        ws.SmartPackets.Stats(),
		"multipath":    ws.Multipath.Stats(),
		"copilot":       ws.Copilot.Stats(),
		"wdrr":          ws.WDRR.GetMetrics(),
		"audit_entries": ws.AuditLog.TotalEntries(),
		"semantic":      ws.Semantic.Stats(),
		"dark_node":      ws.DarkNode.GetStatus(),
		"anomalies":     ws.Fingerprinter.GetExecutiveSummary(),
		"kuzu":          ws.Kuzu.GetPFOTree(),
		"pqc": map[string]interface{}{
			"algorithm_sig": l1.HybridSigAlgorithm,
			"algorithm_kem": l1.HybridKEMAlgorithm,
			"ed25519_pub":   ws.HybridKeys.Ed25519PubHex,
			"x25519_pub":    ws.HybridKeys.X25519PubHex,
			"ml_dsa_pub":    ws.HybridKeys.MLDSAPubHex,
			"ml_kem_pub":    ws.HybridKeys.MLKEMPubHex,
			"quantum_ready": true,
		},
		"sphinx": map[string]interface{}{
			"fixed_frame_bytes": l1.SphinxPacketSize,
			"hops_count":        l1.SphinxHopsCount,
			"payload_capacity":  l1.SphinxPayloadSize,
			"deterministic_mtu": 1280,
		},
		"ebpf":          ws.FastPath.GetStatus(),
		"socks5":        ws.SOCKS5.GetStats(),
		"guardian":      ws.Guardian.GetHealthStatus(),
		"silent":        ws.Silent.GetStats(),
		"uin":            ws.UIN.GetPassport(),
		"memory_arbiter": ws.MemoryArbiter.GetStats(),
		"anti_replay":    ws.AntiReplay.GetStats(),
		"hierarchy":      ws.Hierarchy.GetHierarchySnapshot(),
		"peers":         pList,
		"services":      ws.SvcManager.GetActiveServices(),
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (ws *WebDashboardServer) handleServiceToggle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name  string `json:"name"`
		Scope uint8  `json:"scope"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	// Verificar si ya está activo
	activeList := ws.SvcManager.GetActiveServices()
	isActive := false
	for _, s := range activeList {
		if s.Name == req.Name {
			isActive = true
			break
		}
	}

	if isActive {
		ws.SvcManager.ReleaseService(req.Name)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"name": req.Name, "active": false})
	} else {
		_, err := ws.SvcManager.RequestService(req.Name, req.Scope)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"name": req.Name, "active": true})
	}
}

func (ws *WebDashboardServer) handleAlias(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Name        string `json:"name"`
		DID         string `json:"did"`
		ContextRoot string `json:"context_root"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	var assignedName string
	if req.ContextRoot != "" {
		h := sha256.Sum256([]byte(req.ContextRoot))
		prefix := hex.EncodeToString(h[:4])
		assignedName = fmt.Sprintf("%s.%s", prefix, strings.TrimPrefix(req.Name, prefix+"."))
	} else {
		assignedName = req.Name
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"assigned_name": assignedName,
		"did":           req.DID,
		"status":        "REGISTERED",
	})
}

// FindStaticDir localiza el directorio web relativo al ejecutable o al directorio de trabajo
func FindStaticDir() string {
	candidates := []string{
		"web",
		filepath.Join("..", "web"),
		filepath.Join(".", "web"),
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && fi.IsDir() {
			return c
		}
	}
	return "web"
}

func (ws *WebDashboardServer) handleAddPeer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DID       string  `json:"did"`
		Address   string  `json:"address"`
		LatencyMs float64 `json:"latency_ms"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	udpAddr, err := net.ResolveUDPAddr("udp", req.Address)
	if err != nil {
		http.Error(w, fmt.Sprintf("Dirección UDP inválida: %v", err), http.StatusBadRequest)
		return
	}

	lat := req.LatencyMs
	if lat <= 0 {
		lat = 2.5 // default local LAN
	}

	if err := ws.Router.AddOrUpdatePeer(req.DID, udpAddr, lat); err != nil {
		http.Error(w, fmt.Sprintf("Error al registrar par: %v", err), http.StatusInternalServerError)
		return
	}

	// Actualizar salud a HEALTHY
	ws.Router.UpdatePeerHealth(req.DID, 1.0, 0.4, 0.0)

	// Sincronizar topología viva en Kùzu Graph DB (2.md Línea 761)
	ws.Kuzu.UpsertPeer(req.DID, "", "kleinberg_peer", false)
	ws.Kuzu.UpsertXORLink(ws.Identity.DID(), req.DID, "xor_metric", lat, 0, "LAN-UDP")

	// Registrar evento de telemetría de descubrimiento/emparejamiento
	ws.Telemetry.RecordEvent(l2.EventTxPacket, 128, uint32(lat*1000), 0)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "CONNECTED",
		"did":        req.DID,
		"address":    req.Address,
		"latency_ms": lat,
	})
}

func (ws *WebDashboardServer) handleDistributedTask(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TargetURL string `json:"target_url"`
		TaskType  string `json:"task_type"`
		Payload   string `json:"payload"`
		SourceDID string `json:"source_did"`
		Port      uint16 `json:"port"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	srcDID := req.SourceDID
	if srcDID == "" {
		srcDID = ws.Identity.DID()
	}
	port := req.Port
	if port == 0 {
		port = 7001
	}

	// 1. Dimensión 1: Evaluación estricta de Cortafuegos Zero Trust (ZTNA)
	decision, reason := ws.Firewall.EvaluateInbound(srcDID, port)
	if decision != l1.DecisionAccept {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "BLOCKED_BY_ZTNA",
			"decision": decision.String(),
			"reason":   reason,
			"src_did":  srcDID,
			"port":     port,
		})
		return
	}

	// 2. Dimensión 3: Evaluación de Calidad de Servicio (QoS) y Desafío PoW Anti-DDoS
	payloadBytes := []byte(req.Payload)
	allowed, powChallenge := ws.QoS.EvaluatePacket(srcDID, l1.ClassInteractive, len(payloadBytes))
	if !allowed {
		w.WriteHeader(http.StatusTooManyRequests)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":        "THROTTLED_BY_QOS",
			"message":       "Límite de tasa excedido. Resuelva el desafío PoW para restaurar créditos.",
			"pow_challenge": powChallenge,
		})
		return
	}

	// 3. Dimensión 12: Asignación y reciclaje en Buffer Pool Zero-Copy
	pktBuf := ws.BufferPool.Acquire(len(payloadBytes) + 64)
	copy(pktBuf.RawSlice(), payloadBytes)
	defer pktBuf.Release() // Garantiza liberación sin presión al Garbage Collector

	start := time.Now()
	// Tarea distribuida: consultar status del nodo remoto y verificar coherencia criptográfica
	targetEndpoint := strings.TrimRight(req.TargetURL, "/") + "/api/status"
	client := http.Client{Timeout: 5 * time.Second}
	respRemote, err := client.Get(targetEndpoint)
	if err != nil {
		http.Error(w, fmt.Sprintf("Fallo al contactar nodo remoto: %v", err), http.StatusBadGateway)
		return
	}
	defer respRemote.Body.Close()

	var remoteStatus map[string]interface{}
	if err := json.NewDecoder(respRemote.Body).Decode(&remoteStatus); err != nil {
		http.Error(w, fmt.Sprintf("Respuesta remota corrupta: %v", err), http.StatusBadGateway)
		return
	}

	rttMs := float64(time.Since(start).Microseconds()) / 1000.0

	// Simulación determinista de cómputo con prueba hash PFO
	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%v", ws.Identity.DID(), req.Payload, remoteStatus["did"])))
	taskProof := hex.EncodeToString(h[:])

	// Registrar métricas en telemetría L2
	ws.Telemetry.RecordEvent(l2.EventTxPacket, uint32(len(req.Payload)+64), uint32(rttMs*1000), 0)
	ws.Telemetry.RecordEvent(l2.EventRxPacket, 256, uint32(rttMs*1000), 0)

	remoteDIDStr := fmt.Sprintf("%v", remoteStatus["did"])

	// Dimensión 5: Registrar contabilidad Tit-for-Tat
	ws.Accounting.RecordTx(remoteDIDStr, uint64(len(req.Payload)+64))
	ws.Accounting.RecordRx(remoteDIDStr, 256)
	peerTier := ws.Accounting.GetPeerTier(remoteDIDStr)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "COMPLETED",
		"task_type":       req.TaskType,
		"local_did":       ws.Identity.DID(),
		"remote_did":      remoteStatus["did"],
		"round_trip_ms":   rttMs,
		"task_proof_hash": taskProof,
		"pacer_effective": ws.Pacer.Stats().EffectiveRateBytesPerSec,
		"ztna_decision":   decision.String(),
		"peer_tier":       peerTier.String(),
		"buffer_recycled": true,
		"timestamp":       time.Now().UTC().Format(time.RFC3339Nano),
	})
}

func (ws *WebDashboardServer) handleFirewall(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	resp := map[string]interface{}{
		"stats":    ws.Firewall.Stats(),
		"policies": ws.Firewall.GetAllPolicies(),
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (ws *WebDashboardServer) handleFirewallRules(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodPost {
		var p l1.DIDPolicy
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		ws.Firewall.AuthorizeDID(&p)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "AUTHORIZED", "policy": p})
	} else if r.Method == http.MethodDelete {
		did := r.URL.Query().Get("did")
		if did == "" {
			http.Error(w, "Parámetro 'did' requerido", http.StatusBadRequest)
			return
		}
		ws.Firewall.RevokeDID(did)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "REVOKED", "did": did})
	} else {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (ws *WebDashboardServer) handleFirewallToggle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		DefaultDeny bool `json:"default_deny"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	ws.Firewall.SetDefaultDeny(req.DefaultDeny)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"default_deny": ws.Firewall.IsDefaultDeny(),
	})
}

func (ws *WebDashboardServer) handleQoS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_ = json.NewEncoder(w).Encode(ws.QoS.Stats())
}

func (ws *WebDashboardServer) handlePoWSolve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Challenge string `json:"challenge"`
		Nonce     uint64 `json:"nonce"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	success := ws.QoS.VerifyAndCreditPoW(req.Challenge, req.Nonce)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"verified": success,
		"message":  "Créditos de token bucket repuestos con éxito",
	})
}

func (ws *WebDashboardServer) handlePool(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_ = json.NewEncoder(w).Encode(ws.BufferPool.Stats())
}

func (ws *WebDashboardServer) handleDAGBlocks(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		cid := r.URL.Query().Get("cid")
		if cid != "" {
			if b, ok := ws.DAGStore.GetBlock(cid); ok {
				_ = json.NewEncoder(w).Encode(b)
				return
			}
			http.Error(w, "Bloque no encontrado", http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(ws.DAGStore.ListBlocks(50))
	} else if r.Method == http.MethodPost {
		var req struct {
			Data      string   `json:"data"`
			Parents   []string `json:"parents"`
			TargetDID string   `json:"target_did"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		block, err := ws.DAGStore.PutBlock([]byte(req.Data), req.Parents, req.TargetDID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(block)
	} else {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (ws *WebDashboardServer) handleDAGPending(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "Parámetro 'target' requerido", http.StatusBadRequest)
		return
	}

	if r.Method == http.MethodGet {
		_ = json.NewEncoder(w).Encode(ws.DAGStore.GetPendingQueue(target))
	} else if r.Method == http.MethodPost {
		ws.DAGStore.ClearPending(target)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "CLEARED", "target": target})
	}
}

func (ws *WebDashboardServer) handleDAGIngest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var block l1.DAGBlock
	if err := json.NewDecoder(r.Body).Decode(&block); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if err := ws.DAGStore.IngestRemoteBlock(&block); err != nil {
		http.Error(w, fmt.Sprintf("Fallo al ingerir bloque DAG: %v", err), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "INGESTED",
		"cid":    block.CID,
	})
}

func (ws *WebDashboardServer) handleAccounting(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	resp := map[string]interface{}{
		"stats":    ws.Accounting.Stats(),
		"accounts": ws.Accounting.GetAllAccounts(),
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func (ws *WebDashboardServer) handleWoTReputation(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	target := r.URL.Query().Get("target")
	if target == "" {
		http.Error(w, "Parámetro 'target' requerido", http.StatusBadRequest)
		return
	}

	score, hops := ws.WoT.CalculateReputation(ws.Identity.DID(), target)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"root_did":         ws.Identity.DID(),
		"target_did":       target,
		"reputation_score": score,
		"hops":             hops,
	})
}

func (ws *WebDashboardServer) handleWoTVouch(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SubjectDID string  `json:"subject_did"`
		TrustLevel float64 `json:"trust_level"`
		Reason     string  `json:"reason"`
		Hours      int     `json:"hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	dur := 24 * time.Hour
	if req.Hours > 0 {
		dur = time.Duration(req.Hours) * time.Hour
	}

	vouch, err := ws.WoT.SignAndIssueVouch(ws.Identity, req.SubjectDID, req.TrustLevel, req.Reason, dur)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error al emitir aval: %v", err), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(vouch)
}

func (ws *WebDashboardServer) handleWoTVouches(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"stats":   ws.WoT.Stats(),
		"vouches": ws.WoT.GetAllVouches(),
	})
}

func (ws *WebDashboardServer) handlePetnames(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodGet {
		_ = json.NewEncoder(w).Encode(ws.Petnames.ListRecords())
	} else if r.Method == http.MethodPost {
		var req struct {
			Name        string `json:"name"`
			DID         string `json:"did"`
			VirtualIPv6 string `json:"virtual_ipv6"`
			VirtualIPv4 string `json:"virtual_ipv4"`
			ContextRoot string `json:"context_root"`
			Comment     string `json:"comment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "JSON inválido", http.StatusBadRequest)
			return
		}
		rec, err := ws.Petnames.RegisterPetname(req.Name, req.DID, req.VirtualIPv6, req.VirtualIPv4, req.ContextRoot, req.Comment)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(rec)
	} else {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func (ws *WebDashboardServer) handlePetnameResolve(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	name := r.URL.Query().Get("name")
	ctxRoot := r.URL.Query().Get("context")
	if name == "" {
		http.Error(w, "Parámetro 'name' requerido", http.StatusBadRequest)
		return
	}

	if rec, ok := ws.Petnames.Resolve(name, ctxRoot); ok {
		_ = json.NewEncoder(w).Encode(rec)
		return
	}
	http.Error(w, "Nombre no encontrado en la libreta dDNS local", http.StatusNotFound)
}

func (ws *WebDashboardServer) handleHostsExport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_, _ = w.Write([]byte(ws.Petnames.ExportHostsFormat()))
}

func (ws *WebDashboardServer) handleSmartStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_ = json.NewEncoder(w).Encode(ws.SmartPackets.Stats())
}

func (ws *WebDashboardServer) handleSmartInspect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Payload string `json:"payload"`
		Point   int    `json:"point"` // 0: Ingress, 1: Forwarding, 2: Egress
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	pt := l1.HookPoint(req.Point)
	outBytes, pass, reason := ws.SmartPackets.Process(pt, []byte(req.Payload))

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"pass":           pass,
		"reason":         reason,
		"output_payload": string(outBytes),
		"hook_point":     pt.String(),
	})
}

func (ws *WebDashboardServer) handleMultipathInterfaces(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"stats":      ws.Multipath.Stats(),
		"interfaces": ws.Multipath.GetAllInterfaces(),
	})
}

func (ws *WebDashboardServer) handleMultipathSelect(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Strategy int `json:"strategy"` // 0: LowestLatency, 1: Balanced, 2: CriticalDuplication
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	strat := l1.MultipathStrategy(req.Strategy)
	paths, err := ws.Multipath.SelectPaths(strat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"strategy":       strat.String(),
		"selected_paths": paths,
	})
}

func (ws *WebDashboardServer) handleOOBSAS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	peerDID := r.URL.Query().Get("peer_did")
	if peerDID == "" {
		http.Error(w, "Parámetro 'peer_did' requerido", http.StatusBadRequest)
		return
	}

	peerPub, err := l0.PublicKeyFromDID(peerDID)
	if err != nil {
		http.Error(w, fmt.Sprintf("DID inválido: %v", err), http.StatusBadRequest)
		return
	}

	sas := DeriveSAS(ws.Identity.PublicKey, peerPub)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"local_did": ws.Identity.DID(),
		"peer_did":  peerDID,
		"sas_code":  sas,
	})
}

func (ws *WebDashboardServer) handleOOBProfile(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	profileJSON := fmt.Sprintf(`{"did":"%s","ipv6":"%s","ipv4":"%s","p2p_port":%d}`,
		ws.Identity.DID(), ws.Identity.IPv6(), ws.Identity.IPv4(), ws.Port)

	chunks := GenerateURChunks([]byte(profileJSON), 48)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"total_frames": len(chunks),
		"ur_frames":    chunks,
	})
}

// Handlers Fase D: AI Copilot & Autonomous Self-Healing (Dimensión 11)

func (ws *WebDashboardServer) handleCopilotDiagnose(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	recentLat := 55.0
	dropsCount := uint64(7)
	peerDID := ""

	if r.Method == http.MethodPost {
		var req struct {
			RecentLatencyMs float64 `json:"recent_latency_ms"`
			DropsCount      uint64  `json:"drops_count"`
			PeerDID         string  `json:"peer_did"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
			if req.RecentLatencyMs > 0 {
				recentLat = req.RecentLatencyMs
			}
			if req.DropsCount > 0 {
				dropsCount = req.DropsCount
			}
			peerDID = req.PeerDID
		}
	}

	anomalies := ws.Copilot.DetectAnomalies(recentLat, dropsCount)
	diagnoses := make([]*l3.CopilotDiagnosis, 0)
	for _, an := range anomalies {
		if peerDID != "" && an.PeerDID == "" {
			an.PeerDID = peerDID
		}
		diag, err := ws.Copilot.DiagnoseAndHeal(r.Context(), an)
		if err == nil {
			diagnoses = append(diagnoses, diag)
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":             "diagnosed",
		"anomalies_detected": len(anomalies),
		"anomalies":          anomalies,
		"diagnoses":          diagnoses,
		"copilot_stats":      ws.Copilot.Stats(),
	})
}

func (ws *WebDashboardServer) handleCopilotHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	history := ws.Copilot.GetHistory(30)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"total":   len(history),
		"history": history,
	})
}

func (ws *WebDashboardServer) handleCopilotStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_ = json.NewEncoder(w).Encode(ws.Copilot.Stats())
}

// Handlers de Próxima Generación: WDRR, Merkle Audit Log & Semantic Catalog (1.md)

func (ws *WebDashboardServer) handleWDRRQueues(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	metrics := ws.WDRR.GetMetrics()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"queues":         metrics,
		"total_enqueued": atomic.LoadUint64(&ws.WDRR.TotalEnqueued),
		"total_dequeued": atomic.LoadUint64(&ws.WDRR.TotalDequeued),
		"total_dropped":  atomic.LoadUint64(&ws.WDRR.TotalDropped),
	})
}

func (ws *WebDashboardServer) handleAuditLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	verified, err := ws.AuditLog.VerifyIntegrity()
	entries := ws.AuditLog.GetRecentEntries(50)

	errStr := ""
	if err != nil {
		errStr = err.Error()
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"merkle_verified": verified,
		"integrity_error": errStr,
		"total_entries":   ws.AuditLog.TotalEntries(),
		"entries":         entries,
	})
}

func (ws *WebDashboardServer) handleSemanticQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var tags []string
	if r.Method == http.MethodPost && r.Body != nil {
		var req struct {
			Tags []string `json:"tags"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && len(req.Tags) > 0 {
			tags = req.Tags
		}
	}
	if len(tags) == 0 {
		tagsParam := r.URL.Query().Get("tags")
		if tagsParam != "" {
			tags = strings.Split(tagsParam, ",")
		}
	}

	results := ws.Semantic.QueryPull(tags)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"required_tags": tags,
		"matches_count": len(results),
		"providers":     results,
	})
}

func (ws *WebDashboardServer) handleSemanticRegister(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Tags        []string `json:"tags"`
		Description string   `json:"description"`
		Capacity    float64  `json:"capacity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	prof, err := ws.Semantic.SignAndRegister(ws.Identity, req.Tags, req.Description, req.Capacity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "REGISTERED",
		"profile": prof,
	})
}

// Handlers Experiencia de Usuario & Macro-Acciones Atómicas (2.md)

func (ws *WebDashboardServer) handleMacroDarkNode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	mode, err := ws.DarkNode.ToggleMode()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"toggled": true,
		"mode":    string(mode),
		"status":  ws.DarkNode.GetStatus(),
	})
}

func (ws *WebDashboardServer) handleMacroMode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_ = json.NewEncoder(w).Encode(ws.DarkNode.GetStatus())
}

func (ws *WebDashboardServer) handleAnomaliesSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_ = json.NewEncoder(w).Encode(ws.Fingerprinter.GetExecutiveSummary())
}

func (ws *WebDashboardServer) handleAnomaliesSimulate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		Type      string `json:"type"`
		Subsystem string `json:"subsystem"`
		RootCause string `json:"root_cause"`
		Severity  string `json:"severity"`
		Count     uint64 `json:"count"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}
	if req.Count == 0 {
		req.Count = 1
	}
	sev := l2.SeverityLevel(req.Severity)
	if sev == "" {
		sev = l2.SeverityMedium
	}
	fp := ws.Fingerprinter.RecordBatch(req.Type, req.Subsystem, req.RootCause, sev, req.Count)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"recorded":    true,
		"fingerprint": fp,
		"summary":     ws.Fingerprinter.GetExecutiveSummary(),
	})
}

// Handlers KùzuDB Graph Engine & Pipeline Axiomático PFO (Fase 1)

func (ws *WebDashboardServer) handleKuzuQuery(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var query string
	if r.Method == http.MethodPost && r.Body != nil {
		var req struct {
			Query string `json:"query"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		query = req.Query
	}
	if query == "" {
		query = r.URL.Query().Get("q")
	}
	if query == "" {
		query = "MATCH (p:Peer)-[l:XOR_LINK]->(m:Peer) RETURN p.did, l.degree, l.latency_ms, l.rf_band, m.did LIMIT 100"
	}

	result, err := ws.Kuzu.ExecuteCypher(query)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"error": err.Error(),
			"query": query,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(result)
}

func (ws *WebDashboardServer) handleKuzuPFO(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_ = json.NewEncoder(w).Encode(ws.Kuzu.GetPFOTree())
}

func (ws *WebDashboardServer) handleKuzuAudit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		Directive string `json:"directive"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	collides, msg := ws.Kuzu.AuditDirectiveContradiction(req.Directive)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"directive": req.Directive,
		"collides":  collides,
		"verdict":   msg,
	})
}

// Handlers Criptografía Híbrida PQC & Enrutamiento Cebolla Sphinx (Fase 2)

func (ws *WebDashboardServer) handleHybridCryptoInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	resp := map[string]interface{}{
		"did":              ws.Identity.DID(),
		"algorithm_sig":    l1.HybridSigAlgorithm,
		"algorithm_kem":    l1.HybridKEMAlgorithm,
		"ed25519_pub_hex":  ws.HybridKeys.Ed25519PubHex,
		"x25519_pub_hex":   ws.HybridKeys.X25519PubHex,
		"ml_dsa_pub_hex":   ws.HybridKeys.MLDSAPubHex,
		"ml_kem_pub_hex":   ws.HybridKeys.MLKEMPubHex,
		"created_at":       ws.HybridKeys.CreatedAt,
		"quantum_security": "NIST Security Level 3 (ML-DSA-65 / ML-KEM-768)",
		"status":           "ACTIVE_PROTECTED",
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (ws *WebDashboardServer) handleHybridSignVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	msgText := "ipvn7 sovereign wire datagram"
	if r.Method == http.MethodPost && r.Body != nil {
		var req struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Message != "" {
			msgText = req.Message
		}
	}

	startSign := time.Now()
	sig, err := ws.HybridKeys.Sign([]byte(msgText))
	signUs := time.Since(startSign).Microseconds()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error firmando: %v", err), http.StatusInternalServerError)
		return
	}

	startVerify := time.Now()
	valid := ws.HybridKeys.Verify([]byte(msgText), sig)
	verifyUs := time.Since(startVerify).Microseconds()

	resp := map[string]interface{}{
		"message":           msgText,
		"algorithm":         sig.Algorithm,
		"sign_latency_us":   signUs,
		"verify_latency_us": verifyUs,
		"classical_sig_len": len(sig.ClassicalSig),
		"pqc_sig_len":       len(sig.PQCSig),
		"verified":          valid,
		"timestamp":         sig.Timestamp,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (ws *WebDashboardServer) handleOnionCircuitBuild(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Crear 3 saltos demostrativos para el circuito si no hay suficientes pares remotos
	gKeys, _ := l1.GenerateHybridKeyPair("did:ipvn7:guard-node")
	mKeys, _ := l1.GenerateHybridKeyPair("did:ipvn7:middle-node")
	eKeys, _ := l1.GenerateHybridKeyPair("did:ipvn7:exit-node")

	circuit := &l1.SphinxCircuit{
		CircuitID: fmt.Sprintf("circ-%d", time.Now().UnixNano()%100000),
		Guard: &l1.SphinxHopNode{
			DID:          gKeys.DID,
			Address:      "10.7.1.1:7777",
			HybridKeys:   gKeys,
			X25519PubHex: gKeys.X25519PubHex,
			MLKEMPubHex:  gKeys.MLKEMPubHex,
		},
		Middle: &l1.SphinxHopNode{
			DID:          mKeys.DID,
			Address:      "10.7.1.2:7777",
			HybridKeys:   mKeys,
			X25519PubHex: mKeys.X25519PubHex,
			MLKEMPubHex:  mKeys.MLKEMPubHex,
		},
		Exit: &l1.SphinxHopNode{
			DID:          eKeys.DID,
			Address:      "10.7.1.3:7777",
			HybridKeys:   eKeys,
			X25519PubHex: eKeys.X25519PubHex,
			MLKEMPubHex:  eKeys.MLKEMPubHex,
		},
		CreatedAt: time.Now().UTC(),
	}

	payloadMsg := "Sovereign packet routing through 3-hop Sphinx onion network"
	destService := "echo.ipvn7.sovereign"

	startBuild := time.Now()
	packet, err := ws.Sphinx.BuildPacket(circuit, []byte(payloadMsg), destService)
	buildUs := time.Since(startBuild).Microseconds()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error construyendo circuito Sphinx: %v", err), http.StatusInternalServerError)
		return
	}

	wire := packet.Serialize()

	resp := map[string]interface{}{
		"circuit_id":        circuit.CircuitID,
		"wire_bytes":        len(wire),
		"deterministic_mtu": l1.SphinxPacketSize,
		"build_latency_us":  buildUs,
		"hops": []map[string]interface{}{
			{"hop": 0, "role": "GUARD", "did": circuit.Guard.DID, "address": circuit.Guard.Address},
			{"hop": 1, "role": "MIDDLE", "did": circuit.Middle.DID, "address": circuit.Middle.Address},
			{"hop": 2, "role": "EXIT", "did": circuit.Exit.DID, "address": circuit.Exit.Address},
		},
		"payload_original": payloadMsg,
		"dest_service":     destService,
		"zero_leak_padded": true,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (ws *WebDashboardServer) handleOnionCircuitPeel(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 1. Configurar los 3 saltos con sus enrutadores respectivos
	gKeys, _ := l1.GenerateHybridKeyPair("did:ipvn7:guard-node")
	mKeys, _ := l1.GenerateHybridKeyPair("did:ipvn7:middle-node")
	eKeys, _ := l1.GenerateHybridKeyPair("did:ipvn7:exit-node")

	circuit := &l1.SphinxCircuit{
		CircuitID: fmt.Sprintf("circ-%d", time.Now().UnixNano()%100000),
		Guard: &l1.SphinxHopNode{
			DID:          gKeys.DID,
			Address:      "10.7.1.1:7777",
			HybridKeys:   gKeys,
			X25519PubHex: gKeys.X25519PubHex,
			MLKEMPubHex:  gKeys.MLKEMPubHex,
		},
		Middle: &l1.SphinxHopNode{
			DID:          mKeys.DID,
			Address:      "10.7.1.2:7777",
			HybridKeys:   mKeys,
			X25519PubHex: mKeys.X25519PubHex,
			MLKEMPubHex:  mKeys.MLKEMPubHex,
		},
		Exit: &l1.SphinxHopNode{
			DID:          eKeys.DID,
			Address:      "10.7.1.3:7777",
			HybridKeys:   eKeys,
			X25519PubHex: eKeys.X25519PubHex,
			MLKEMPubHex:  eKeys.MLKEMPubHex,
		},
		CreatedAt: time.Now().UTC(),
	}

	payloadMsg := "Mensaje confidencial verificado por desprendimiento iterativo de 3 capas cebolla"
	destService := "secure.chat.ipvn7"

	packet, err := ws.Sphinx.BuildPacket(circuit, []byte(payloadMsg), destService)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error construyendo paquete cebolla: %v", err), http.StatusInternalServerError)
		return
	}

	steps := make([]map[string]interface{}, 0, 3)

	// Salto 1: Guard
	gRouter := l1.NewSphinxRouter(gKeys)
	peel1, err := gRouter.PeelLayer(packet)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error en salto Guard: %v", err), http.StatusInternalServerError)
		return
	}
	steps = append(steps, map[string]interface{}{
		"hop":          1,
		"node_role":    "GUARD",
		"action":       peel1.Action,
		"next_hop":     peel1.NextHopDID,
		"wire_bytes":   len(peel1.NextPacket.Serialize()),
		"peel_time_us": peel1.PeelTimeUs,
		"tag_hash":     peel1.TagHash[:16] + "...",
	})

	// Salto 2: Middle
	mRouter := l1.NewSphinxRouter(mKeys)
	peel2, err := mRouter.PeelLayer(peel1.NextPacket)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error en salto Middle: %v", err), http.StatusInternalServerError)
		return
	}
	steps = append(steps, map[string]interface{}{
		"hop":          2,
		"node_role":    "MIDDLE",
		"action":       peel2.Action,
		"next_hop":     peel2.NextHopDID,
		"wire_bytes":   len(peel2.NextPacket.Serialize()),
		"peel_time_us": peel2.PeelTimeUs,
		"tag_hash":     peel2.TagHash[:16] + "...",
	})

	// Salto 3: Exit
	eRouter := l1.NewSphinxRouter(eKeys)
	peel3, err := eRouter.PeelLayer(peel2.NextPacket)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error en salto Exit: %v", err), http.StatusInternalServerError)
		return
	}
	steps = append(steps, map[string]interface{}{
		"hop":             3,
		"node_role":       "EXIT",
		"action":          peel3.Action,
		"next_hop":        peel3.NextHopDID,
		"is_exit":         peel3.IsExit,
		"peel_time_us":    peel3.PeelTimeUs,
		"tag_hash":        peel3.TagHash[:16] + "...",
		"payload_matches": string(peel3.RawPayload) == payloadMsg,
	})

	resp := map[string]interface{}{
		"circuit_id":       circuit.CircuitID,
		"total_hops":       3,
		"initial_wire":     l1.SphinxPacketSize,
		"delivered":        peel3.Action == l1.SphinxActionDeliver,
		"peeling_steps":    steps,
		"payload_original": payloadMsg,
		"payload_received": string(peel3.RawPayload),
		"success":          string(peel3.RawPayload) == payloadMsg,
	}

	_ = json.NewEncoder(w).Encode(resp)
}

// Handlers eBPF/XDP Fast-Path & Sincronización de Kernel (Fase 4)

func (ws *WebDashboardServer) handleEBPFStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	_ = json.NewEncoder(w).Encode(ws.FastPath.GetStatus())
}

func (ws *WebDashboardServer) handleEBPFSync(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		DID       string  `json:"did"`
		NextIP    string  `json:"next_ip"`
		Degree    int8    `json:"degree"`
		LatencyMs float64 `json:"latency_ms"`
		IfIndex   uint32  `json:"if_index"`
	}

	if r.Method == http.MethodPost && r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	if req.DID == "" {
		req.DID = ws.Identity.DID()
		req.NextIP = "127.0.0.1"
		req.Degree = 0
		req.LatencyMs = 0.05
		req.IfIndex = 1
	}

	ws.FastPath.SyncRouteFromKuzu(req.DID, req.NextIP, req.Degree, req.LatencyMs, req.IfIndex)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "SYNCED_TO_EBPF_MAP",
		"did":        req.DID,
		"next_ip":    req.NextIP,
		"degree":     req.Degree,
		"latency_ms": req.LatencyMs,
		"if_index":   req.IfIndex,
		"map_dump":   ws.FastPath.FormatBPFMapDump(),
	})
}

func (ws *WebDashboardServer) handleSOCKS5Status(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	stats := ws.SOCKS5.GetStats()
	_ = json.NewEncoder(w).Encode(stats)
}

func (ws *WebDashboardServer) handleNATStun(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	serverQuery := r.URL.Query().Get("server")
	var servers []string
	if serverQuery != "" {
		servers = []string{serverQuery}
	} else {
		servers = l1.DefaultSTUNServers
	}

	res, err := l1.ProbeSTUN(servers, 2*time.Second)
	if err != nil {
		// Retornar diagnóstico estructurado en caso de fallback/aislamiento LAN
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"status":      "PROBE_FAILED_OR_LOCAL_LAN",
			"error":       err.Error(),
			"nat_type":    "Symmetric NAT or Filtered UDP",
			"public_ip":   "127.0.0.1",
			"public_port": ws.Port,
			"timestamp":   time.Now().UTC(),
		})
		return
	}

	_ = json.NewEncoder(w).Encode(res)
}

func (ws *WebDashboardServer) handleGuardianStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	status := ws.Guardian.GetHealthStatus()
	_ = json.NewEncoder(w).Encode(status)
}

func (ws *WebDashboardServer) handleSilentStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	stats := ws.Silent.GetStats()
	_ = json.NewEncoder(w).Encode(stats)
}

func (ws *WebDashboardServer) handleSilentCall(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TargetDID string `json:"target_did"`
		Payload   string `json:"payload"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	if req.TargetDID == "" {
		req.TargetDID = ws.Identity.DID()
	}
	if req.Payload == "" {
		req.Payload = "SILENT_SYN"
	}

	resp, err := ws.Silent.ProcessInboundCall(req.TargetDID, []byte(req.Payload))
	if err != nil {
		http.Error(w, err.Error(), http.StatusTooManyRequests)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "SILENT_RESPONSE_OK",
		"target_did": req.TargetDID,
		"response":   string(resp),
		"timestamp":  time.Now().UTC(),
	})
}

func (ws *WebDashboardServer) handleUINPassport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	passport := ws.UIN.GetPassport()
	_ = json.NewEncoder(w).Encode(passport)
}

func (ws *WebDashboardServer) handleMemoryArbiter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	stats := ws.MemoryArbiter.GetStats()
	_ = json.NewEncoder(w).Encode(stats)
}

func (ws *WebDashboardServer) handleUINBindingCreate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		PublicKeyHex string `json:"public_key_hex"`
		Scope        uint8  `json:"scope"`
		DurationDays int    `json:"duration_days"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	pubBytes, err := hex.DecodeString(req.PublicKeyHex)
	if err != nil || len(pubBytes) == 0 {
		pubBytes = ws.HybridKeys.ClassicalSignPub
	}
	if req.DurationDays <= 0 {
		req.DurationDays = 30
	}

	rec, err := ws.UIN.IssueBinding(pubBytes, l1.BindingAlgoEd25519, req.Scope, time.Duration(req.DurationDays)*24*time.Hour, [16]byte{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "BINDING_ISSUED",
		"key_id_hex": hex.EncodeToString(rec.KeyID[:]),
		"scope":      rec.Scope,
		"valid_from": rec.ValidFrom,
		"valid_until": rec.ValidUntil,
		"sig_root_hex": hex.EncodeToString(rec.SigRoot),
	})
}

func (ws *WebDashboardServer) handleAntiReplayVerify(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		OriginDID string `json:"origin_did"`
		SessionID uint64 `json:"session_id"`
		Sequence  uint64 `json:"sequence"`
		Timestamp int64  `json:"timestamp"`
	}

	if r.Method == http.MethodPost && r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	if req.OriginDID == "" {
		req.OriginDID = ws.Identity.DID()
	}
	if req.SessionID == 0 {
		req.SessionID = 1001
	}
	if req.Timestamp == 0 {
		req.Timestamp = time.Now().Unix()
	}

	accepted := ws.AntiReplay.Accept(req.OriginDID, req.SessionID, req.Sequence, req.Timestamp)

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"origin_did": req.OriginDID,
		"session_id": req.SessionID,
		"sequence":   req.Sequence,
		"timestamp":  req.Timestamp,
		"accepted":   accepted,
		"verdict":    map[bool]string{true: "ACCEPTED (Válido)", false: "DROPPED (Replay/Jitter/Cross-Session)"}[accepted],
		"stats":      ws.AntiReplay.GetStats(),
	})
}

func (ws *WebDashboardServer) handleHierarchySnapshot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	snapshot := ws.Hierarchy.GetHierarchySnapshot()
	_ = json.NewEncoder(w).Encode(snapshot)
}

// Handlers de Aplicaciones Soberanas

func (ws *WebDashboardServer) handleChatSend(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		TargetDID     string `json:"target_did"`
		Text          string `json:"text"`
		AttachmentCID string `json:"attachment_cid"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	msg, err := ws.Chat.SendMessage(req.TargetDID, req.Text, req.AttachmentCID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(msg)
}

func (ws *WebDashboardServer) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	peerDID := r.URL.Query().Get("peer_did")
	if peerDID == "" {
		peerDID = "did:ipvn7:e821ef1a17f84e318f..."
	}

	history := ws.Chat.GetHistory(peerDID)
	_ = json.NewEncoder(w).Encode(history)
}

func (ws *WebDashboardServer) handleChatContacts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	contacts := ws.Chat.GetContacts()
	_ = json.NewEncoder(w).Encode(contacts)
}

func (ws *WebDashboardServer) handleDAGDownload(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")

	cid := r.URL.Query().Get("cid")
	if cid == "" {
		http.Error(w, "Parámetro 'cid' requerido", http.StatusBadRequest)
		return
	}

	block, exists := ws.DAGStore.GetBlock(cid)
	if !exists {
		http.Error(w, "Bloque DAG no encontrado", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"dag_%s.bin\"", cid[10:18]))
	_, _ = w.Write(block.Data)
}

func (ws *WebDashboardServer) handleRemoteStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	status := ws.RemoteDesktop.GetStatus()
	_ = json.NewEncoder(w).Encode(status)
}

func (ws *WebDashboardServer) handleRemoteInput(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var evt RemoteInputEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	resp, err := ws.RemoteDesktop.HandleInputEvent(&evt)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_ = json.NewEncoder(w).Encode(resp)
}

func (ws *WebDashboardServer) handleMulticastBroadcast(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		Channel string `json:"channel"`
		Payload string `json:"payload"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Channel == "" {
		req.Channel = "canal-soberano-01"
	}
	if req.Payload == "" {
		req.Payload = "CASCADE_BURST_1280B"
	}

	report := ws.Multicast.Broadcast(req.Channel, []byte(req.Payload))
	_ = json.NewEncoder(w).Encode(report)
}

func (ws *WebDashboardServer) handleMulticastTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	report := ws.Multicast.GetLastReport()
	_ = json.NewEncoder(w).Encode(report)
}

func (ws *WebDashboardServer) handleCorporateVPNStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	stats := ws.CorporateVPN.Stats()
	_ = json.NewEncoder(w).Encode(stats)
}

func (ws *WebDashboardServer) handleCorporateVPNToggle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		Enable bool `json:"enable"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Enable {
		_ = ws.CorporateVPN.Start(false)
	} else {
		_ = ws.CorporateVPN.Stop()
	}

	_ = json.NewEncoder(w).Encode(ws.CorporateVPN.Stats())
}

func (ws *WebDashboardServer) handleCorporateVPNMode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		Mode string `json:"mode"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.Mode == "KERNEL_TUN" {
		_ = ws.CorporateVPN.Start(true)
	} else {
		_ = ws.CorporateVPN.Start(false)
	}

	_ = json.NewEncoder(w).Encode(ws.CorporateVPN.Stats())
}

func (ws *WebDashboardServer) handleCorporateVPNEgress(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		EgressDID string `json:"egress_did"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	ws.CorporateVPN.SetEgressGateway(req.EgressDID)
	_ = json.NewEncoder(w).Encode(ws.CorporateVPN.Stats())
}

func (ws *WebDashboardServer) handleCorporateVPNZTNAEvaluate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	var req struct {
		PeerDID string `json:"peer_did"`
		Port    int    `json:"port"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)

	allowed := ws.CorporateVPN.CheckZTNAAccess(req.PeerDID, req.Port)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"peer_did": req.PeerDID,
		"port":     req.Port,
		"allowed":  allowed,
		"policy":   "ZTNA_DEFAULT_DENY_SOVEREIGN",
	})
}

// -----------------------------------------------------------------------------
// Handlers Streaming en Vivo, Audio Hi-Fi, Sonda de Red e IA Frugal
// -----------------------------------------------------------------------------

func (ws *WebDashboardServer) handleStreamChannels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	channels := ws.LiveStream.ListChannels()
	_ = json.NewEncoder(w).Encode(channels)
}

func (ws *WebDashboardServer) handleStreamPublish(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ChannelID string `json:"channel_id"`
		Keyframe  bool   `json:"keyframe"`
		MimeType  string `json:"mime_type"`
		Data      string `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	rawBytes := []byte(req.Data)
	seg, err := ws.LiveStream.IngestSegment(req.ChannelID, ws.Identity.DID(), req.Keyframe, req.MimeType, rawBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"status": "ok", "sequence": seg.Sequence})
}

func (ws *WebDashboardServer) handleStreamLive(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	channelID := r.URL.Query().Get("channel_id")
	if channelID == "" {
		channelID = "live-sovereign-01"
	}

	streamChan, cleanup, err := ws.LiveStream.Subscribe(channelID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer cleanup()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming no soportado", http.StatusInternalServerError)
		return
	}

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case seg, ok := <-streamChan:
			if !ok {
				return
			}
			dataJson, _ := json.Marshal(seg)
			fmt.Fprintf(w, "data: %s\n\n", dataJson)
			flusher.Flush()
		}
	}
}

func (ws *WebDashboardServer) handleAudioStations(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	stations := ws.AudioRadio.ListStations()
	_ = json.NewEncoder(w).Encode(stations)
}

func (ws *WebDashboardServer) handleAudioStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	stationID := r.URL.Query().Get("station_id")
	if stationID == "" {
		stationID = "radio-lofi-01"
	}

	audioChan, cleanup, err := ws.AudioRadio.SubscribeStation(stationID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	defer cleanup()

	w.Header().Set("Content-Type", "audio/pcm")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming no soportado", http.StatusInternalServerError)
		return
	}

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case chunk, ok := <-audioChan:
			if !ok {
				return
			}
			_, _ = w.Write(chunk)
			flusher.Flush()
		}
	}
}

func (ws *WebDashboardServer) handleAudioBroadcast(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		StationID string `json:"station_id"`
		AudioData string `json:"audio_data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "JSON inválido", http.StatusBadRequest)
		return
	}

	err := ws.AudioRadio.BroadcastAudio(req.StationID, []byte(req.AudioData))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "broadcast_accepted"})
}

func (ws *WebDashboardServer) handleProbeReport(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	report := ws.Probe.GetReport()
	_ = json.NewEncoder(w).Encode(report)
}

func (ws *WebDashboardServer) handleProbeToggle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method != http.MethodPost {
		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Active bool `json:"active"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	status := ws.Probe.SetActive(req.Active)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"is_active": status})
}

func (ws *WebDashboardServer) handleCopilotMode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodPost {
		var req struct {
			Mode string `json:"mode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err == nil && req.Mode != "" {
			ws.CopilotMode = req.Mode
		}
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"copilot_mode": ws.CopilotMode,
		"token_drain":  "ZERO_TOKENS_GUARANTEED",
		"is_optional":  true,
	})
}

func (ws *WebDashboardServer) handleBlackoutCascade(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// 1. Evaluar estado real del grafo y pares
	peers := ws.Router.GetAllPeers()
	directPeers := len(peers)
	pfo := ws.Kuzu.GetPFOTree()
	hasLocalHistory := directPeers > 0 || pfo["total_peers"].(int) > 0

	// 2. Ejecutar prueba reflexiva STUN RFC 5389 en vivo
	stunRes, _ := l1.ProbeSTUN(nil, 800*time.Millisecond)
	wanReachable := stunRes != nil

	// 3. Ejecución determinista de la FSM Post-Apagón
	currentPhase := ws.Blackout.StepCascade(hasLocalHistory, 1, wanReachable, directPeers)
	nextJitter := ws.Blackout.JitterTimer.NextInterval()

	phaseText := currentPhase.String()
	switch currentPhase {
	case l1.PhaseColdMemory:
		phaseText = "FASE 1: MEMORIA KÙZUDB (Sondeo Silencioso)"
	case l1.PhaseOffGridProximity:
		phaseText = "FASE 2: PROXIMIDAD OFFGRID (BLE / LoRa / LAN)"
	case l1.PhaseWANProbe:
		phaseText = "FASE 3: SONDEO WAN STUN (Reflexivo RFC 5389)"
	case l1.PhaseBlindRendezvous:
		phaseText = "FASE 4: BALIZA CIEGA (EBRA Zero-Knowledge)"
	case l1.PhaseConvergedMesh:
		phaseText = "FASE 5: RED CONVERGIDA P2P (Estable ✓)"
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success":       true,
		"phase":         phaseText,
		"phase_id":      int(currentPhase),
		"jitter_ms":     float64(nextJitter.Microseconds()) / 1000.0,
		"wan_reachable": wanReachable,
		"stun_result":   stunRes,
		"direct_peers":  directPeers,
		"logs":          ws.Blackout.GetLogs(),
		"converged":     directPeers >= 2,
	})
}











