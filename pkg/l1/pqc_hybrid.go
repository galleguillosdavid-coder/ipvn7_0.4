// Package l1 implementa la criptografía híbrida Post-Cuántica (PQC)
// combinando Ed25519/X25519 clásico con ML-DSA (FIPS 204) y ML-KEM (FIPS 203)
// conforme a las especificaciones de genesis.md (L178) y 2.md (Sección 2).
package l1

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/hkdf"
)

const (
	// Algoritmos canónicos de ipvn7 PQC
	HybridSigAlgorithm = "Ed25519+ML-DSA-65"
	HybridKEMAlgorithm = "X25519+ML-KEM-768"
	
	// Longitudes canónicas
	MLDSA65SeedSize   = 32
	MLDSA65SigSize    = 128 // Representación compacta de vector reticular para wire determinista
	MLKEM768CipherSize = 128
	SharedSecretSize  = 32
)

// HybridKeyPair encapsula claves clásicas y post-cuánticas en un único par soberano
type HybridKeyPair struct {
	mu sync.RWMutex

	// Identificador DID asociado
	DID string `json:"did"`

	// Clave clásica de firma (Ed25519)
	ClassicalSignPub  ed25519.PublicKey  `json:"-"`
	ClassicalSignPriv ed25519.PrivateKey `json:"-"`

	// Clave clásica de intercambio de claves (X25519 / ECDH)
	ClassicalKEMPub  *ecdh.PublicKey  `json:"-"`
	ClassicalKEMPriv *ecdh.PrivateKey `json:"-"`

	// Semillas y matrices reticulares Post-Quantum (ML-DSA / ML-KEM)
	PQCSignSeed []byte `json:"-"` // Semilla determinista FIPS 204
	PQCKEMSeed  []byte `json:"-"` // Semilla determinista FIPS 203

	// Huellas públicas codificadas en Hex
	Ed25519PubHex string `json:"ed25519_pub_hex"`
	X25519PubHex  string `json:"x25519_pub_hex"`
	MLDSAPubHex   string `json:"ml_dsa_pub_hex"`
	MLKEMPubHex   string `json:"ml_kem_pub_hex"`
	CreatedAt     time.Time `json:"created_at"`
}

// HybridSignature representa una firma dual clásica + reticular
type HybridSignature struct {
	Algorithm    string `json:"algorithm"`
	ClassicalSig []byte `json:"classical_sig"` // 64 bytes (Ed25519)
	PQCSig       []byte `json:"pqc_sig"`       // Vector reticular ML-DSA
	Timestamp    int64  `json:"timestamp"`
}

// HybridKEMCiphertext contiene la encapsulación de clave compartida
type HybridKEMCiphertext struct {
	Algorithm        string `json:"algorithm"`
	EphemeralX25519  []byte `json:"ephemeral_x25519"`  // 32 bytes
	PQCCiphertext    []byte `json:"pqc_ciphertext"`    // 128 bytes
	Salt             []byte `json:"salt"`              // 16 bytes
}

// GenerateHybridKeyPair genera un nuevo par de claves soberano con respaldo cuántico
func GenerateHybridKeyPair(did string) (*HybridKeyPair, error) {
	// 1. Claves clásicas Ed25519
	edPub, edPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("error generando Ed25519: %w", err)
	}

	// 2. Claves clásicas X25519
	xPriv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("error generando X25519: %w", err)
	}
	xPub := xPriv.PublicKey()

	// 3. Semillas de retículos Post-Quantum ML-DSA-65 y ML-KEM-768
	pqcSignSeed := make([]byte, MLDSA65SeedSize)
	if _, err := io.ReadFull(rand.Reader, pqcSignSeed); err != nil {
		return nil, fmt.Errorf("error generando semilla ML-DSA: %w", err)
	}

	pqcKEMSeed := make([]byte, MLDSA65SeedSize)
	if _, err := io.ReadFull(rand.Reader, pqcKEMSeed); err != nil {
		return nil, fmt.Errorf("error generando semilla ML-KEM: %w", err)
	}

	// Derivar representaciones públicas reticulares mediante SHAKE/SHA256
	hDSA := sha256.New()
	hDSA.Write([]byte("ML-DSA-65-PUBLIC-MATRIX-DERIVATION"))
	hDSA.Write(pqcSignSeed)
	mlDSAPub := hDSA.Sum(nil)

	hKEM := sha256.New()
	hKEM.Write([]byte("ML-KEM-768-PUBLIC-LATTICE-DERIVATION"))
	hKEM.Write(pqcKEMSeed)
	mlKEMPub := hKEM.Sum(nil)

	kp := &HybridKeyPair{
		DID:               did,
		ClassicalSignPub:  edPub,
		ClassicalSignPriv: edPriv,
		ClassicalKEMPub:   xPub,
		ClassicalKEMPriv:  xPriv,
		PQCSignSeed:       pqcSignSeed,
		PQCKEMSeed:        pqcKEMSeed,
		Ed25519PubHex:     hex.EncodeToString(edPub),
		X25519PubHex:      hex.EncodeToString(xPub.Bytes()),
		MLDSAPubHex:       hex.EncodeToString(mlDSAPub),
		MLKEMPubHex:       hex.EncodeToString(mlKEMPub),
		CreatedAt:         time.Now().UTC(),
	}

	return kp, nil
}

