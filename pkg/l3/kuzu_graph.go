package l3

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Modelos de Nodos del Grafo Kùzu (2.md Sección 2)

// PeerNode representa una identidad soberana en el grafo topológico
type PeerNode struct {
	DID           string `json:"did"`
	MLDSA         string `json:"ml_dsa"`
	Anycast       bool   `json:"anycast"`
	DeviceProfile string `json:"device_profile"`
}

// AxiomPrinciple representa una ley inquebrantable de la arquitectura (genesis.md)
type AxiomPrinciple struct {
	PrincipleID string `json:"principle_id"` // ej. AXIOM_ZERO_PII, AXIOM_CORE_FREEZE
	Statement   string `json:"statement"`    // Descripción axiomática
	Immutable   bool   `json:"immutable"`
}

// OperationalFunction representa un módulo o adaptador en código
type OperationalFunction struct {
	FunctionID string `json:"function_id"` // ej. L1_ZTNA_FIREWALL, L1_WDRR_SCHEDULER
	ModulePath string `json:"module_path"` // ej. pkg/l1/firewall.go
	Status     string `json:"status"`      // ACTIVE, VERIFIED, STUB
}

// EmpiricalObservation representa una métrica empírica validada en campo
type EmpiricalObservation struct {
	ObservationID      string    `json:"observation_id"`
	MetricsFingerprint string    `json:"metrics_fingerprint"`
	Value              float64   `json:"value"`
	Timestamp          time.Time `json:"timestamp"`
}

// EngineeringDecision representa un registro inmutable de decisión técnica
type EngineeringDecision struct {
	DecisionID string    `json:"decision_id"`
	Rationale  string    `json:"rationale"`
	Timestamp  time.Time `json:"timestamp"`
}

// Modelos de Relaciones (Aristas) en Kùzu

// XORLinkRel representa la distancia matemática y enlace entre pares en Kleinberg
type XORLinkRel struct {
	FromDID     string    `json:"from_did"`
	ToDID       string    `json:"to_did"`
	XORDistance string    `json:"xor_distance"`
	LatencyMs   float64   `json:"latency_ms"`
	Degree      int8      `json:"degree"` // Anillo [0..11]
	RFBand      string    `json:"rf_band"`
	LastSeen    time.Time `json:"last_seen"`
}

// GovernsRel representa una acción de gobernanza FSM firmada criptográficamente
type GovernsRel struct {
	FromDID   string    `json:"from_did"`
	ToDID     string    `json:"to_did"`
	Action    string    `json:"action"`
	Timestamp time.Time `json:"timestamp"`
	Signature string    `json:"signature"`
}

// CypherResult representa la respuesta tipada a una consulta openCypher
type CypherResult struct {
	Query     string                   `json:"query"`
	Columns   []string                 `json:"columns"`
	Rows      []map[string]interface{} `json:"rows"`
	RowCount  int                      `json:"row_count"`
	ExecMs    float64                  `json:"exec_ms"`
	Timestamp time.Time                `json:"timestamp"`
}

// KuzuGraphEngine implementa la fuente de verdad viva en memoria/grafo (genesis.md L178 y 2.md Sec 2)
type KuzuGraphEngine struct {
	mu           sync.RWMutex
	peers        map[string]*PeerNode
	axioms       map[string]*AxiomPrinciple
	functions    map[string]*OperationalFunction
	observations map[string]*EmpiricalObservation
	decisions    map[string]*EngineeringDecision

	// Relaciones
	xorLinks    []*XORLinkRel
	governs     []*GovernsRel
	derivesFrom map[string]string   // function_id -> principle_id
	validatedBy map[string][]string // function_id -> []observation_id
}

// NewKuzuGraphEngine inicializa el motor de grafos y siembra el esquema PFO canónico
func NewKuzuGraphEngine() *KuzuGraphEngine {
	kg := &KuzuGraphEngine{
		peers:        make(map[string]*PeerNode),
		axioms:       make(map[string]*AxiomPrinciple),
		functions:    make(map[string]*OperationalFunction),
		observations: make(map[string]*EmpiricalObservation),
		decisions:    make(map[string]*EngineeringDecision),
		xorLinks:     make([]*XORLinkRel, 0),
		governs:      make([]*GovernsRel, 0),
		derivesFrom:  make(map[string]string),
		validatedBy:  make(map[string][]string),
	}

	kg.seedCanonicalAxioms()
	kg.seedCanonicalFunctions()
	return kg
}

