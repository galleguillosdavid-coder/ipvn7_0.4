// Package l1 implementa la arquitectura de Identidad Universal UIN (Universal Intent & Identity Network)
// rescatada de ip7uin_MVP_Spec_v1.1.docx (Secciones 2, 3 y 4) y de IPv7_UIN_Arquitectura_Unificada_v4.docx.
// Principio: Desacoplar la identidad soberana de la entidad (root_id de 256 bits) de las claves
// operativas efímeras y rotativas mediante registros de vinculación criptográfica (BindingRecord).
// Respeta estrictamente el Core Freeze (pkg/l0/ permanece inmutable).
package l1

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Modos de Identidad según ip7uin spec
type IdentityMode string

const (
	IdentityModeLegacy IdentityMode = "legacy" // Basado únicamente en clave pública directa
	IdentityModeHybrid IdentityMode = "hybrid" // Precedencia a BindingRecord válido, fallback a clave directa
	IdentityModeRoot   IdentityMode = "root"   // Estricto: solo root_id autenticado con BindingRecord válido
)

// Constantes de Algoritmo en BindingRecord
const (
	BindingAlgoEd25519 = 0x01
	BindingAlgoMLDSA65 = 0x02
)

// Ámbitos de autorización (Scope)
const (
	BindingScopeGeneral    = 0x00
	BindingScopeTelemetry  = 0x01
	BindingScopeControl    = 0x02
	BindingScopeAIAgent    = 0x03
	BindingScopeSettlement = 0x04
)

// BindingRecord representa la trama binaria de vinculación canónica (Wire Format v1)
type BindingRecord struct {
	Version    uint8     `json:"version"`      // Versión wire (1)
	RootID     [32]byte  `json:"root_id"`      // Identificador de raíz soberana de 256 bits
	KeyID      [16]byte  `json:"key_id"`       // Identificador único de clave subordinada
	Algo       uint8     `json:"algo"`         // 1=Ed25519, 2=ML-DSA-65
	Scope      uint8     `json:"scope"`        // Ámbito de autorización
	PublicKey  []byte    `json:"public_key"`   // Clave pública delegada
	ValidFrom  time.Time `json:"valid_from"`   // Inicio de vigencia
	ValidUntil time.Time `json:"valid_until"`  // Expiración
	PrevKeyID  [16]byte  `json:"prev_key_id"`  // Enlace a clave previa (cadena de rotación)
	SigRoot    []byte    `json:"sig_root"`     // Firma criptográfica de la raíz sobre el registro
}

// UINPassport representa el pasaporte digital de la entidad
type UINPassport struct {
	RootIDHex       string           `json:"root_id_hex"`
	EntityDID       string           `json:"entity_did"`
	Mode            IdentityMode     `json:"mode"`
	ActiveBindings  []*BindingRecord `json:"active_bindings"`
	RevokedBindings []string         `json:"revoked_bindings"`
	CreatedAt       time.Time        `json:"created_at"`
}

// UINIdentityManager gestiona la identidad de entidad, delegaciones y revocaciones
type UINIdentityManager struct {
	mu           sync.RWMutex
	rootPrivKey  ed25519.PrivateKey
	rootPubKey   ed25519.PublicKey
	rootID       [32]byte
	mode         IdentityMode
	bindings     map[string]*BindingRecord // key: hex(KeyID)
	revocations  map[string]time.Time      // key: hex(KeyID) -> fecha de revocación
}

// NewUINIdentityManager inicializa el gestor con generación local autónoma de root_id
func NewUINIdentityManager(mode IdentityMode) (*UINIdentityManager, error) {
	if mode == "" {
		mode = IdentityModeHybrid
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("falla generando raíz UIN: %w", err)
	}

	// root_id = SHA-256(root_pubkey || "ip7uin:root:v1")
	h := sha256.New()
	h.Write(pub)
	h.Write([]byte("ip7uin:root:v1"))
	var rootID [32]byte
	copy(rootID[:], h.Sum(nil))

	mgr := &UINIdentityManager{
		rootPrivKey: priv,
		rootPubKey:  pub,
		rootID:      rootID,
		mode:        mode,
		bindings:    make(map[string]*BindingRecord),
		revocations: make(map[string]time.Time),
	}

	return mgr, nil
}

// RootID retorna el identificador canónico de 256 bits
func (m *UINIdentityManager) RootID() [32]byte {
	return m.rootID
}

