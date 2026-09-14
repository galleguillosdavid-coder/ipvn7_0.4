// Package l1 implementa la Jerarquía de Clases de Nodos y el Ámbito de Intención UIN
// rescatados de IPv7_Especificacion_Tecnica_v4.pdf (Páginas 4-7) y de
// IPv7_UIN_Arquitectura_Unificada_v4.docx (Partes II y III).
// Define 5 clases formales de nodos (0 a 4) con reputación jerárquica y metadatos de intención
// para la Web de Agentes de IA e Internet de las Cosas Confiable (Trusted IoT).
package l1

import (
	"fmt"
	"sync"
	"time"
)

// NodeClass define la categoría de nodo en la topología soberana
type NodeClass uint8

const (
	NodeClassSensor   NodeClass = 0 // Sensor IoT: DID efímero, delegado en Gateway Clase 2
	NodeClassTerminal NodeClass = 1 // Terminal móvil/PC: Claves renovables por sesión
	NodeClassGateway  NodeClass = 2 // Gateway doméstico/oficina: Garante de Clase 0 y 1
	NodeClassMeshNode NodeClass = 3 // Nodo de Malla: Tránsito soberano, PQC continuo
	NodeClassAnchor   NodeClass = 4 // Ancla Soberana: Máxima reputación y alta disponibilidad
)

// IntentScope define el propósito de la transmisión para agentes de IA y ZTNA
type IntentScope string

const (
	IntentGeneral    IntentScope = "general"     // Comunicación estándar
	IntentAIAgent    IntentScope = "ai_agent"    // Coordinación autónoma entre modelos de IA
	IntentFinancial  IntentScope = "settlement"  // Liquidación de micropagos Tit-for-Tat
	IntentTelemetry  IntentScope = "telemetry"   // Métricas pasivas de observabilidad
	IntentActuator   IntentScope = "actuator"    // Comandos de control crítico a infraestructura física
)

// NodeProfile representa la ficha técnica de un nodo en la jerarquía
type NodeProfile struct {
	DID            string      `json:"did"`
	Class          NodeClass   `json:"class"`
	ClassName      string      `json:"class_name"`
	GatewayDID     string      `json:"gateway_did,omitempty"` // Si es clase 0 o 1, su nodo garante
	Subordinates   []string    `json:"subordinates,omitempty"` // Si es clase 2, sus subordinados
	AllowedIntents []IntentScope `json:"allowed_intents"`
	Reputation     float64     `json:"reputation"`
	RegisteredAt   time.Time   `json:"registered_at"`
}

// NodeHierarchyManager gestiona las clases de nodos y las políticas de intención
type NodeHierarchyManager struct {
	mu       sync.RWMutex
	localDID string
	profiles map[string]*NodeProfile
}

// NewNodeHierarchyManager inicializa el gestor de jerarquía
func NewNodeHierarchyManager(localDID string, localClass NodeClass) *NodeHierarchyManager {
	mgr := &NodeHierarchyManager{
		localDID: localDID,
		profiles: make(map[string]*NodeProfile),
	}

	mgr.RegisterNode(localDID, localClass, "", []IntentScope{
		IntentGeneral,
		IntentAIAgent,
		IntentFinancial,
		IntentTelemetry,
		IntentActuator,
	}, 100.0)

	return mgr
}

// ClassName retorna la designación canónica de una clase
func (c NodeClass) ClassName() string {
	switch c {
	case NodeClassSensor:
		return "Clase 0: Sensor IoT Efímero"
	case NodeClassTerminal:
		return "Clase 1: Terminal Cliente"
	case NodeClassGateway:
		return "Clase 2: Gateway Garante"
	case NodeClassMeshNode:
		return "Clase 3: Nodo de Malla PQC"
	case NodeClassAnchor:
		return "Clase 4: Ancla Soberana"
	default:
		return "Clase Desconocida"
	}
}

// RegisterNode registra o actualiza un perfil en la jerarquía
func (m *NodeHierarchyManager) RegisterNode(did string, class NodeClass, gatewayDID string, intents []IntentScope, reputation float64) *NodeProfile {
	m.mu.Lock()
	defer m.mu.Unlock()

	p := &NodeProfile{
		DID:            did,
		Class:          class,
		ClassName:      class.ClassName(),
		GatewayDID:     gatewayDID,
		AllowedIntents: intents,
		Reputation:     reputation,
		RegisteredAt:   time.Now().UTC(),
	}

	// Si tiene gateway, registrar como subordinado en el gateway
	if gatewayDID != "" {
		if gw, exists := m.profiles[gatewayDID]; exists {
			gw.Subordinates = append(gw.Subordinates, did)
		}
	}

	m.profiles[did] = p
	return p
}

// ValidateIntent evalúa si un nodo emisor está autorizado para transmitir bajo una intención específica
func (m *NodeHierarchyManager) ValidateIntent(srcDID string, intent IntentScope) (bool, string) {
	m.mu.RLock()
	p, exists := m.profiles[srcDID]
	m.mu.RUnlock()

	if !exists {
		// Si no está registrado explícitamente, solo permitimos IntentGeneral
		if intent == IntentGeneral || intent == IntentTelemetry {
			return true, "Permitido bajo política general de visitante"
		}
		return false, fmt.Sprintf("Nodo '%s' no autorizado para la intención restringida '%s'", srcDID, intent)
	}

	// Sensores Clase 0 no pueden emitir comandos a actuadores ni transacciones financieras directamente
	if p.Class == NodeClassSensor && (intent == IntentActuator || intent == IntentFinancial) {
		return false, fmt.Sprintf("Nodos Clase 0 (Sensor) tienen prohibido emitir intención '%s'. Requiere delegación de Gateway Clase 2.", intent)
	}

	for _, allowed := range p.AllowedIntents {
		if allowed == intent {
			return true, fmt.Sprintf("Intención '%s' autorizada para nodo %s", intent, p.ClassName)
		}
	}

	return false, fmt.Sprintf("La intención '%s' no está en la lista de capacidades del nodo", intent)
}

// GetHierarchySnapshot retorna la lista de todos los perfiles de nodo
func (m *NodeHierarchyManager) GetHierarchySnapshot() []*NodeProfile {
	m.mu.RLock()
	defer m.mu.RUnlock()

	list := make([]*NodeProfile, 0, len(m.profiles))
	for _, p := range m.profiles {
		pCopy := *p
		list = append(list, &pCopy)
	}
	return list
}
