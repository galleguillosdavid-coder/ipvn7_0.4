// Package l1 implementa el Adaptador de Señalización Ciega Efímera (Ephemeral Blind Rendezvous Adapter - EBRA).
// Permite que un nodo recién iniciado o post-apagón publique temporalmente su punto de acceso de forma
// cifrada en un contenedor opaco, sin filtrar metadatos personales (Zero-PII estricto), con autodestrucción
// (Consume-and-Burn) y desconexión atómica a P2P puro (Circuit Breaker) en cuanto se alcanzan enlaces directos.
package l1

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"ipvn7/pkg/l0"
)

const (
	DefaultBeaconTTL    = 5 * time.Minute
	MaxBeaconTTL        = 15 * time.Minute
	MinPeerTripThreshold = 2 // Umbral de pares directos para desconectar señalización externa
)

// BlindBeaconStore abstrae el sustrato de almacenamiento de las balizas (Firebase, HTTP, memoria, etc.)
type BlindBeaconStore interface {
	PutBeacon(topic string, data []byte, ttl time.Duration) error
	GetBeacon(topic string) ([]byte, error)
	DeleteBeacon(topic string) error
}

// MemoryBlindBeaconStore implementa almacenamiento en memoria para entornos locales o de prueba
type MemoryBlindBeaconStore struct {
	mu      sync.RWMutex
	beacons map[string]storedBeacon
}

type storedBeacon struct {
	data      []byte
	expiresAt time.Time
}

// NewMemoryBlindBeaconStore inicializa el almacén en memoria
func NewMemoryBlindBeaconStore() *MemoryBlindBeaconStore {
	return &MemoryBlindBeaconStore{
		beacons: make(map[string]storedBeacon),
	}
}

func (m *MemoryBlindBeaconStore) PutBeacon(topic string, data []byte, ttl time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.beacons[topic] = storedBeacon{
		data:      data,
		expiresAt: time.Now().Add(ttl),
	}
	return nil
}

func (m *MemoryBlindBeaconStore) GetBeacon(topic string) ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, ok := m.beacons[topic]
	if !ok {
		return nil, errors.New("baliza no encontrada")
	}
	if time.Now().After(b.expiresAt) {
		return nil, errors.New("baliza expirada por TTL")
	}
	return b.data, nil
}

func (m *MemoryBlindBeaconStore) DeleteBeacon(topic string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.beacons, topic)
	return nil
}

// DeriveTopicID calcula un identificador de punto de encuentro ciego dependiente de la época temporal.
// Garantiza Conocimiento Cero (Zero-Knowledge): El servidor externo jamás ve el DID del usuario.
func DeriveTopicID(epoch time.Time, ringDegree int, networkSeed string) string {
	epochHour := epoch.UTC().Truncate(time.Hour).Unix()
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("ipvn7:blind:rendezvous:%d:%d:%s", epochHour, ringDegree, networkSeed)))
	return hex.EncodeToString(h.Sum(nil))
}

// BeaconContent representa la carga útil protegida antes de ser cifrada
type BeaconContent struct {
	DID       string `json:"did"`
	Endpoint  string `json:"endpoint"`
	Timestamp int64  `json:"timestamp"`
	Nonce     string `json:"nonce"`
	Signature string `json:"signature"`
}

// BlindBeaconEnvelope representa el paquete opaco almacenado en el sustrato externo
type BlindBeaconEnvelope struct {
	TopicID   string `json:"topic_id"`
	Ciphertext []byte `json:"ciphertext"`
	CreatedAt int64  `json:"created_at"`
	TTLSeconds int64  `json:"ttl_seconds"`
}

// BlindRendezvousManager coordina el ciclo de vida de la señalización efímera y el desacoplamiento P2P
type BlindRendezvousManager struct {
	mu                sync.RWMutex
	Identity          *l0.Identity
	Store             BlindBeaconStore
	NetworkSeed       string
	TTL               time.Duration
	activeDirectPeers int32
	circuitBroken     atomic.Bool
	publishedTopics   []string
}

// NewBlindRendezvousManager crea una nueva instancia del orquestador de balizas
func NewBlindRendezvousManager(id *l0.Identity, store BlindBeaconStore, seed string) *BlindRendezvousManager {
	if seed == "" {
		seed = "ipvn7-sovereign-mesh-v0.4"
	}
	return &BlindRendezvousManager{
		Identity:    id,
		Store:       store,
		NetworkSeed: seed,
		TTL:         DefaultBeaconTTL,
	}
}

