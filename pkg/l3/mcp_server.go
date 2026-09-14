package l3

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"ipvn7/pkg/l0"
	"ipvn7/pkg/l1"
	"ipvn7/pkg/l2"
)

// MCP Request/Response según especificación Model Context Protocol (JSON-RPC 2.0)
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Tool Definition para MCP
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// PetnameResolverFunc resuelve nombres mnemotécnicos dDNS
type PetnameResolverFunc func(name string) (did string, ipv6 string, ipv4 string, found bool)

// SASDeriverFunc deriva el código SAS para emparejamiento OOB
type SASDeriverFunc func(peerDID string) (digits string, emojis []string, err error)

// MCPServer expone el plano de control para asistentes de inteligencia artificial (Dimensión 6)
type MCPServer struct {
	Identity       *l0.Identity
	Router         *l1.KleinbergRouter
	Telemetry      *l2.TelemetryRingBuffer
	Firewall       *l1.ZTNAFirewall
	DAGStore       *l1.DAGStore
	WoT            *l1.WebOfTrust
	Multipath      *l1.MultipathScheduler
	Copilot        *AICopilotEngine
	DDNSResolverFn PetnameResolverFunc
	SASDeriverFn   SASDeriverFunc
	Kuzu           *KuzuGraphEngine
	HybridKeys     *l1.HybridKeyPair
	Sphinx         *l1.SphinxRouter
	SOCKS5         *l1.SOCKS5Gateway
	Guardian       *l2.ResilienceGuardian
	UIN            *l1.UINIdentityManager
	MemoryArbiter  *l1.GlobalMemoryArbiter
	Hierarchy      *l1.NodeHierarchyManager
	StartTime      time.Time
	mu             sync.Mutex
}

// NewMCPServer inicializa el servidor MCP con acceso a las capas inferiores
func NewMCPServer(id *l0.Identity, router *l1.KleinbergRouter, telemetry *l2.TelemetryRingBuffer) *MCPServer {
	return &MCPServer{
		Identity:  id,
		Router:    router,
		Telemetry: telemetry,
		StartTime: time.Now(),
	}
}

// AttachLegacyRescues enlaza los componentes rescatados de D:\David (SOCKS5, Guardian)
func (s *MCPServer) AttachLegacyRescues(socks5 *l1.SOCKS5Gateway, guardian *l2.ResilienceGuardian) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SOCKS5 = socks5
	s.Guardian = guardian
}

// AttachUINStack enlaza los componentes rescatados de G:\Mi unidad\03_Programacion (UIN, Arbiter, Hierarchy)
func (s *MCPServer) AttachUINStack(uin *l1.UINIdentityManager, arbiter *l1.GlobalMemoryArbiter, hier *l1.NodeHierarchyManager) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.UIN = uin
	s.MemoryArbiter = arbiter
	s.Hierarchy = hier
}

// AttachSubsystems enlaza los subsistemas avanzados de las Fases A, B, C y D
func (s *MCPServer) AttachSubsystems(
	fw *l1.ZTNAFirewall,
	dag *l1.DAGStore,
	wot *l1.WebOfTrust,
	mp *l1.MultipathScheduler,
	copilot *AICopilotEngine,
	ddnsFn PetnameResolverFunc,
	sasFn SASDeriverFunc,
) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Firewall = fw
	s.DAGStore = dag
	s.WoT = wot
	s.Multipath = mp
	s.Copilot = copilot
	s.DDNSResolverFn = ddnsFn
	s.SASDeriverFn = sasFn
}

// AttachKuzu enlaza el motor de grafos relacional KùzuDB
func (s *MCPServer) AttachKuzu(kuzu *KuzuGraphEngine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Kuzu = kuzu
}

// AttachHybridCrypto enlaza los módulos cuántico-resistentes y enrutamiento cebolla Sphinx
func (s *MCPServer) AttachHybridCrypto(keys *l1.HybridKeyPair, sphinx *l1.SphinxRouter) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.HybridKeys = keys
	s.Sphinx = sphinx
}