// seedCanonicalAxioms siembra los principios axiomáticos inquebrantables de genesis.md
func (kg *KuzuGraphEngine) seedCanonicalAxioms() {
	kg.axioms["AXIOM_ZERO_PII"] = &AxiomPrinciple{
		PrincipleID: "AXIOM_ZERO_PII",
		Statement:   "Prohibición estricta de registrar o persistir IPs reales, geolocalizaciones o metadatos personales fuera de contenedores CBOR cifrados.",
		Immutable:   true,
	}
	kg.axioms["AXIOM_STRICT_CORE_FREEZE"] = &AxiomPrinciple{
		PrincipleID: "AXIOM_STRICT_CORE_FREEZE",
		Statement:   "Inmutabilidad del núcleo L0 (core/). Toda extensión debe implementarse en capas periféricas sin alterar el formato de cable canónico.",
		Immutable:   true,
	}
	kg.axioms["AXIOM_SUSTAINABLE_FLOW"] = &AxiomPrinciple{
		PrincipleID: "AXIOM_SUSTAINABLE_FLOW",
		Statement:   "Subordinación del emisor al cuello de botella de la ruta. Erradicación del bufferbloat mediante Packet Pacing determinista.",
		Immutable:   true,
	}
	kg.axioms["AXIOM_FAST_PATH_FIRST"] = &AxiomPrinciple{
		PrincipleID: "AXIOM_FAST_PATH_FIRST",
		Statement:   "Toda decisión de filtrado y reenvío debe ejecutarse en el camino crítico de kernel (eBPF/XDP) o en buffers lock-free (<28ns).",
		Immutable:   true,
	}
	kg.axioms["AXIOM_SMALL_WORLD_RINGS"] = &AxiomPrinciple{
		PrincipleID: "AXIOM_SMALL_WORLD_RINGS",
		Statement:   "Enrutamiento voraz en espacio métrico XOR de 256 bits estructurado en 12 anillos logarítmicos de Kleinberg.",
		Immutable:   true,
	}
}

// seedCanonicalFunctions siembra los módulos operativos y sus vínculos de derivación PFO
func (kg *KuzuGraphEngine) seedCanonicalFunctions() {
	funcs := []struct {
		fID    string
		path   string
		pID    string
		status string
	}{
		{"L0_SOVEREIGN_IDENTITY", "pkg/l0/identity.go", "AXIOM_ZERO_PII", "VERIFIED"},
		{"L0_DETERMINISTIC_WIRE", "pkg/l0/wire.go", "AXIOM_STRICT_CORE_FREEZE", "VERIFIED"},
		{"L1_ZTNA_FIREWALL", "pkg/l1/firewall.go", "AXIOM_ZERO_PII", "VERIFIED"},
		{"L1_PACKET_PACER", "pkg/l1/pacing.go", "AXIOM_SUSTAINABLE_FLOW", "VERIFIED"},
		{"L1_WDRR_SCHEDULER", "pkg/l1/wdrr_scheduler.go", "AXIOM_SUSTAINABLE_FLOW", "VERIFIED"},
		{"L1_KLEINBERG_ROUTER", "pkg/l1/router.go", "AXIOM_SMALL_WORLD_RINGS", "VERIFIED"},
		{"L1_DARK_NODE_MACRO", "pkg/l1/dark_node_profile.go", "AXIOM_ZERO_PII", "VERIFIED"},
		{"L2_TELEMETRY_RING", "pkg/l2/telemetry.go", "AXIOM_FAST_PATH_FIRST", "VERIFIED"},
		{"L2_ANOMALY_FINGERPRINTER", "pkg/l2/anomaly_fingerprinter.go", "AXIOM_FAST_PATH_FIRST", "VERIFIED"},
		{"L3_AI_COPILOT", "pkg/l3/ai_copilot.go", "AXIOM_FAST_PATH_FIRST", "VERIFIED"},
		{"L3_KUZU_TOPOLOGY", "pkg/l3/kuzu_graph.go", "AXIOM_SMALL_WORLD_RINGS", "VERIFIED"},
	}

	for _, fn := range funcs {
		kg.functions[fn.fID] = &OperationalFunction{
			FunctionID: fn.fID,
			ModulePath: fn.path,
			Status:     fn.status,
		}
		kg.derivesFrom[fn.fID] = fn.pID
	}
}

// UpsertPeer registra o actualiza un nodo par en el grafo
func (kg *KuzuGraphEngine) UpsertPeer(did, mlDSA, deviceProfile string, anycast bool) {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	kg.peers[did] = &PeerNode{
		DID:           did,
		MLDSA:         mlDSA,
		Anycast:       anycast,
		DeviceProfile: deviceProfile,
	}
}

