package l1

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"ipvn7/pkg/l0"
)

// CapabilityProfile representa las capacidades y atributos firmados de un nodo (1.md Sección 1 y 7)
type CapabilityProfile struct {
	DID         string    `json:"did"`
	Tags        []string  `json:"tags"`        // ej. ["geo/chile", "domain/compute", "service/storage"]
	Description string    `json:"description"` // Descripción humana
	Capacity    float64   `json:"capacity"`    // Capacidad demostrada (0.0 a 100.0)
	UpdatedAt   time.Time `json:"updated_at"`
	Signature   []byte    `json:"signature"`   // Firma Ed25519 del propietario sobre el resumen de tags
}

// ComputeHash genera el resumen canónico para validación criptográfica
func (cp *CapabilityProfile) ComputeHash() []byte {
	h := sha256.New()
	h.Write([]byte(cp.DID))
	for _, t := range cp.Tags {
		h.Write([]byte(strings.ToLower(strings.TrimSpace(t))))
	}
	h.Write([]byte(fmt.Sprintf("%.2f", cp.Capacity)))
	return h.Sum(nil)
}

// SemanticCatalog administra el registro silencioso y las consultas semánticas por pull
type SemanticCatalog struct {
	mu       sync.RWMutex
	profiles map[string]*CapabilityProfile // DID -> Perfil
	tagIndex map[string]map[string]bool    // Tag normalizado -> Conjunto de DIDs
}

// NewSemanticCatalog inicializa el catálogo semántico de malla
func NewSemanticCatalog() *SemanticCatalog {
	return &SemanticCatalog{
		profiles: make(map[string]*CapabilityProfile),
		tagIndex: make(map[string]map[string]bool),
	}
}

// RegisterProfile valida e indexa un perfil de capacidades firmado
func (sc *SemanticCatalog) RegisterProfile(profile *CapabilityProfile) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if profile.DID == "" || len(profile.Tags) == 0 {
		return errors.New("DID y al menos un tag son obligatorios")
	}

	// 1. Verificar firma Ed25519 si está presente
	pubKey, err := l0.PublicKeyFromDID(profile.DID)
	if err == nil && len(profile.Signature) > 0 {
		hash := profile.ComputeHash()
		if !l0.VerifySignature(pubKey, hash, profile.Signature) {
			return errors.New("firma criptográfica de capacidades inválida")
		}
	}

	if profile.UpdatedAt.IsZero() {
		profile.UpdatedAt = time.Now()
	}

	// 2. Limpiar indexación anterior si ya existía
	if old, exists := sc.profiles[profile.DID]; exists {
		for _, oldTag := range old.Tags {
			cleanTag := strings.ToLower(strings.TrimSpace(oldTag))
			if set, ok := sc.tagIndex[cleanTag]; ok {
				delete(set, profile.DID)
			}
		}
	}

	// 3. Registrar nuevo perfil e indexar tags
	sc.profiles[profile.DID] = profile
	for _, tag := range profile.Tags {
		cleanTag := strings.ToLower(strings.TrimSpace(tag))
		if _, ok := sc.tagIndex[cleanTag]; !ok {
			sc.tagIndex[cleanTag] = make(map[string]bool)
		}
		sc.tagIndex[cleanTag][profile.DID] = true
	}

	return nil
}

// SignAndRegister crea, firma y registra un perfil usando la identidad soberana
func (sc *SemanticCatalog) SignAndRegister(id *l0.Identity, tags []string, desc string, capacity float64) (*CapabilityProfile, error) {
	cleanTags := make([]string, len(tags))
	for i, t := range tags {
		cleanTags[i] = strings.ToLower(strings.TrimSpace(t))
	}

	cp := &CapabilityProfile{
		DID:         id.DID(),
		Tags:        cleanTags,
		Description: desc,
		Capacity:    capacity,
		UpdatedAt:   time.Now(),
	}

	hash := cp.ComputeHash()
	cp.Signature = id.Sign(hash)

	if err := sc.RegisterProfile(cp); err != nil {
		return nil, err
	}

	return cp, nil
}

// QueryPull realiza una búsqueda por intención (pull) requiriendo la intersección de tags
func (sc *SemanticCatalog) QueryPull(requiredTags []string) []*CapabilityProfile {
	sc.mu.RLock()
	defer sc.mu.RUnlock()

	if len(requiredTags) == 0 {
		res := make([]*CapabilityProfile, 0, len(sc.profiles))
		for _, p := range sc.profiles {
			res = append(res, p)
		}
		return res
	}

	// Intersección de conjuntos sobre los tags requeridos
	var matchingDIDs map[string]bool

	for i, tag := range requiredTags {
		cleanTag := strings.ToLower(strings.TrimSpace(tag))
		didsWithTag, exists := sc.tagIndex[cleanTag]
		if !exists || len(didsWithTag) == 0 {
			return []*CapabilityProfile{} // Si un tag no tiene ningún par, la intersección es vacía
		}

		if i == 0 {
			matchingDIDs = make(map[string]bool)
			for did := range didsWithTag {
				matchingDIDs[did] = true
			}
		} else {
			for did := range matchingDIDs {
				if !didsWithTag[did] {
					delete(matchingDIDs, did)
				}
			}
		}
	}

	results := make([]*CapabilityProfile, 0, len(matchingDIDs))
	for did := range matchingDIDs {
		if p, ok := sc.profiles[did]; ok {
			results = append(results, p)
		}
	}

	return results
}

// TotalProfiles retorna la cantidad de perfiles registrados
func (sc *SemanticCatalog) TotalProfiles() int {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return len(sc.profiles)
}

// Stats retorna estadísticas agregadas del catálogo
func (sc *SemanticCatalog) Stats() map[string]interface{} {
	sc.mu.RLock()
	defer sc.mu.RUnlock()
	return map[string]interface{}{
		"total_profiles": len(sc.profiles),
		"indexed_tags":   len(sc.tagIndex),
	}
}

// Ensure unused package import doesn't error
var _ = hex.EncodeToString
