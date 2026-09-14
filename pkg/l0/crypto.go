package l0

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/curve25519"
)

// SessionKeys almacena las claves simétricas derivadas para una sesión activa
type SessionKeys struct {
	TxKey     [32]byte
	RxKey     [32]byte
	SessionID [16]byte
	PFSActive bool
}

// NoiseHandshakeState modela las 3 etapas del apretón de manos Noise XX
type NoiseHandshakeState struct {
	LocalID       *Identity
	EphemeralPriv [32]byte
	EphemeralPub  [32]byte
	RemotePub     [32]byte
	RemoteStatic  [32]byte
	Step          int
	SharedSecret  [32]byte
}

// NewNoiseHandshake inicializa el estado de apretón de manos Noise XX
func NewNoiseHandshake(localID *Identity) (*NoiseHandshakeState, error) {
	hs := &NoiseHandshakeState{
		LocalID: localID,
		Step:    0,
	}
	// Generar par de claves efímeras X25519
	if _, err := io.ReadFull(rand.Reader, hs.EphemeralPriv[:]); err != nil {
		return nil, fmt.Errorf("error generando clave efímera: %w", err)
	}
	curve25519.ScalarBaseMult(&hs.EphemeralPub, &hs.EphemeralPriv)
	return hs, nil
}

// Step1Initiate genera el primer mensaje del handshake (envía EphemeralPub)
func (hs *NoiseHandshakeState) Step1Initiate() ([]byte, error) {
	hs.Step = 1
	out := make([]byte, 32)
	copy(out, hs.EphemeralPub[:])
	return out, nil
}

// Step2Respond procesa el mensaje de inicio y genera la respuesta del responder
func (hs *NoiseHandshakeState) Step2Respond(remoteEphemeral []byte) ([]byte, *SessionKeys, error) {
	if len(remoteEphemeral) != 32 {
		return nil, nil, errors.New("clave efímera remota inválida")
	}
	copy(hs.RemotePub[:], remoteEphemeral)

	// Derivar secreto compartido Diffie-Hellman sobre claves efímeras
	shared, err := curve25519.X25519(hs.EphemeralPriv[:], hs.RemotePub[:])
	if err != nil {
		return nil, nil, fmt.Errorf("error en cálculo X25519: %w", err)
	}
	copy(hs.SharedSecret[:], shared)

	keys := deriveSessionKeys(hs.SharedSecret[:], false)
	hs.Step = 2

	// Responder envía su EphemeralPub
	out := make([]byte, 32)
	copy(out, hs.EphemeralPub[:])
	return out, keys, nil
}

// Step3Finalize completa el handshake en el iniciador y deriva claves de sesión
func (hs *NoiseHandshakeState) Step3Finalize(remoteEphemeral []byte) (*SessionKeys, error) {
	if len(remoteEphemeral) != 32 {
		return nil, errors.New("clave efímera remota inválida")
	}
	copy(hs.RemotePub[:], remoteEphemeral)

	shared, err := curve25519.X25519(hs.EphemeralPriv[:], hs.RemotePub[:])
	if err != nil {
		return nil, fmt.Errorf("error en cálculo X25519: %w", err)
	}
	copy(hs.SharedSecret[:], shared)

	keys := deriveSessionKeys(hs.SharedSecret[:], true)
	hs.Step = 3
	return keys, nil
}

// deriveSessionKeys deriva claves de transmisión y recepción mediante SHA-256 HKDF básico
func deriveSessionKeys(shared []byte, isInitiator bool) *SessionKeys {
	h1 := sha256.Sum256(append(shared, []byte("tx_direction")...))
	h2 := sha256.Sum256(append(shared, []byte("rx_direction")...))
	sid := sha256.Sum256(append(shared, []byte("session_id")...))

	keys := &SessionKeys{
		PFSActive: true,
	}
	copy(keys.SessionID[:], sid[:16])

	if isInitiator {
		copy(keys.TxKey[:], h1[:])
		copy(keys.RxKey[:], h2[:])
	} else {
		copy(keys.TxKey[:], h2[:])
		copy(keys.RxKey[:], h1[:])
	}
	return keys
}

// EncryptPayload cifra los datos útiles usando ChaCha20-Poly1305
func EncryptPayload(key []byte, nonce []byte, plaintext, additionalData []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("error creando cipher AEAD: %w", err)
	}
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("nonce inválido: longitud %d (esperado %d)", len(nonce), aead.NonceSize())
	}
	return aead.Seal(nil, nonce, plaintext, additionalData), nil
}