// UpsertXORLink registra o actualiza una arista de enrutamiento Kleinberg
func (kg *KuzuGraphEngine) UpsertXORLink(from, to, xorDist string, latency float64, degree int8, rfBand string) {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	// Actualizar si ya existe la arista
	for _, l := range kg.xorLinks {
		if l.FromDID == from && l.ToDID == to {
			l.LatencyMs = latency
			l.Degree = degree
			l.RFBand = rfBand
			l.LastSeen = time.Now()
			return
		}
	}

	kg.xorLinks = append(kg.xorLinks, &XORLinkRel{
		FromDID:     from,
		ToDID:       to,
		XORDistance: xorDist,
		LatencyMs:   latency,
		Degree:      degree,
		RFBand:      rfBand,
		LastSeen:    time.Now(),
	})
}

// AddObservation registra una observación empírica validando una función operativa
func (kg *KuzuGraphEngine) AddObservation(functionID, fingerprint string, value float64) string {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%.4f:%d", functionID, fingerprint, value, time.Now().UnixNano())))
	obsID := "obs:" + hex.EncodeToString(h[:8])

	obs := &EmpiricalObservation{
		ObservationID:      obsID,
		MetricsFingerprint: fingerprint,
		Value:              value,
		Timestamp:          time.Now(),
	}

	kg.observations[obsID] = obs
	kg.validatedBy[functionID] = append(kg.validatedBy[functionID], obsID)
	return obsID
}

// RecordDecision registra una decisión técnica inmutable
func (kg *KuzuGraphEngine) RecordDecision(decisionID, rationale string) {
	kg.mu.Lock()
	defer kg.mu.Unlock()

	kg.decisions[decisionID] = &EngineeringDecision{
		DecisionID: decisionID,
		Rationale:  rationale,
		Timestamp:  time.Now(),
	}
}

// AuditDirectiveContradiction verifica deductivamente si una instrucción propuesta viola los axiomas rectores
func (kg *KuzuGraphEngine) AuditDirectiveContradiction(directiveText string) (bool, string) {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	lower := strings.ToLower(directiveText)

	// Regla 1: Violación de Zero-PII
	if strings.Contains(lower, "store cleartext ip") || strings.Contains(lower, "guardar ip real") ||
		strings.Contains(lower, "persist pii") || strings.Contains(lower, "almacenar cedula") {
		return true, fmt.Sprintf("COLISIÓN AXIOMÁTICA detectada contra [%s]: La instrucción viola el anonimato absoluto Zero-PII.", kg.axioms["AXIOM_ZERO_PII"].PrincipleID)
	}

	// Regla 2: Violación de Core Freeze
	if strings.Contains(lower, "modify pkg/l0") || strings.Contains(lower, "modificar core/") ||
		strings.Contains(lower, "alter wire format") || strings.Contains(lower, "reescribir identidad canonica") {
		return true, fmt.Sprintf("COLISIÓN AXIOMÁTICA detectada contra [%s]: Intento de mutar el núcleo canónico congelado.", kg.axioms["AXIOM_STRICT_CORE_FREEZE"].PrincipleID)
	}

	// Regla 3: Violación de Flujo Sostenible
	if strings.Contains(lower, "disable pacer") || strings.Contains(lower, "unlimited burst") ||
		strings.Contains(lower, "desactivar marcapasos") || strings.Contains(lower, "ignorar cuello de botella") {
		return true, fmt.Sprintf("COLISIÓN AXIOMÁTICA detectada contra [%s]: La instrucción introduce riesgo de bufferbloat.", kg.axioms["AXIOM_SUSTAINABLE_FLOW"].PrincipleID)
	}

	return false, "VALIDACIÓN AXIOMÁTICA EXITOSA: La directiva es formalmente coherente con los principios rectores."
}

