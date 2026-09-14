package l1

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Constantes de alcance topológico (Scoping de radio de explosión)
const (
	ScopeLocal  uint8 = 1 // 1 salto físico directo (Wi-Fi, BLE, LAN inmediata)
	ScopeMesh   uint8 = 2 // Malla conocida y pares directos
	ScopeGlobal uint8 = 3 // Anillos completos de la red
)

// CapabilityRecord representa una capacidad o servicio ofrecido por un nodo soberano
type CapabilityRecord struct {
	ServiceTag   string    `json:"service_tag"` // ej: "print/raw", "storage/dag", "compute/wasm"
	ProviderDID  string    `json:"provider_did"`
	Endpoint     string    `json:"endpoint"`
	CapacityMbps float64   `json:"capacity_mbps"`
	RegisteredAt time.Time `json:"registered_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// IntentQuery representa una consulta semántica por demanda (Pull en silencio de red)
type IntentQuery struct {
	TargetTag    string `json:"target_tag"`
	RequiredMbps float64 `json:"required_mbps"`
	Scope        uint8  `json:"scope"`
	MaxHopLimit  int    `json:"max_hop_limit"`
	RequesterDID string `json:"requester_did"`
}

// SemanticRegistry gestiona las capacidades locales y la resolución de intenciones silenciosa
type SemanticRegistry struct {
	mu           sync.RWMutex
	localDID     string
	capabilities map[string][]*CapabilityRecord
}

// NewSemanticRegistry inicializa el registro semántico
func NewSemanticRegistry(localDID string) *SemanticRegistry {
	return &SemanticRegistry{
		localDID:     localDID,
		capabilities: make(map[string][]*CapabilityRecord),
	}
}

// RegisterCapability registra silenciosamente una capacidad propia o de un par vecino
func (sr *SemanticRegistry) RegisterCapability(tag, providerDID, endpoint string, capacityMbps float64, ttl time.Duration) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	normalizedTag := strings.ToLower(strings.TrimSpace(tag))
	rec := &CapabilityRecord{
		ServiceTag:   normalizedTag,
		ProviderDID:  providerDID,
		Endpoint:     endpoint,
		CapacityMbps: capacityMbps,
		RegisteredAt: time.Now(),
		ExpiresAt:    time.Now().Add(ttl),
	}

	// Evitar duplicados del mismo proveedor
	list := sr.capabilities[normalizedTag]
	updated := false
	for i, existing := range list {
		if existing.ProviderDID == providerDID {
			list[i] = rec
			updated = true
			break
		}
	}
	if !updated {
		sr.capabilities[normalizedTag] = append(list, rec)
	}
}

// QueryIntent resuelve una intención semántica bajo demanda sin emitir broadcast ruidoso
func (sr *SemanticRegistry) QueryIntent(query IntentQuery) (*CapabilityRecord, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	normalizedTag := strings.ToLower(strings.TrimSpace(query.TargetTag))
	candidates, found := sr.capabilities[normalizedTag]
	if !found || len(candidates) == 0 {
		return nil, fmt.Errorf("no se encontraron proveedores para la capacidad requerida '%s'", query.TargetTag)
	}

	now := time.Now()
	var bestCandidate *CapabilityRecord

	for _, cand := range candidates {
		if cand.ExpiresAt.Before(now) {
			continue // Expirado
		}
		if cand.CapacityMbps < query.RequiredMbps {
			continue // No cumple capacidad requerida
		}

		if bestCandidate == nil || cand.CapacityMbps > bestCandidate.CapacityMbps {
			bestCandidate = cand
		}
	}

	if bestCandidate == nil {
		return nil, errors.New("no hay candidatos activos que cumplan con la capacidad requerida")
	}

	return bestCandidate, nil
}

// ListLocalCapabilities lista todas las capacidades actualmente registradas
func (sr *SemanticRegistry) ListLocalCapabilities() []*CapabilityRecord {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	res := make([]*CapabilityRecord, 0)
	now := time.Now()
	for _, list := range sr.capabilities {
		for _, rec := range list {
			if rec.ExpiresAt.After(now) {
				res = append(res, rec)
			}
		}
	}
	return res
}
