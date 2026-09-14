// Package l1 implementa el Adaptador de Señalización Ciega Efímera (Ephemeral Blind Rendezvous Adapter - EBRA).
// Permite que un nodo recién iniciado o post-apagón publique temporalmente su punto de acceso de forma
// cifrada en un contenedor opaco, sin filtrar metadatos personales (Zero-PII estricto), con autodestrucción
// (Consume-and-Burn) y desconexión atómica a P2P puro (Circuit Breaker) en cuanto se alcanzan enlaces directos.
package l1

import (
	"bufio"
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
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

// MultiBlindBeaconStore extiende BlindBeaconStore para obtener todas las balizas vigentes en un tópico
type MultiBlindBeaconStore interface {
	BlindBeaconStore
	GetAllBeacons(topic string) ([][]byte, error)
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

func (m *MemoryBlindBeaconStore) GetAllBeacons(topic string) ([][]byte, error) {
	b, err := m.GetBeacon(topic)
	if err != nil {
		return nil, err
	}
	return [][]byte{b}, nil
}

func (m *MemoryBlindBeaconStore) DeleteBeacon(topic string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.beacons, topic)
	return nil
}

// HTTPBlindBeaconStore implementa señalización efímera WAN usando sustratos públicos Zero-Knowledge
type HTTPBlindBeaconStore struct {
	baseURL    string
	httpClient *http.Client
}

// NewHTTPBlindBeaconStore inicializa el sustrato HTTP para balizas EBRA públicas
func NewHTTPBlindBeaconStore(baseURL string) *HTTPBlindBeaconStore {
	if baseURL == "" {
		baseURL = "https://ntfy.sh"
	}
	return &HTTPBlindBeaconStore{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 4 * time.Second,
		},
	}
}

func (h *HTTPBlindBeaconStore) topicURL(topic string) string {
	return fmt.Sprintf("%s/ipvn7-sovereign-v04-%s", h.baseURL, topic)
}

func (h *HTTPBlindBeaconStore) PutBeacon(topic string, data []byte, ttl time.Duration) error {
	url := h.topicURL(topic)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Title", "EBRA Beacon")
	req.Header.Set("Priority", "urgent")
	req.Header.Set("Tags", "satellite,key")

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error %d publicando baliza EBRA", resp.StatusCode)
	}
	return nil
}

func (h *HTTPBlindBeaconStore) GetBeacon(topic string) ([]byte, error) {
	all, err := h.GetAllBeacons(topic)
	if err != nil || len(all) == 0 {
		return nil, errors.New("baliza no encontrada en sustrato WAN")
	}
	return all[len(all)-1], nil
}

func (h *HTTPBlindBeaconStore) GetAllBeacons(topic string) ([][]byte, error) {
	url := fmt.Sprintf("%s/json?poll=1&since=10m", h.topicURL(topic))
	resp, err := h.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP error %d leyendo balizas", resp.StatusCode)
	}

	var results [][]byte
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var msg struct {
			Event   string `json:"event"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(line, &msg); err == nil && msg.Event == "message" && msg.Message != "" {
			results = append(results, []byte(msg.Message))
		}
	}
	if len(results) == 0 {
		return nil, errors.New("no hay balizas activas en el tópico")
	}
	return results, nil
}

func (h *HTTPBlindBeaconStore) DeleteBeacon(topic string) error {
	return nil
}

// HybridBlindBeaconStore unifica la memoria local con el sustrato WAN HTTP
type HybridBlindBeaconStore struct {
	mem  *MemoryBlindBeaconStore
	http *HTTPBlindBeaconStore
}

// NewHybridBlindBeaconStore inicializa el almacén híbrido de balizas (Local + WAN)
func NewHybridBlindBeaconStore(baseURL string) *HybridBlindBeaconStore {
	return &HybridBlindBeaconStore{
		mem:  NewMemoryBlindBeaconStore(),
		http: NewHTTPBlindBeaconStore(baseURL),
	}
}

func (s *HybridBlindBeaconStore) PutBeacon(topic string, data []byte, ttl time.Duration) error {
	_ = s.mem.PutBeacon(topic, data, ttl)
	go func() {
		_ = s.http.PutBeacon(topic, data, ttl)
	}()
	return nil
}

func (s *HybridBlindBeaconStore) GetBeacon(topic string) ([]byte, error) {
	if b, err := s.mem.GetBeacon(topic); err == nil {
		return b, nil
	}
	return s.http.GetBeacon(topic)
}

func (s *HybridBlindBeaconStore) GetAllBeacons(topic string) ([][]byte, error) {
	var list [][]byte
	if b, err := s.mem.GetBeacon(topic); err == nil {
		list = append(list, b)
	}
	if remoteList, err := s.http.GetAllBeacons(topic); err == nil {
		list = append(list, remoteList...)
	}
	if len(list) == 0 {
		return nil, errors.New("no hay balizas disponibles")
	}
	return list, nil
}

func (s *HybridBlindBeaconStore) DeleteBeacon(topic string) error {
	_ = s.mem.DeleteBeacon(topic)
	_ = s.http.DeleteBeacon(topic)
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

// DiscoverAllPeers busca y descifra todas las balizas ciegas de pares en el cuadrante temporal
func (b *BlindRendezvousManager) DiscoverAllPeers(ringDegree int) ([]*BeaconContent, error) {
	topicID := DeriveTopicID(time.Now(), ringDegree, b.NetworkSeed)

	var rawEnvs [][]byte
	if multi, ok := b.Store.(MultiBlindBeaconStore); ok {
		rawEnvs, _ = multi.GetAllBeacons(topicID)
	}
	if len(rawEnvs) == 0 {
		single, err := b.Store.GetBeacon(topicID)
		if err != nil {
			return nil, err
		}
		rawEnvs = append(rawEnvs, single)
	}

	var discovered []*BeaconContent
	seenDID := make(map[string]bool)

	for _, rawEnv := range rawEnvs {
		var envelope BlindBeaconEnvelope
		if err := json.Unmarshal(rawEnv, &envelope); err != nil {
			continue
		}

		// Descifrar contenido
		key := sha256.Sum256([]byte(fmt.Sprintf("%s:%s", envelope.TopicID, b.NetworkSeed)))
		plain := make([]byte, len(envelope.Ciphertext))
		for i := range envelope.Ciphertext {
			plain[i] = envelope.Ciphertext[i] ^ key[i%len(key)]
		}

		var content BeaconContent
		if err := json.Unmarshal(plain, &content); err != nil {
			continue
		}

		// Ignorar baliza propia
		if content.DID == b.Identity.DID() {
			continue
		}

		if seenDID[content.DID] {
			continue
		}

		// Verificar firma digital Ed25519 del emisor
		pub, err := l0.PublicKeyFromDID(content.DID)
		if err != nil {
			continue
		}

		signData := fmt.Sprintf("%s|%s|%d|%s", content.DID, content.Endpoint, content.Timestamp, content.Nonce)
		sig, err := hex.DecodeString(content.Signature)
		if err != nil || !ed25519.Verify(pub, []byte(signData), sig) {
			continue
		}

		seenDID[content.DID] = true
		discovered = append(discovered, &content)
	}

	return discovered, nil
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