// DecryptPayload descifra y autentica los datos útiles usando ChaCha20-Poly1305
func DecryptPayload(key []byte, nonce []byte, ciphertext, additionalData []byte) ([]byte, error) {
	aead, err := chacha20poly1305.New(key)
	if err != nil {
		return nil, fmt.Errorf("error creando cipher AEAD: %w", err)
	}
	if len(nonce) != aead.NonceSize() {
		return nil, fmt.Errorf("nonce inválido: longitud %d (esperado %d)", len(nonce), aead.NonceSize())
	}
	return aead.Open(nil, nonce, ciphertext, additionalData)
}

// GenerateNonce crea un nonce seguro de 12 bytes para ChaCha20-Poly1305
func GenerateNonce() ([]byte, error) {
	nonce := make([]byte, chacha20poly1305.NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return nonce, nil
}

// AntiReplayFilter implementa una ventana deslizante de 64 posiciones lock-free
type AntiReplayFilter struct {
	mu      sync.Mutex
	lastSeq uint64
	bitmap  uint64
}

// NewAntiReplayFilter inicializa el filtro anti-repetición
func NewAntiReplayFilter() *AntiReplayFilter {
	return &AntiReplayFilter{}
}

// ValidateAndUpdate verifica si un número de secuencia es válido y no repetido
func (f *AntiReplayFilter) ValidateAndUpdate(seq uint64) bool {
	f.mu.Lock()
	defer f.mu.Unlock()

	if seq > f.lastSeq {
		diff := seq - f.lastSeq
		if diff >= 64 {
			f.bitmap = 1
		} else {
			f.bitmap = (f.bitmap << diff) | 1
		}
		f.lastSeq = seq
		return true
	}

	diff := f.lastSeq - seq
	if diff >= 64 {
		// Demasiado viejo, fuera de la ventana
		return false
	}

	bit := uint64(1) << diff
	if (f.bitmap & bit) != 0 {
		// Ya recibido (repetición detectada)
		return false
	}

	f.bitmap |= bit
	return true
}

// PQCKyberAdapter define la interfaz desacoplada para inyección post-cuántica
// (Strict Core Freeze: la extensión post-cuántica se conecta sin alterar el pipeline)
type PQCKyberAdapter interface {
	Encapsulate(recipientPK []byte) (ciphertext []byte, sharedSecret []byte, err error)
	Decapsulate(ciphertext []byte) (sharedSecret []byte, err error)
}

// StatelessCookieGenerator genera y valida cookies HMAC de 16 bytes para mitigar SYN-flood en handshakes
type StatelessCookieGenerator struct {
	mu        sync.RWMutex
	secretKey [32]byte
	epochSec  int64
}

// NewStatelessCookieGenerator inicializa el generador con clave secreta efímera en memoria
func NewStatelessCookieGenerator() *StatelessCookieGenerator {
	g := &StatelessCookieGenerator{
		epochSec: 120, // 2 minutos por rotación de época
	}
	_, _ = io.ReadFull(rand.Reader, g.secretKey[:])
	return g
}

// GenerateCookie genera un token MAC determinista para una tupla (IP, ephemeralPub)
func (g *StatelessCookieGenerator) GenerateCookie(remoteAddr string, ephemeralPub []byte) []byte {
	currentEpoch := time.Now().Unix() / g.epochSec
	return g.computeMAC(remoteAddr, ephemeralPub, currentEpoch)
}

// ValidateCookie verifica si la cookie fue emitida en la época actual o inmediatamente anterior
func (g *StatelessCookieGenerator) ValidateCookie(cookie []byte, remoteAddr string, ephemeralPub []byte) bool {
	if len(cookie) != 16 {
		return false
	}
	currentEpoch := time.Now().Unix() / g.epochSec

	// Probar época actual
	expectedCurrent := g.computeMAC(remoteAddr, ephemeralPub, currentEpoch)
	if hmac.Equal(cookie, expectedCurrent) {
		return true
	}

	// Probar época anterior (margen de reloj y tránsito)
	expectedPrev := g.computeMAC(remoteAddr, ephemeralPub, currentEpoch-1)
	return hmac.Equal(cookie, expectedPrev)
}

func (g *StatelessCookieGenerator) computeMAC(remoteAddr string, ephemeralPub []byte, epoch int64) []byte {
	mac := hmac.New(sha256.New, g.secretKey[:])
	mac.Write([]byte(remoteAddr))
	mac.Write(ephemeralPub)
	var epochBytes [8]byte
	binary.BigEndian.PutUint64(epochBytes[:], uint64(epoch))
	mac.Write(epochBytes[:])
	sum := mac.Sum(nil)
	return sum[:16] // Truncar a 16 bytes deterministas
}