// RootIDHex retorna la representación hexadecimal del root_id
func (m *UINIdentityManager) RootIDHex() string {
	return hex.EncodeToString(m.rootID[:])
}

// EntityDID deriva el DID canónico de entidad
func (m *UINIdentityManager) EntityDID() string {
	return fmt.Sprintf("did:ipvn7:uin:%s", hex.EncodeToString(m.rootID[:16]))
}

// IssueBinding emite y firma un nuevo BindingRecord para una clave subordinada o agente IA
func (m *UINIdentityManager) IssueBinding(pubKey []byte, algo uint8, scope uint8, duration time.Duration, prevKeyID [16]byte) (*BindingRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var keyID [16]byte
	if _, err := rand.Read(keyID[:]); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	record := &BindingRecord{
		Version:    1,
		RootID:     m.rootID,
		KeyID:      keyID,
		Algo:       algo,
		Scope:      scope,
		PublicKey:  pubKey,
		ValidFrom:  now,
		ValidUntil: now.Add(duration),
		PrevKeyID:  prevKeyID,
	}

	// Firmar registro canónico con la clave privada de la raíz
	payloadToSign := record.canonicalBytes()
	record.SigRoot = ed25519.Sign(m.rootPrivKey, payloadToSign)

	keyHex := hex.EncodeToString(keyID[:])
	m.bindings[keyHex] = record
	return record, nil
}

// RevokeBinding revoca atómicamente un enlace de clave
func (m *UINIdentityManager) RevokeBinding(keyID [16]byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	keyHex := hex.EncodeToString(keyID[:])
	if _, exists := m.bindings[keyHex]; !exists {
		return errors.New("binding no encontrado")
	}

	delete(m.bindings, keyHex)
	m.revocations[keyHex] = time.Now().UTC()
	return nil
}

// ValidateBinding valida criptográficamente la integridad y vigencia de un BindingRecord
func (m *UINIdentityManager) ValidateBinding(rec *BindingRecord) bool {
	if rec == nil || rec.Version != 1 {
		return false
	}

	keyHex := hex.EncodeToString(rec.KeyID[:])

	m.mu.RLock()
	_, revoked := m.revocations[keyHex]
	m.mu.RUnlock()
	if revoked {
		return false
	}

	now := time.Now().UTC()
	if now.Before(rec.ValidFrom) || now.After(rec.ValidUntil) {
		return false
	}

	// Verificar firma contra la clave raíz de la entidad
	payload := rec.canonicalBytes()
	return ed25519.Verify(m.rootPubKey, payload, rec.SigRoot)
}

// GetPassport genera el pasaporte de entidad completo con sus delegaciones activas
func (m *UINIdentityManager) GetPassport() UINPassport {
	m.mu.RLock()
	defer m.mu.RUnlock()

	active := make([]*BindingRecord, 0, len(m.bindings))
	for _, b := range m.bindings {
		bCopy := *b
		active = append(active, &bCopy)
	}

	revoked := make([]string, 0, len(m.revocations))
	for k := range m.revocations {
		revoked = append(revoked, k)
	}

	return UINPassport{
		RootIDHex:       m.RootIDHex(),
		EntityDID:       m.EntityDID(),
		Mode:            m.mode,
		ActiveBindings:  active,
		RevokedBindings: revoked,
		CreatedAt:       time.Now().UTC(),
	}
}

func (rec *BindingRecord) canonicalBytes() []byte {
	buf := make([]byte, 1+32+16+1+1+8+8+16+len(rec.PublicKey))
	offset := 0

	buf[offset] = rec.Version
	offset++

	copy(buf[offset:offset+32], rec.RootID[:])
	offset += 32

	copy(buf[offset:offset+16], rec.KeyID[:])
	offset += 16

	buf[offset] = rec.Algo
	offset++

	buf[offset] = rec.Scope
	offset++

	binary.BigEndian.PutUint64(buf[offset:offset+8], uint64(rec.ValidFrom.Unix()))
	offset += 8

	binary.BigEndian.PutUint64(buf[offset:offset+8], uint64(rec.ValidUntil.Unix()))
	offset += 8

	copy(buf[offset:offset+16], rec.PrevKeyID[:])
	offset += 16

	copy(buf[offset:], rec.PublicKey)
	return buf
}