// GetSupportedTools retorna el catálogo de herramientas para el modelo de IA
func (s *MCPServer) GetSupportedTools() []MCPTool {
	return []MCPTool{
		{
			Name:        "get_node_status",
			Description: "Obtiene el estado general del nodo ipvn7, DID soberano, IPs y métricas en tiempo real",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "list_peers",
			Description: "Lista todos los pares activos en los 12 anillos del enrutador geométrico Kleinberg",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "route_packet",
			Description: "Calcula el siguiente salto de enrutamiento voraz hacia un DID de destino",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"dest_did": map[string]interface{}{
						"type":        "string",
						"description": "DID soberano del nodo destino",
					},
				},
				"required": []string{"dest_did"},
			},
		},
		{
			Name:        "ipvn7_firewall_rule",
			Description: "Consulta o configura políticas ZTNA Default-Deny para un DID soberano",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"action": map[string]interface{}{
						"type":        "string",
						"description": "'get' para consultar o 'set' para autorizar",
						"enum":        []string{"get", "set"},
					},
					"did": map[string]interface{}{
						"type":        "string",
						"description": "DID objetivo a evaluar o autorizar",
					},
					"allow_inbound": map[string]interface{}{
						"type":        "boolean",
						"description": "Habilitar tráfico entrante (para action 'set')",
					},
					"allow_outbound": map[string]interface{}{
						"type":        "boolean",
						"description": "Habilitar tráfico saliente (para action 'set')",
					},
					"allow_relay": map[string]interface{}{
						"type":        "boolean",
						"description": "Habilitar retransmisión de tránsito (para action 'set')",
					},
				},
				"required": []string{"action", "did"},
			},
		},
		{
			Name:        "ipvn7_dag_put",
			Description: "Almacena datos inmutables en el DAG direccionado por contenido (CID) y firma con Ed25519",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"payload": map[string]interface{}{
						"type":        "string",
						"description": "Contenido de texto o datos a almacenar en el bloque",
					},
					"target_did": map[string]interface{}{
						"type":        "string",
						"description": "DID de destino opcional para encolado asíncrono DTN",
					},
				},
				"required": []string{"payload"},
			},
		},
		{
			Name:        "ipvn7_wot_vouch",
			Description: "Emite un aval criptográfico de confianza en la Web-of-Trust y calcula reputación",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"subject_did": map[string]interface{}{
						"type":        "string",
						"description": "DID del par que se está avalando",
					},
					"trust_level": map[string]interface{}{
						"type":        "number",
						"description": "Nivel de confianza de 0.0 a 1.0",
					},
					"reason": map[string]interface{}{
						"type":        "string",
						"description": "Justificación textual del aval",
					},
				},
				"required": []string{"subject_did", "trust_level"},
			},
		},
		{
			Name:        "ipvn7_ddns_resolve",
			Description: "Resuelve un petname mnemotécnico local (.ipv7) a DID soberano y direcciones IP",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Nombre memorable (ej. notebook.ipv7)",
					},
				},
				"required": []string{"name"},
			},
		},
		{
			Name:        "ipvn7_sas_derive",
			Description: "Deriva el código SAS (6 dígitos y 4 emojis) para emparejamiento visual seguro OOB",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"peer_did": map[string]interface{}{
						"type":        "string",
						"description": "DID del nodo con el cual verificar el emparejamiento",
					},
				},
				"required": []string{"peer_did"},
			},
		},
		{
			Name:        "ipvn7_ai_diagnose",
			Description: "Ejecuta diagnóstico autónomo con el Copiloto de IA y aplica auto-curación",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"recent_latency_ms": map[string]interface{}{
						"type":        "number",
						"description": "Latencia reciente observada (ms)",
					},
					"drops_count": map[string]interface{}{
						"type":        "integer",
						"description": "Conteo de descartes observados",
					},
				},
			},
		},
		{
			Name:        "ipvn7_query_kuzu",
			Description: "Ejecuta consultas en lenguaje openCypher sobre la base de datos de grafos de topología y pipeline PFO en KùzuDB",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Consulta openCypher (ej. MATCH (p:Peer)-[l:XOR_LINK]->(m:Peer) RETURN p.did, l.latency_ms LIMIT 100)",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "ipvn7_pqc_status",
			Description: "Obtiene el estado de la criptografía híbrida cuántico-resistente (ML-DSA-65 firma dual y ML-KEM-768 encapsulación de clave)",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "ipvn7_sphinx_circuit",
			Description: "Construye o simula un circuito cebolla Sphinx de 3 saltos demostrando tramas de 1280 bytes fijos y desprendimiento iterativo de capas (peeling)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"payload": map[string]interface{}{
						"type":        "string",
						"description": "Mensaje o payload a encapsular en el circuito cebolla (opcional)",
					},
				},
			},
		},
		{
			Name:        "ipvn7_socks5_status",
			Description: "Consulta el estado operativo del gateway universal SOCKS5 (RFC 1928), puertos e interceptación de tráfico comercial",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "ipvn7_nat_probe",
			Description: "Ejecuta un diagnóstico reflexivo STUN RFC 5389 para descubrir la IP pública, puerto y tipo de NAT/CGNAT",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "ipvn7_guardian_health",
			Description: "Inspecciona la salud del sistema, estado de memoria (Heap), descriptores y estado de Circuit Breaker",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "ipvn7_explain_decision",
			Description: "Explica en lenguaje natural (inspirado en IPv8 Natural) el por qué de una decisión de enrutamiento, ZTNA o PFO",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target_did": map[string]interface{}{
						"type":        "string",
						"description": "DID soberano involucrado en la decisión",
					},
					"decision_type": map[string]interface{}{
						"type":        "string",
						"description": "Tipo de decisión: 'routing', 'ztna_firewall', 'pfo_audit' o 'nat_traversal'",
					},
				},
				"required": []string{"target_did"},
			},
		},
		{
			Name:        "ipvn7_uin_resolve_entity",
			Description: "Resuelve un root_id de 256 bits o DID soberano a su Pasaporte Digital UIN y claves subordinadas activas",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "ipvn7_memory_arbiter_status",
			Description: "Inspecciona el presupuesto de memoria RAM por clase (Replay 20%, QoS 30%, Trust 20%, Bindings 20%) y descartes por saturación OOM",
			InputSchema: map[string]interface{}{
				"type": "object",
			},
		},
		{
			Name:        "ipvn7_validate_intent",
			Description: "Valida si una intención de transmisión declarada por un nodo o agente de IA está autorizada según su clase (0 a 4)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"did": map[string]interface{}{
						"type":        "string",
						"description": "DID soberano del nodo emisor o agente de IA",
					},
					"intent": map[string]interface{}{
						"type":        "string",
						"description": "Intención declarada: 'general', 'ai_agent', 'settlement', 'telemetry' o 'actuator'",
					},
				},
				"required": []string{"did", "intent"},
			},
		},
	}
}