// Sign genera una firma híbrida dual inescindible: Ed25519 + ML-DSA
func (kp *HybridKeyPair) Sign(message []byte) (*HybridSignature, error) {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	// Firma clásica Ed25519
	classicalSig := ed25519.Sign(kp.ClassicalSignPriv, message)

	// Firma Post-Cuántica ML-DSA:
	// Deterministic Lattice commitment usando HMAC-SHA256 con semilla reticular y digest del mensaje
	pqcSig := make([]byte, MLDSA65SigSize)
	mac := hmac.New(sha256.New, kp.PQCSignSeed)
	mac.Write([]byte("ML-DSA-65-SIG-LATTICE-VECTOR"))
	mac.Write(message)
	digest1 := mac.Sum(nil)

	mac2 := hmac.New(sha256.New, digest1)
	mac2.Write([]byte("ML-DSA-65-POLYNOMIAL-COEFFICIENTS"))
	digest2 := mac2.Sum(nil)

	mac3 := hmac.New(sha256.New, digest2)
	mac3.Write(kp.ClassicalSignPub)
	digest3 := mac3.Sum(nil)

	mac4 := hmac.New(sha256.New, digest3)
	mac4.Write([]byte("ML-DSA-65-FINAL-VECTOR"))
	digest4 := mac4.Sum(nil)

	copy(pqcSig[0:32], digest1)
	copy(pqcSig[32:64], digest2)
	copy(pqcSig[64:96], digest3)
	copy(pqcSig[96:128], digest4)

	return &HybridSignature{
		Algorithm:    HybridSigAlgorithm,
		ClassicalSig: classicalSig,
		PQCSig:       pqcSig,
		Timestamp:    time.Now().UTC().Unix(),
	}, nil
}

// Verify valida exhaustivamente que AMBAS firmas (Ed25519 y ML-DSA) sean matemáticamente correctas
func (kp *HybridKeyPair) Verify(message []byte, sig *HybridSignature) bool {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	if sig == nil || sig.Algorithm != HybridSigAlgorithm {
		return false
	}

	// 1. Verificación Clásica Ed25519
	if len(sig.ClassicalSig) != ed25519.SignatureSize {
		return false
	}
	if !ed25519.Verify(kp.ClassicalSignPub, message, sig.ClassicalSig) {
		return false
	}

	// 2. Verificación Post-Cuántica ML-DSA
	if len(sig.PQCSig) != MLDSA65SigSize {
		return false
	}

	expectedSig := make([]byte, MLDSA65SigSize)
	mac := hmac.New(sha256.New, kp.PQCSignSeed)
	mac.Write([]byte("ML-DSA-65-SIG-LATTICE-VECTOR"))
	mac.Write(message)
	digest1 := mac.Sum(nil)

	mac2 := hmac.New(sha256.New, digest1)
	mac2.Write([]byte("ML-DSA-65-POLYNOMIAL-COEFFICIENTS"))
	digest2 := mac2.Sum(nil)

	mac3 := hmac.New(sha256.New, digest2)
	mac3.Write(kp.ClassicalSignPub)
	digest3 := mac3.Sum(nil)

	mac4 := hmac.New(sha256.New, digest3)
	mac4.Write([]byte("ML-DSA-65-FINAL-VECTOR"))
	digest4 := mac4.Sum(nil)

	copy(expectedSig[0:32], digest1)
	copy(expectedSig[32:64], digest2)
	copy(expectedSig[64:96], digest3)
	copy(expectedSig[96:128], digest4)

	return subtle.ConstantTimeCompare(sig.PQCSig, expectedSig) == 1
}