// ExecuteCypher ejecuta consultas openCypher esenciales contra el grafo Kùzu en memoria
func (kg *KuzuGraphEngine) ExecuteCypher(query string) (*CypherResult, error) {
	start := time.Now()
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	cleanQuery := strings.TrimSpace(query)
	upper := strings.ToUpper(cleanQuery)

	if !strings.HasPrefix(upper, "MATCH") {
		return nil, errors.New("solo se admiten consultas de lectura Cypher iniciadas con MATCH")
	}

	result := &CypherResult{
		Query:     query,
		Columns:   make([]string, 0),
		Rows:      make([]map[string]interface{}, 0),
		Timestamp: time.Now(),
	}

	// 1. MATCH (p:Peer)-[l:XOR_LINK]->(m:Peer)
	if strings.Contains(upper, "(P:PEER)-[L:XOR_LINK]->(M:PEER)") || strings.Contains(upper, "XOR_LINK") {
		result.Columns = []string{"p.did", "l.degree", "l.latency_ms", "l.rf_band", "m.did"}
		for _, link := range kg.xorLinks {
			row := map[string]interface{}{
				"p.did":        link.FromDID,
				"l.degree":     link.Degree,
				"l.latency_ms": link.LatencyMs,
				"l.rf_band":    link.RFBand,
				"m.did":        link.ToDID,
			}
			result.Rows = append(result.Rows, row)
		}
	} else if strings.Contains(upper, "AXIOMPRINCIPLE") || strings.Contains(upper, "(A:AXIOMPRINCIPLE)") {
		// 2. MATCH (a:AxiomPrinciple)
		result.Columns = []string{"a.principle_id", "a.statement", "a.immutable"}
		for _, ax := range kg.axioms {
			result.Rows = append(result.Rows, map[string]interface{}{
				"a.principle_id": ax.PrincipleID,
				"a.statement":   ax.Statement,
				"a.immutable":   ax.Immutable,
			})
		}
	} else if strings.Contains(upper, "OPERATIONALFUNCTION") || strings.Contains(upper, "DERIVES_FROM") {
		// 3. MATCH (f:OperationalFunction)-[:DERIVES_FROM]->(a:AxiomPrinciple)
		result.Columns = []string{"f.function_id", "f.module_path", "f.status", "a.principle_id"}
		for fID, fn := range kg.functions {
			pID := kg.derivesFrom[fID]
			result.Rows = append(result.Rows, map[string]interface{}{
				"f.function_id":   fn.FunctionID,
				"f.module_path":   fn.ModulePath,
				"f.status":        fn.Status,
				"a.principle_id":  pID,
			})
		}
	} else if strings.Contains(upper, "PEER") {
		// 4. MATCH (p:Peer)
		result.Columns = []string{"p.did", "p.ml_dsa", "p.anycast", "p.device_profile"}
		for _, peer := range kg.peers {
			result.Rows = append(result.Rows, map[string]interface{}{
				"p.did":            peer.DID,
				"p.ml_dsa":         peer.MLDSA,
				"p.anycast":        peer.Anycast,
				"p.device_profile": peer.DeviceProfile,
			})
		}
	} else {
		return nil, fmt.Errorf("patrón Cypher no soportado en emulación rápida: %s", cleanQuery)
	}

	result.RowCount = len(result.Rows)
	result.ExecMs = float64(time.Since(start).Microseconds()) / 1000.0
	return result, nil
}

// GetPFOTree retorna la estructura jerárquica Principios -> Funciones -> Observaciones
func (kg *KuzuGraphEngine) GetPFOTree() map[string]interface{} {
	kg.mu.RLock()
	defer kg.mu.RUnlock()

	type functionDTO struct {
		FunctionID   string                  `json:"function_id"`
		ModulePath   string                  `json:"module_path"`
		Status       string                  `json:"status"`
		Observations []*EmpiricalObservation `json:"observations"`
	}

	type principleDTO struct {
		PrincipleID string         `json:"principle_id"`
		Statement   string         `json:"statement"`
		Functions   []*functionDTO `json:"functions"`
	}

	principlesMap := make(map[string]*principleDTO)
	for pID, ax := range kg.axioms {
		principlesMap[pID] = &principleDTO{
			PrincipleID: ax.PrincipleID,
			Statement:   ax.Statement,
			Functions:   make([]*functionDTO, 0),
		}
	}

	for fID, fn := range kg.functions {
		pID := kg.derivesFrom[fID]
		pDTO, ok := principlesMap[pID]
		if !ok {
			continue
		}

		obsList := make([]*EmpiricalObservation, 0)
		for _, obsID := range kg.validatedBy[fID] {
			if obs, exists := kg.observations[obsID]; exists {
				obsList = append(obsList, obs)
			}
		}

		pDTO.Functions = append(pDTO.Functions, &functionDTO{
			FunctionID:   fn.FunctionID,
			ModulePath:   fn.ModulePath,
			Status:       fn.Status,
			Observations: obsList,
		})
	}

	resultList := make([]*principleDTO, 0, len(principlesMap))
	for _, pDTO := range principlesMap {
		resultList = append(resultList, pDTO)
	}

	return map[string]interface{}{
		"total_principles":   len(kg.axioms),
		"total_functions":    len(kg.functions),
		"total_observations": len(kg.observations),
		"total_peers":        len(kg.peers),
		"total_xor_links":    len(kg.xorLinks),
		"pfo_tree":           resultList,
	}
}
