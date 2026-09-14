package l1

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

// AuditEntry representa un evento auditado criptográficamente inmutable (1.md Sección 22)
type AuditEntry struct {
	Index      uint64                 `json:"index"`
	Timestamp  time.Time              `json:"timestamp"`
	EventType  string                 `json:"event_type"` // ej. "ZTNA_DROP", "FSM_TRANSITION", "FAILOVER", "COPILOT_HEAL"
	ActorDID   string                 `json:"actor_did"`
	Details    map[string]interface{} `json:"details"`
	PrevHash   string                 `json:"prev_hash"`
	EntryHash  string                 `json:"entry_hash"`
}

// ComputeHash calcula el hash criptográfico SHA-256 canónico de la entrada
func (e *AuditEntry) ComputeHash() string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%d", e.Index)))
	h.Write([]byte(fmt.Sprintf("%d", e.Timestamp.UnixNano())))
	h.Write([]byte(e.EventType))
	h.Write([]byte(e.ActorDID))
	h.Write([]byte(e.PrevHash))

	detailsJSON, _ := json.Marshal(e.Details)
	h.Write(detailsJSON)

	return hex.EncodeToString(h.Sum(nil))
}

// MerkleAuditLog gestiona la bitácora inalterable de auditoría de seguridad
type MerkleAuditLog struct {
	mu      sync.RWMutex
	entries []*AuditEntry
}

// NewMerkleAuditLog inicializa la bitácora criptográfica con bloque génesis
func NewMerkleAuditLog() *MerkleAuditLog {
	genesis := &AuditEntry{
		Index:     0,
		Timestamp: time.Now(),
		EventType: "GENESIS_AUDIT_LOG",
		ActorDID:  "did:ipvn7:genesis",
		Details:   map[string]interface{}{"status": "INITIALIZED", "system": "ipvn7-NOS"},
		PrevHash:  "0000000000000000000000000000000000000000000000000000000000000000",
	}
	genesis.EntryHash = genesis.ComputeHash()

	return &MerkleAuditLog{
		entries: []*AuditEntry{genesis},
	}
}

// AppendEvent registra un nuevo evento crítico en la cadena criptográfica
func (mal *MerkleAuditLog) AppendEvent(eventType, actorDID string, details map[string]interface{}) (*AuditEntry, error) {
	mal.mu.Lock()
	defer mal.mu.Unlock()

	lastEntry := mal.entries[len(mal.entries)-1]
	newIdx := lastEntry.Index + 1

	entry := &AuditEntry{
		Index:     newIdx,
		Timestamp: time.Now(),
		EventType: eventType,
		ActorDID:  actorDID,
		Details:   details,
		PrevHash:  lastEntry.EntryHash,
	}
	entry.EntryHash = entry.ComputeHash()

	mal.entries = append(mal.entries, entry)
	return entry, nil
}

// VerifyIntegrity audita la cadena completa desde el génesis comprobando hashes y enlaces
func (mal *MerkleAuditLog) VerifyIntegrity() (bool, error) {
	mal.mu.RLock()
	defer mal.mu.RUnlock()

	if len(mal.entries) == 0 {
		return false, errors.New("bitácora vacía")
	}

	for i := 0; i < len(mal.entries); i++ {
		curr := mal.entries[i]

		// 1. Verificar hash canónico del bloque actual
		expectedHash := curr.ComputeHash()
		if curr.EntryHash != expectedHash {
			return false, fmt.Errorf("violación de integridad en índice %d: hash inválido", curr.Index)
		}

		// 2. Verificar enlace hacia el bloque predecesor
		if i > 0 {
			prev := mal.entries[i-1]
			if curr.PrevHash != prev.EntryHash {
				return false, fmt.Errorf("rotura de cadena criptográfica en índice %d: prev_hash no coincide", curr.Index)
			}
		}
	}

	return true, nil
}

// GetRecentEntries retorna los últimos N eventos auditados
func (mal *MerkleAuditLog) GetRecentEntries(limit int) []*AuditEntry {
	mal.mu.RLock()
	defer mal.mu.RUnlock()

	if len(mal.entries) == 0 {
		return []*AuditEntry{}
	}

	start := 0
	if limit > 0 && len(mal.entries) > limit {
		start = len(mal.entries) - limit
	}

	res := make([]*AuditEntry, len(mal.entries)-start)
	copy(res, mal.entries[start:])
	return res
}

// TotalEntries retorna el conteo de eventos registrados
func (mal *MerkleAuditLog) TotalEntries() int {
	mal.mu.RLock()
	defer mal.mu.RUnlock()
	return len(mal.entries)
}
