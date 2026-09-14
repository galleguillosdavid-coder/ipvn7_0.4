package l1

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"ipvn7/pkg/l0"
)

// Vouch representa un aval criptográfico firmado entre dos identidades soberanas (Dimensión 9)
type Vouch struct {
	IssuerDID  string    `json:"issuer_did"`  // DID del emisor que avala
	SubjectDID string    `json:"subject_did"` // DID del par avalado
	TrustLevel float64   `json:"trust_level"` // Grado de confianza (0.0 a 1.0)
	Reason     string    `json:"reason"`      // Motivo mnemotécnico del aval
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Signature  []byte    `json:"signature"` // Firma Ed25519 sobre el hash del aval
}

// ComputeHash genera el identificador canónico del aval
func (v *Vouch) ComputeHash() []byte {
	h := sha256.New()
	h.Write([]byte(v.IssuerDID))
	h.Write([]byte(v.SubjectDID))
	h.Write([]byte(fmt.Sprintf("%.4f", v.TrustLevel)))
	h.Write([]byte(v.Reason))
	h.Write([]byte(fmt.Sprintf("%d", v.IssuedAt.Unix())))
	h.Write([]byte(fmt.Sprintf("%d", v.ExpiresAt.Unix())))
	return h.Sum(nil)
}

// WebOfTrust gestiona el grafo social descentralizado y el cálculo de reputación atenuada
type WebOfTrust struct {
	mu           sync.RWMutex
	vouches      map[string][]*Vouch // IssuerDID -> Lista de avales otorgados
	subjectIndex map[string][]*Vouch // SubjectDID -> Lista de avales recibidos
	attenuation  float64             // Coeficiente de pérdida por salto (0.85 = -15% por salto)
	maxHops      int                 // Límite estricto de 3 grados de separación
}

// NewWebOfTrust inicializa la red de confianza descentralizada
func NewWebOfTrust() *WebOfTrust {
	return &WebOfTrust{
		vouches:      make(map[string][]*Vouch),
		subjectIndex: make(map[string][]*Vouch),
		attenuation:  0.85, // -15% de confianza por cada grado de distancia
		maxHops:      3,    // Máximo 3 saltos
	}
}

// SignAndIssueVouch emite y firma un aval utilizando una identidad soberana
func (wot *WebOfTrust) SignAndIssueVouch(id *l0.Identity, subjectDID string, trustLevel float64, reason string, duration time.Duration) (*Vouch, error) {
	if trustLevel < 0.0 || trustLevel > 1.0 {
		return nil, errors.New("trust_level debe estar en el rango [0.0, 1.0]")
	}

	now := time.Now()
	v := &Vouch{
		IssuerDID:  id.DID(),
		SubjectDID: subjectDID,
		TrustLevel: trustLevel,
		Reason:     reason,
		IssuedAt:   now,
		ExpiresAt:  now.Add(duration),
	}

	hash := v.ComputeHash()
	v.Signature = id.Sign(hash)

	if err := wot.AddVouch(v); err != nil {
		return nil, err
	}

	return v, nil
}

// AddVouch valida e indexa un aval firmado criptográficamente
func (wot *WebOfTrust) AddVouch(v *Vouch) error {
	wot.mu.Lock()
	defer wot.mu.Unlock()

	// 1. Verificar validez temporal
	if !v.ExpiresAt.IsZero() && time.Now().After(v.ExpiresAt) {
		return errors.New("el aval ha expirado")
	}

	// 2. Verificar firma digital Ed25519 del emisor
	pubKey, err := l0.PublicKeyFromDID(v.IssuerDID)
	if err != nil {
		return fmt.Errorf("issuer_did inválido: %w", err)
	}

	hash := v.ComputeHash()
	if !l0.VerifySignature(pubKey, hash, v.Signature) {
		return errors.New("firma criptográfica Ed25519 del aval es inválida")
	}

	// 3. Indexar en el grafo
	wot.vouches[v.IssuerDID] = append(wot.vouches[v.IssuerDID], v)
	wot.subjectIndex[v.SubjectDID] = append(wot.subjectIndex[v.SubjectDID], v)

	return nil
}

// CalculateReputation calcula la reputación atenuada (0 a 100) desde rootDID hacia targetDID usando BFS
func (wot *WebOfTrust) CalculateReputation(rootDID string, targetDID string) (float64, int) {
	if rootDID == targetDID {
		return 100.0, 0 // Confianza absoluta en uno mismo
	}

	wot.mu.RLock()
	defer wot.mu.RUnlock()

	// BFS con seguimiento de distancia en saltos
	type qItem struct {
		did        string
		dist       int
		trustAccum float64
	}

	queue := []qItem{{did: rootDID, dist: 0, trustAccum: 1.0}}
	visited := make(map[string]bool)
	visited[rootDID] = true

	bestScore := 0.0
	minHops := -1

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if curr.did == targetDID {
			// Calcular atenuación geométrica: trustAccum * (0.85^dist) * 100
			attenuated := curr.trustAccum * math.Pow(wot.attenuation, float64(curr.dist)) * 100.0
			if attenuated > bestScore {
				bestScore = attenuated
				minHops = curr.dist
			}
			continue
		}

		if curr.dist >= wot.maxHops {
			continue
		}

		for _, vouch := range wot.vouches[curr.did] {
			// Ignorar avales expirados
			if !vouch.ExpiresAt.IsZero() && time.Now().After(vouch.ExpiresAt) {
				continue
			}

			if !visited[vouch.SubjectDID] {
				visited[vouch.SubjectDID] = true
				nextTrust := curr.trustAccum * vouch.TrustLevel
				queue = append(queue, qItem{
					did:        vouch.SubjectDID,
					dist:       curr.dist + 1,
					trustAccum: nextTrust,
				})
			}
		}
	}

	return math.Round(bestScore*100) / 100, minHops
}

// GetVouchesFor retorna los avales recibidos por un par
func (wot *WebOfTrust) GetVouchesFor(subjectDID string) []*Vouch {
	wot.mu.RLock()
	defer wot.mu.RUnlock()

	list, ok := wot.subjectIndex[subjectDID]
	if !ok {
		return nil
	}
	res := make([]*Vouch, len(list))
	copy(res, list)
	return res
}

// GetAllVouches retorna todos los avales registrados en el grafo
func (wot *WebOfTrust) GetAllVouches() []*Vouch {
	wot.mu.RLock()
	defer wot.mu.RUnlock()

	res := make([]*Vouch, 0)
	for _, list := range wot.vouches {
		res = append(res, list...)
	}
	return res
}

// WoTStats resume el estado del grafo social de confianza
type WoTStats struct {
	TotalVouches   int `json:"total_vouches"`
	UniqueIssuers  int `json:"unique_issuers"`
	UniqueSubjects int `json:"unique_subjects"`
}

// Stats genera la instantánea de métricas del WoT
func (wot *WebOfTrust) Stats() WoTStats {
	wot.mu.RLock()
	defer wot.mu.RUnlock()

	return WoTStats{
		TotalVouches:   len(wot.GetAllVouches()),
		UniqueIssuers:  len(wot.vouches),
		UniqueSubjects: len(wot.subjectIndex),
	}
}

// Ensure unused package import doesn't error
var _ = hex.EncodeToString