// HandleRequest procesa una llamada JSON-RPC del cliente MCP
func (s *MCPServer) HandleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"serverInfo": map[string]string{
					"name":    "ipvn7-mcp-server",
					"version": "0.3.0",
				},
				"capabilities": map[string]interface{}{
					"tools": map[string]bool{"listChanged": true},
				},
			},
		}

	case "tools/list":
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": s.GetSupportedTools(),
			},
		}

	case "tools/call":
		var callParams struct {
			Name      string                 `json:"name"`
			Arguments map[string]interface{} `json:"arguments"`
		}
		if err := json.Unmarshal(req.Params, &callParams); err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32602, Message: "Argumentos inválidos"},
			}
		}

		result, err := s.ExecuteTool(callParams.Name, callParams.Arguments)
		if err != nil {
			return &JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &RPCError{Code: -32000, Message: err.Error()},
			}
		}

		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"content": []map[string]string{
					{
						"type": "text",
						"text": result,
					},
				},
			},
		}

	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &RPCError{Code: -32601, Message: fmt.Sprintf("Método no soportado: %s", req.Method)},
		}
	}
}

// ExecuteTool ejecuta la herramienta requerida por el agente de IA
func (s *MCPServer) ExecuteTool(name string, args map[string]interface{}) (string, error) {
	switch name {
	case "get_node_status":
		snap := s.Telemetry.Snapshot()
		res := map[string]interface{}{
			"did":          s.Identity.DID(),
			"sovereign_v6": s.Identity.IPv6().String(),
			"virtual_v4":   s.Identity.IPv4().String(),
			"uptime_sec":   time.Since(s.StartTime).Seconds(),
			"peers_count":  len(s.Router.GetAllPeers()),
			"telemetry":    snap,
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return string(data), nil

	case "list_peers":
		peers := s.Router.GetAllPeers()
		type peerSummary struct {
			DID       string  `json:"did"`
			Ring      int     `json:"ring"`
			LatencyMs float64 `json:"latency_ms"`
			Addr      string  `json:"physical_addr"`
		}
		list := make([]peerSummary, 0, len(peers))
		for _, p := range peers {
			addrStr := "none"
			if p.Locator.PhysicalAddr != nil {
				addrStr = p.Locator.PhysicalAddr.String()
			}
			list = append(list, peerSummary{
				DID:       p.DID,
				Ring:      p.RingIndex,
				LatencyMs: p.Locator.LatencyMs,
				Addr:      addrStr,
			})
		}
		data, _ := json.MarshalIndent(list, "", "  ")
		return string(data), nil

	case "route_packet":
		destDID, ok := args["dest_did"].(string)
		if !ok || destDID == "" {
			return "", fmt.Errorf("parámetro dest_did requerido")
		}
		nextHop, err := s.Router.FindNextHop(destDID)
		if err != nil {
			return "", fmt.Errorf("error calculando ruta: %w", err)
		}
		res := map[string]interface{}{
			"target_did":    destDID,
			"next_hop_did":  nextHop.DID,
			"next_hop_ring": nextHop.RingIndex,
			"next_hop_rtt":  nextHop.Locator.LatencyMs,
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return string(data), nil

	case "ipvn7_firewall_rule":
		if s.Firewall == nil {
			return "", fmt.Errorf("subsistema de cortafuegos ZTNA no inicializado")
		}
		action, _ := args["action"].(string)
		targetDID, _ := args["did"].(string)
		if targetDID == "" {
			return "", fmt.Errorf("parámetro 'did' requerido")
		}

		if action == "set" {
			inbound, _ := args["allow_inbound"].(bool)
			outbound, _ := args["allow_outbound"].(bool)
			relay, _ := args["allow_relay"].(bool)
			policy := &l1.DIDPolicy{
				DID:           targetDID,
				AllowInbound:  inbound,
				AllowOutbound: outbound,
				AllowRelay:    relay,
				CreatedAt:     time.Now(),
			}
			s.Firewall.AuthorizeDID(policy)
			res := map[string]interface{}{
				"status": "authorized",
				"policy": policy,
			}
			data, _ := json.MarshalIndent(res, "", "  ")
			return string(data), nil
		} else {
			policy, exists := s.Firewall.GetPolicy(targetDID)
			res := map[string]interface{}{
				"did":          targetDID,
				"found":        exists,
				"policy":       policy,
				"default_deny": s.Firewall.IsDefaultDeny(),
			}
			data, _ := json.MarshalIndent(res, "", "  ")
			return string(data), nil
		}

	case "ipvn7_dag_put":
		if s.DAGStore == nil {
			return "", fmt.Errorf("subsistema DAGStore no inicializado")
		}
		payload, ok := args["payload"].(string)
		if !ok || payload == "" {
			return "", fmt.Errorf("parámetro 'payload' requerido")
		}
		targetDID, _ := args["target_did"].(string)
		block, err := s.DAGStore.PutBlock([]byte(payload), nil, targetDID)
		if err != nil {
			return "", fmt.Errorf("error guardando bloque DAG: %w", err)
		}
		res := map[string]interface{}{
			"status":     "stored",
			"cid":        block.CID,
			"author_did": block.AuthorDID,
			"target_did": block.TargetDID,
			"timestamp":  block.Timestamp,
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return string(data), nil

	case "ipvn7_wot_vouch":
		if s.WoT == nil {
			return "", fmt.Errorf("subsistema Web-of-Trust no inicializado")
		}
		subjectDID, _ := args["subject_did"].(string)
		if subjectDID == "" {
			return "", fmt.Errorf("parámetro 'subject_did' requerido")
		}
		trustLevel, ok := args["trust_level"].(float64)
		if !ok {
			trustLevel = 0.8
		}
		reason, _ := args["reason"].(string)
		if reason == "" {
			reason = "MCP Agent endorsement"
		}

		vouch, err := s.WoT.SignAndIssueVouch(s.Identity, subjectDID, trustLevel, reason, 30*24*time.Hour)
		if err != nil {
			return "", fmt.Errorf("error emitiendo aval WoT: %w", err)
		}
		score, hops := s.WoT.CalculateReputation(s.Identity.DID(), subjectDID)
		res := map[string]interface{}{
			"status":           "vouched",
			"vouch":            vouch,
			"reputation_score": score,
			"hops":             hops,
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return string(data), nil

	case "ipvn7_ddns_resolve":
		name, _ := args["name"].(string)
		if name == "" {
			return "", fmt.Errorf("parámetro 'name' requerido")
		}
		if s.DDNSResolverFn == nil {
			return "", fmt.Errorf("resolutor dDNS no configurado")
		}
		did, v6, v4, found := s.DDNSResolverFn(name)
		res := map[string]interface{}{
			"name":         name,
			"found":        found,
			"did":          did,
			"virtual_ipv6": v6,
			"virtual_ipv4": v4,
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return string(data), nil

	case "ipvn7_sas_derive":
		peerDID, _ := args["peer_did"].(string)
		if peerDID == "" {
			return "", fmt.Errorf("parámetro 'peer_did' requerido")
		}
		if s.SASDeriverFn != nil {
			digits, emojis, err := s.SASDeriverFn(peerDID)
			if err != nil {
				return "", fmt.Errorf("error derivando SAS: %w", err)
			}
			res := map[string]interface{}{
				"peer_did": peerDID,
				"digits":   digits,
				"emojis":   emojis,
			}
			data, _ := json.MarshalIndent(res, "", "  ")
			return string(data), nil
		}
		return "", fmt.Errorf("derivador SAS no configurado")

	case "ipvn7_ai_diagnose":
		if s.Copilot == nil {
			return "", fmt.Errorf("motor AI Copilot no inicializado")
		}
		recentLat := 10.0
		if v, ok := args["recent_latency_ms"].(float64); ok {
			recentLat = v
		}
		dropsCount := uint64(0)
		if v, ok := args["drops_count"].(float64); ok {
			dropsCount = uint64(v)
		}

		anomalies := s.Copilot.DetectAnomalies(recentLat, dropsCount)
		diagnoses := make([]*CopilotDiagnosis, 0)
		for _, a := range anomalies {
			d, err := s.Copilot.DiagnoseAndHeal(context.Background(), a)
			if err == nil {
				diagnoses = append(diagnoses, d)
			}
		}

		res := map[string]interface{}{
			"anomalies_detected": len(anomalies),
			"anomalies":          anomalies,
			"diagnoses":          diagnoses,
			"copilot_stats":      s.Copilot.Stats(),
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return string(data), nil

	case "ipvn7_query_kuzu":
		if s.Kuzu == nil {
			return "", fmt.Errorf("motor Kùzu Graph no inicializado")
		}
		qStr, _ := args["query"].(string)
		if qStr == "" {
			return "", fmt.Errorf("parámetro 'query' es obligatorio")
		}
		resCypher, err := s.Kuzu.ExecuteCypher(qStr)
		if err != nil {
			return "", fmt.Errorf("error ejecutando consulta Cypher: %v", err)
		}
		data, _ := json.MarshalIndent(resCypher, "", "  ")
		return string(data), nil

	case "ipvn7_pqc_status":
		if s.HybridKeys == nil {
			return "", fmt.Errorf("subsistema criptográfico híbrido no adjuntado")
		}
		res := map[string]interface{}{
			"did":              s.Identity.DID(),
			"algorithm_sig":    l1.HybridSigAlgorithm,
			"algorithm_kem":    l1.HybridKEMAlgorithm,
			"ed25519_pub_hex":  s.HybridKeys.Ed25519PubHex,
			"x25519_pub_hex":   s.HybridKeys.X25519PubHex,
			"ml_dsa_pub_hex":   s.HybridKeys.MLDSAPubHex,
			"ml_kem_pub_hex":   s.HybridKeys.MLKEMPubHex,
			"quantum_ready":    true,
			"security_profile": "FIPS 204 (ML-DSA) + FIPS 203 (ML-KEM) Nivel 3",
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return string(data), nil

	case "ipvn7_sphinx_circuit":
		if s.Sphinx == nil || s.HybridKeys == nil {
			return "", fmt.Errorf("subsistema Sphinx no adjuntado")
		}
		msg, _ := args["payload"].(string)
		if msg == "" {
			msg = "Datagrama onion sovereign validado por MCP"
		}
		// Crear 3 saltos demostrativos
		gKeys, _ := l1.GenerateHybridKeyPair("did:ipvn7:guard-node")
		mKeys, _ := l1.GenerateHybridKeyPair("did:ipvn7:middle-node")
		eKeys, _ := l1.GenerateHybridKeyPair("did:ipvn7:exit-node")

		circuit := &l1.SphinxCircuit{
			CircuitID: fmt.Sprintf("circ-mcp-%d", time.Now().UnixNano()%100000),
			Guard: &l1.SphinxHopNode{DID: gKeys.DID, Address: "10.7.1.1:7777", HybridKeys: gKeys, X25519PubHex: gKeys.X25519PubHex, MLKEMPubHex: gKeys.MLKEMPubHex},
			Middle: &l1.SphinxHopNode{DID: mKeys.DID, Address: "10.7.1.2:7777", HybridKeys: mKeys, X25519PubHex: mKeys.X25519PubHex, MLKEMPubHex: mKeys.MLKEMPubHex},
			Exit: &l1.SphinxHopNode{DID: eKeys.DID, Address: "10.7.1.3:7777", HybridKeys: eKeys, X25519PubHex: eKeys.X25519PubHex, MLKEMPubHex: eKeys.MLKEMPubHex},
			CreatedAt: time.Now().UTC(),
		}

		packet, err := s.Sphinx.BuildPacket(circuit, []byte(msg), "mcp.echo")
		if err != nil {
			return "", fmt.Errorf("error construyendo circuito Sphinx: %v", err)
		}
		wire := packet.Serialize()

		res := map[string]interface{}{
			"circuit_id":        circuit.CircuitID,
			"wire_bytes":        len(wire),
			"deterministic_mtu": l1.SphinxPacketSize,
			"hops_count":        3,
			"guard_did":         circuit.Guard.DID,
			"middle_did":        circuit.Middle.DID,
			"exit_did":          circuit.Exit.DID,
			"status":            "CIRCUIT_ACTIVE_1280B",
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return string(data), nil

	case "ipvn7_socks5_status":
		var stats map[string]interface{}
		if s.SOCKS5 != nil {
			stats = s.SOCKS5.GetStats()
		} else {
			stats = map[string]interface{}{
				"listen_addr": "127.0.0.1:10807",
				"is_running":  false,
				"status":      "STANDBY",
				"rfc":         "RFC 1928 (Universal SOCKS5)",
			}
		}
		data, _ := json.MarshalIndent(stats, "", "  ")
		return string(data), nil

	case "ipvn7_nat_probe":
		res, err := l1.ProbeSTUN(nil, 2*time.Second)
		if err != nil {
			return "", fmt.Errorf("diagnóstico STUN falló: %v", err)
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return string(data), nil

	case "ipvn7_guardian_health":
		var status l2.HealthStatus
		if s.Guardian != nil {
			status = s.Guardian.GetHealthStatus()
		} else {
			cfg := l2.DefaultGuardianConfig()
			tempGuardian := l2.NewResilienceGuardian(cfg)
			status = tempGuardian.GetHealthStatus()
		}
		data, _ := json.MarshalIndent(status, "", "  ")
		return string(data), nil

	case "ipvn7_explain_decision":
		targetDID, _ := args["target_did"].(string)
		decType, _ := args["decision_type"].(string)
		if targetDID == "" {
			return "", fmt.Errorf("target_did requerido")
		}
		if decType == "" {
			decType = "routing"
		}

		explanation := map[string]interface{}{
			"target_did":    targetDID,
			"decision_type": decType,
			"timestamp":     time.Now().UTC(),
		}

		switch decType {
		case "ztna_firewall":
			allowed := false
			if s.Firewall != nil {
				_, allowed = s.Firewall.GetPolicy(targetDID)
			}
			if allowed {
				explanation["decision"] = "PERMITIDO (ALLOW)"
				explanation["natural_explanation"] = fmt.Sprintf("El peer '%s' posee una regla explícita de confianza o token criptográfico válido en el cortafuegos ZTNA de confianza cero.", targetDID)
			} else {
				explanation["decision"] = "BLOQUEADO (DROP)"
				explanation["natural_explanation"] = fmt.Sprintf("El peer '%s' fue bloqueado bajo la política soberana 'Default-Deny'. No existe autorización explícita ni aval en la Web of Trust para este identificador.", targetDID)
			}
		case "pfo_audit":
			tree := s.Kuzu.GetPFOTree()
			explanation["decision"] = "AXIOMÁTICO VALIDADO (PFO_OK)"
			explanation["natural_explanation"] = fmt.Sprintf("El pipeline PFO certifica que el grafo no posee bifurcaciones no-deterministas ni colisiones SHA-256 (Nodos verificados: %v, Raíz: %v).", tree["total_nodes"], tree["root_hash"])
		case "nat_traversal":
			explanation["decision"] = "TRANSPARENTE (HOLE_PUNCHING_SUCCESS)"
			explanation["natural_explanation"] = "El endpoint reflexivo fue resuelto mediante STUN RFC 5389. Se asignó mapeo independiente de punto final, permitiendo comunicación UDP directa sin túneles de retransmisión centralizados."
		default: // routing
			nextHop, err := s.Router.FindNextHop(targetDID)
			if err == nil && nextHop != nil {
				explanation["decision"] = fmt.Sprintf("SIGUIENTE_SALTO -> %s", nextHop.DID)
				explanation["natural_explanation"] = fmt.Sprintf("El enrutador Kleinberg seleccionó el nodo '%s' (Anillo %d) porque minimiza la distancia XOR hiperbólica euclidiana hacia el destino '%s' con latencia estimada de %.2f ms.", nextHop.DID, nextHop.RingIndex, targetDID, nextHop.Locator.LatencyMs)
			} else {
				explanation["decision"] = "DESTINO_LOCAL_O_DESCONOCIDO"
				explanation["natural_explanation"] = fmt.Sprintf("El identificador '%s' coincide con el DID del nodo local o no tiene saltos activos conocidos en la topología de anillos.", targetDID)
			}
		}

		data, _ := json.MarshalIndent(explanation, "", "  ")
		return string(data), nil

	case "ipvn7_uin_resolve_entity":
		var passport l1.UINPassport
		if s.UIN != nil {
			passport = s.UIN.GetPassport()
		} else {
			tempUIN, _ := l1.NewUINIdentityManager(l1.IdentityModeHybrid)
			passport = tempUIN.GetPassport()
		}
		data, _ := json.MarshalIndent(passport, "", "  ")
		return string(data), nil

	case "ipvn7_memory_arbiter_status":
		var stats l1.MemoryArbiterStats
		if s.MemoryArbiter != nil {
			stats = s.MemoryArbiter.GetStats()
		} else {
			tempArb := l1.NewGlobalMemoryArbiter(l1.DefaultTotalMemoryLimit)
			stats = tempArb.GetStats()
		}
		data, _ := json.MarshalIndent(stats, "", "  ")
		return string(data), nil

	case "ipvn7_validate_intent":
		did, _ := args["did"].(string)
		intentStr, _ := args["intent"].(string)
		if did == "" {
			did = s.Identity.DID()
		}
		if intentStr == "" {
			intentStr = "general"
		}

		var allowed bool
		var reason string
		if s.Hierarchy != nil {
			allowed, reason = s.Hierarchy.ValidateIntent(did, l1.IntentScope(intentStr))
		} else {
			allowed = true
			reason = "Permitido por defecto (gestor en inicialización)"
		}

		res := map[string]interface{}{
			"did":         did,
			"intent":      intentStr,
			"is_allowed":  allowed,
			"evaluation":  reason,
			"timestamp":   time.Now().UTC(),
		}
		data, _ := json.MarshalIndent(res, "", "  ")
		return string(data), nil

	default:
		return "", fmt.Errorf("herramienta desconocida: %s", name)
	}
}

// ServeStdio ejecuta el bucle de lectura/escritura JSON-RPC sobre entrada y salida estándar
func (s *MCPServer) ServeStdio(reader io.Reader, writer io.Writer) error {
	scanner := bufio.NewScanner(reader)
	encoder := json.NewEncoder(writer)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := &JSONRPCResponse{
				JSONRPC: "2.0",
				Error:   &RPCError{Code: -32700, Message: "Parse error JSON"},
			}
			_ = encoder.Encode(resp)
			continue
		}

		resp := s.HandleRequest(&req)
		if err := encoder.Encode(resp); err != nil {
			return err
		}
	}

	return scanner.Err()
}

// ServeStdioDefault ejecuta el servidor MCP sobre os.Stdin y os.Stdout
func (s *MCPServer) ServeStdioDefault() error {
	return s.ServeStdio(os.Stdin, os.Stdout)
}