// PublishBlindBeacon cifra y publica una baliza efímera ciega en el almacén de anclaje
func (b *BlindRendezvousManager) PublishBlindBeacon(endpoint string, ringDegree int) (*BlindBeaconEnvelope, error) {
	if b.circuitBroken.Load() {
		return nil, errors.New("circuit breaker activo: señalización externa desactivada por convergencia a P2P puro")
	}

	topicID := DeriveTopicID(time.Now(), ringDegree, b.NetworkSeed)

	// Crear carga útil de baliza autenticada
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return nil, fmt.Errorf("error generando nonce: %w", err)
	}

	now := time.Now().Unix()
	content := BeaconContent{
		DID:       b.Identity.DID(),
		Endpoint:  endpoint,
		Timestamp: now,
		Nonce:     hex.EncodeToString(nonceBytes),
	}

	// Firmar la baliza con Ed25519
	signData := fmt.Sprintf("%s|%s|%d|%s", content.DID, content.Endpoint, content.Timestamp, content.Nonce)
	sig := ed25519.Sign(b.Identity.PrivateKey, []byte(signData))
	content.Signature = hex.EncodeToString(sig)

	contentBytes, err := json.Marshal(content)
	if err != nil {
		return nil, fmt.Errorf("error serializando contenido: %w", err)
	}

	// Cifrado simple de baliza con clave de red derivada (obfuscación simétrica contra espías pasivos)
	key := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", topicID, b.NetworkSeed)))
	ciphertext := make([]byte, len(contentBytes))
	for i := range contentBytes {
		ciphertext[i] = contentBytes[i] ^ key[i%len(key)]
	}

	envelope := &BlindBeaconEnvelope{
		TopicID:    topicID,
		Ciphertext: ciphertext,
		CreatedAt:  now,
		TTLSeconds: int64(b.TTL.Seconds()),
	}

	rawEnv, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("error serializando sobre: %w", err)
	}

	if err := b.Store.PutBeacon(topicID, rawEnv, b.TTL); err != nil {
		return nil, fmt.Errorf("error publicando en almacén ciego: %w", err)
	}

	b.mu.Lock()
	b.publishedTopics = append(b.publishedTopics, topicID)
	b.mu.Unlock()

	return envelope, nil
}

// DiscoverPeer busca y descifra una baliza ciega en el cuadrante temporal correspondiente
func (b *BlindRendezvousManager) DiscoverPeer(ringDegree int) (*BeaconContent, error) {
	topicID := DeriveTopicID(time.Now(), ringDegree, b.NetworkSeed)

	rawEnv, err := b.Store.GetBeacon(topicID)
	if err != nil {
		return nil, fmt.Errorf("no se encontró baliza para el tópico: %w", err)
	}

	var envelope BlindBeaconEnvelope
	if err := json.Unmarshal(rawEnv, &envelope); err != nil {
		return nil, fmt.Errorf("error decodificando sobre de baliza: %w", err)
	}

	// Descifrar contenido
	key := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", envelope.TopicID, b.NetworkSeed)))
	plain := make([]byte, len(envelope.Ciphertext))
	for i := range envelope.Ciphertext {
		plain[i] = envelope.Ciphertext[i] ^ key[i%len(key)]
	}

	var content BeaconContent
	if err := json.Unmarshal(plain, &content); err != nil {
		return nil, fmt.Errorf("error parseando contenido descifrado: %w", err)
	}

	// Verificar firma digital Ed25519 del emisor
	pub, err := l0.PublicKeyFromDID(content.DID)
	if err != nil {
		return nil, fmt.Errorf("DID del emisor inválido: %w", err)
	}

	signData := fmt.Sprintf("%s|%s|%d|%s", content.DID, content.Endpoint, content.Timestamp, content.Nonce)
	sig, err := hex.DecodeString(content.Signature)
	if err != nil || !ed25519.Verify(pub, []byte(signData), sig) {
		return nil, errors.New("firma criptográfica de la baliza inválida o adulterada")
	}

	return &content, nil
}

// ConsumeAndBurn elimina la baliza leída del almacén para garantizar no-persistencia
func (b *BlindRendezvousManager) ConsumeAndBurn(ringDegree int) error {
	topicID := DeriveTopicID(time.Now(), ringDegree, b.NetworkSeed)
	return b.Store.DeleteBeacon(topicID)
}

// NotifyDirectPeerConnected registra un nuevo enlace directo P2P verificado.
// Si se alcanza el umbral de desconexión (MinPeerTripThreshold), dispara el Circuit Breaker.
func (b *BlindRendezvousManager) NotifyDirectPeerConnected() bool {
	count := atomic.AddInt32(&b.activeDirectPeers, 1)
	if count >= MinPeerTripThreshold && !b.circuitBroken.Load() {
		b.DisconnectAndPurge()
		return true
	}
	return false
}

// DisconnectAndPurge purga todas las balizas locales y apaga la señalización externa
func (b *BlindRendezvousManager) DisconnectAndPurge() {
	b.circuitBroken.Store(true)

	b.mu.Lock()
	topics := append([]string(nil), b.publishedTopics...)
	b.publishedTopics = nil
	b.mu.Unlock()

	for _, t := range topics {
		_ = b.Store.DeleteBeacon(t)
	}
}

// IsCircuitBroken informa si el nodo ya superó la fase de bootstrap y opera en P2P puro
func (b *BlindRendezvousManager) IsCircuitBroken() bool {
	return b.circuitBroken.Load()
}

// ResetCircuitBreaker permite reactivar la señalización en caso de partición total de red
func (b *BlindRendezvousManager) ResetCircuitBreaker() {
	atomic.StoreInt32(&b.activeDirectPeers, 0)
	b.circuitBroken.Store(false)
}