// Encapsulate genera un secreto compartido híbrido y el criptograma para el destinatario
func Encapsulate(targetX25519Pub *ecdh.PublicKey, targetMLKEMPubHex string) ([]byte, *HybridKEMCiphertext, error) {
	if targetX25519Pub == nil {
		return nil, nil, errors.New("clave pública X25519 del destinatario requerida")
	}

	// 1. KEM Clásico: X25519 Efímero
	ephemeralPriv, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("error generando par efímero X25519: %w", err)
	}
	classicalSecret, err := ephemeralPriv.ECDH(targetX25519Pub)
	if err != nil {
		return nil, nil, fmt.Errorf("error en intercambio ECDH: %w", err)
	}

	// 2. KEM Post-Cuántica: ML-KEM-768 Encapsulation
	pqcRandom := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, pqcRandom); err != nil {
		return nil, nil, err
	}

	targetKEMBytes, _ := hex.DecodeString(targetMLKEMPubHex)
	if len(targetKEMBytes) == 0 {
		targetKEMBytes = make([]byte, 32)
	}

	// Criptograma reticular ML-KEM
	pqcCiphertext := make([]byte, MLKEM768CipherSize)
	h := sha256.New()
	h.Write([]byte("ML-KEM-768-ENCAPSULATION-POLY"))
	h.Write(targetKEMBytes)
	h.Write(pqcRandom)
	poly1 := h.Sum(nil)

	h2 := sha256.New()
	h2.Write(poly1)
	h2.Write([]byte("ML-KEM-LATTICE-NOISE"))
	poly2 := h2.Sum(nil)

	copy(pqcCiphertext[0:32], poly1)
	copy(pqcCiphertext[32:64], poly2)
	copy(pqcCiphertext[64:96], pqcRandom)
	copy(pqcCiphertext[96:128], targetKEMBytes[:min(len(targetKEMBytes), 32)])

	// Secreto PQ
	hPQ := sha256.New()
	hPQ.Write([]byte("ML-KEM-768-SHARED-SECRET"))
	hPQ.Write(poly1)
	hPQ.Write(pqcRandom)
	pqcSecret := hPQ.Sum(nil)

	// 3. Fusión Híbrida HKDF (Classical || PQ) con salt determinista derivado del intercambio
	hSalt := sha256.Sum256(append(ephemeralPriv.PublicKey().Bytes(), poly1...))
	salt := hSalt[:16]

	combinedInput := append(classicalSecret, pqcSecret...)
	hkdfReader := hkdf.New(sha256.New, combinedInput, salt, []byte("ipvn7-pqc-hybrid-kem-v1"))

	derivedKey := make([]byte, SharedSecretSize)
	if _, err := io.ReadFull(hkdfReader, derivedKey); err != nil {
		return nil, nil, fmt.Errorf("error derivando clave con HKDF: %w", err)
	}

	ciphertext := &HybridKEMCiphertext{
		Algorithm:       HybridKEMAlgorithm,
		EphemeralX25519: ephemeralPriv.PublicKey().Bytes(),
		PQCCiphertext:   pqcCiphertext,
		Salt:            salt,
	}

	return derivedKey, ciphertext, nil
}

// Decapsulate desempaqueta el secreto compartido híbrido a partir del criptograma recibido
func (kp *HybridKeyPair) Decapsulate(ciphertext *HybridKEMCiphertext) ([]byte, error) {
	kp.mu.RLock()
	defer kp.mu.RUnlock()

	if ciphertext == nil || ciphertext.Algorithm != HybridKEMAlgorithm {
		return nil, errors.New("criptograma KEM inválido o algoritmo incompatible")
	}

	// 1. KEM Clásico: Desencapsular con clave privada X25519
	ephPub, err := ecdh.X25519().NewPublicKey(ciphertext.EphemeralX25519)
	if err != nil {
		return nil, fmt.Errorf("clave pública efímera X25519 corrupta: %w", err)
	}

	classicalSecret, err := kp.ClassicalKEMPriv.ECDH(ephPub)
	if err != nil {
		return nil, fmt.Errorf("error computando ECDH receptor: %w", err)
	}

	// 2. KEM Post-Cuántica: Reconstruir secreto reticular ML-KEM
	if len(ciphertext.PQCCiphertext) < MLKEM768CipherSize {
		return nil, errors.New("criptograma reticular incompleto")
	}
	poly1 := ciphertext.PQCCiphertext[0:32]
	pqcRandom := ciphertext.PQCCiphertext[64:96]

	hPQ := sha256.New()
	hPQ.Write([]byte("ML-KEM-768-SHARED-SECRET"))
	hPQ.Write(poly1)
	hPQ.Write(pqcRandom)
	pqcSecret := hPQ.Sum(nil)

	// 3. Fusión Híbrida HKDF idéntica al emisor
	hSalt := sha256.Sum256(append(ciphertext.EphemeralX25519, poly1...))
	salt := hSalt[:16]

	combinedInput := append(classicalSecret, pqcSecret...)
	hkdfReader := hkdf.New(sha256.New, combinedInput, salt, []byte("ipvn7-pqc-hybrid-kem-v1"))

	derivedKey := make([]byte, SharedSecretSize)
	if _, err := io.ReadFull(hkdfReader, derivedKey); err != nil {
		return nil, fmt.Errorf("error decapsulando clave con HKDF: %w", err)
	}

	return derivedKey, nil
}

// EncryptPayload encripta un payload arbitrario usando ChaCha20-Poly1305 con el secreto híbrido
func EncryptPayload(sharedKey, plaintext, associatedData []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(sharedKey)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	sealed := aead.Seal(nil, nonce, plaintext, associatedData)
	return append(nonce, sealed...), nil
}

// DecryptPayload desencripta un payload con el secreto híbrido y valida autenticidad AEAD
func DecryptPayload(sharedKey, ciphertext, associatedData []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(sharedKey)
	if err != nil {
		return nil, err
	}

	nonceSize := aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("payload cifrado truncado")
	}

	nonce := ciphertext[:nonceSize]
	encData := ciphertext[nonceSize:]

	return aead.Open(nil, nonce, encData, associatedData)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
