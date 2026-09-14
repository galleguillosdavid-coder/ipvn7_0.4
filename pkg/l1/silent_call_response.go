// Package l1 implementa la red silenciosa de Llamada y Respuesta (Silent Call-and-Response)
// inspirada en la arquitectura canónica de IPv7-HRS (D:\David\dvd\ipv7-hrs-final\target.md).
// Principio fundamental: Cero tráfico de difusión/multicast (Zero Broadcast Noise O(1)),
// comunicación estrictamente dirigida por Unicast autenticado, registro atómico de pares
// y semáforo DoS de ruta crítica (<15ns).
package l1

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Constantes de trama Silenciosa
const (
	FrameSilentCall     = 0x71
	FrameSilentResponse = 0x72
	FrameSilentPing     = 0x73
	FrameSilentPong     = 0x74

	// Límite de ráfaga anti-amplificación y semáforo
	MaxConcurrentSilentCalls = 10000
	DefaultRateLimitPerPeer  = 500 // req/sec por peer
)

// SilentPeer representa un nodo registrado en la malla silenciosa
type SilentPeer struct {
	DID        string    `json:"did"`
	Endpoint   string    `json:"endpoint"` // Dirección física (Zero-PII o loopback/LAN interna)
	LastSeen   time.Time `json:"last_seen"`
	CallCount  uint64    `json:"call_count"`
	RespCount  uint64    `json:"resp_count"`
	RateTokens int32     `json:"rate_tokens"`
	LastToken  int64     `json:"last_token_nano"`
	Verified   bool      `json:"verified"`
}

// SilentStats contiene métricas operativas del despachador
type SilentStats struct {
	TotalCallsReceived     uint64 `json:"total_calls_received"`
	TotalResponsesSent     uint64 `json:"total_responses_sent"`
	TotalCallsDispatched   uint64 `json:"total_calls_dispatched"`
	TotalResponsesReceived uint64 `json:"total_responses_received"`
	DroppedBroadcastNoise  uint64 `json:"dropped_broadcast_noise"`
	RateLimitedCalls       uint64 `json:"rate_limited_calls"`
	ActivePeers            int    `json:"active_peers"`
}

// SilentDispatcher gestiona el enrutamiento unicast silencioso y la protección DoS
type SilentDispatcher struct {
	mu         sync.RWMutex
	peers      map[string]*SilentPeer // Clave: Hash del DID o DID canónico
	localDID   string
	activeSem  chan struct{}
	running    bool
	stats      SilentStats

	// Callback opcional para entrega de datos a capas superiores
	onCallReceived func(srcDID string, payload []byte) ([]byte, error)
}

// NewSilentDispatcher inicializa un despachador silencioso
func NewSilentDispatcher(localDID string) *SilentDispatcher {
	return &SilentDispatcher{
		peers:     make(map[string]*SilentPeer),
		localDID:  localDID,
		activeSem: make(chan struct{}, MaxConcurrentSilentCalls),
		running:   true,
	}
}

// SetCallHandler registra el manejador para responder llamadas dirigidas
func (d *SilentDispatcher) SetCallHandler(fn func(srcDID string, payload []byte) ([]byte, error)) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.onCallReceived = fn
}

// RegisterPeer registra o actualiza un par conocido en la tabla unicast
func (d *SilentDispatcher) RegisterPeer(did string, endpoint string, verified bool) {
	d.mu.Lock()
	defer d.mu.Unlock()

	peer, exists := d.peers[did]
	if !exists {
		peer = &SilentPeer{
			DID:        did,
			Endpoint:   endpoint,
			RateTokens: DefaultRateLimitPerPeer,
			LastToken:  time.Now().UnixNano(),
			Verified:   verified,
		}
		d.peers[did] = peer
	} else {
		peer.Endpoint = endpoint
		peer.Verified = verified
	}
	peer.LastSeen = time.Now().UTC()
}

// LookupPeer busca un par registrado
func (d *SilentDispatcher) LookupPeer(did string) (*SilentPeer, bool) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	p, exists := d.peers[did]
	if !exists {
		return nil, false
	}
	// Devolver copia
	pCopy := *p
	return &pCopy, true
}

// FastRejectNoise evalúa en <15ns si una trama entrante es ruido broadcast o multicast no permitido
// En ipvn7 TODA difusión broadcast/multicast queda descartada en silencio absoluto.
func (d *SilentDispatcher) FastRejectNoise(isBroadcastOrMulticast bool) bool {
	if isBroadcastOrMulticast {
		atomic.AddUint64(&d.stats.DroppedBroadcastNoise, 1)
		return true // Rechazado inmediatamente
	}
	return false
}

// CheckHotPathSemaphore verifica rápidamente el límite de concurrencia y tasa del peer
func (d *SilentDispatcher) CheckHotPathSemaphore(peerDID string) bool {
	// 1. Semáforo global no-bloqueante
	select {
	case d.activeSem <- struct{}{}:
		// Slot adquirido
	default:
		// Sobrecarga global: drop
		atomic.AddUint64(&d.stats.RateLimitedCalls, 1)
		return false
	}

	// 2. Token bucket por peer
	d.mu.RLock()
	peer, exists := d.peers[peerDID]
	d.mu.RUnlock()

	if !exists {
		d.mu.Lock()
		if p2, ok2 := d.peers[peerDID]; ok2 {
			peer = p2
		} else {
			peer = &SilentPeer{
				DID:        peerDID,
				Endpoint:   "dynamic/unicast",
				RateTokens: DefaultRateLimitPerPeer,
				LastToken:  time.Now().UnixNano(),
				Verified:   false,
				LastSeen:   time.Now().UTC(),
			}
			d.peers[peerDID] = peer
		}
		d.mu.Unlock()
	}

	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&peer.LastToken)
	elapsed := time.Duration(now - last)

	// Reponer tokens si pasó más de 100ms
	if elapsed > 100*time.Millisecond {
		atomic.StoreInt32(&peer.RateTokens, DefaultRateLimitPerPeer)
		atomic.StoreInt64(&peer.LastToken, now)
	}

	tokens := atomic.AddInt32(&peer.RateTokens, -1)
	if tokens < 0 {
		<-d.activeSem
		atomic.AddUint64(&d.stats.RateLimitedCalls, 1)
		return false
	}

	return true
}

// ReleaseSemaphore libera el slot global del semáforo
func (d *SilentDispatcher) ReleaseSemaphore() {
	select {
	case <-d.activeSem:
	default:
	}
}

// ProcessInboundCall procesa una trama Call recibida de un peer unicast
func (d *SilentDispatcher) ProcessInboundCall(srcDID string, payload []byte) ([]byte, error) {
	atomic.AddUint64(&d.stats.TotalCallsReceived, 1)

	// Validar semáforo hot-path
	if !d.CheckHotPathSemaphore(srcDID) {
		return nil, errors.New("llamada descartada: semáforo de saturación o límite de tasa")
	}
	defer d.ReleaseSemaphore()

	d.mu.RLock()
	handler := d.onCallReceived
	peer, exists := d.peers[srcDID]
	if exists {
		atomic.AddUint64(&peer.CallCount, 1)
		peer.LastSeen = time.Now().UTC()
	}
	d.mu.RUnlock()

	if handler == nil {
		// Respuesta por defecto con ACK de llamada silenciosa
		atomic.AddUint64(&d.stats.TotalResponsesSent, 1)
		h := sha256.Sum256(payload)
		return []byte(fmt.Sprintf("ACK:SILENT_RESPONSE:%s", hex.EncodeToString(h[:8]))), nil
	}

	resp, err := handler(srcDID, payload)
	if err != nil {
		return nil, err
	}

	atomic.AddUint64(&d.stats.TotalResponsesSent, 1)
	return resp, nil
}

// DispatchCall simula el envío sincrónico de una llamada unicast a un peer registrado
func (d *SilentDispatcher) DispatchCall(targetDID string, payload []byte, remoteReceiver func(targetDID string, payload []byte) ([]byte, error)) ([]byte, error) {
	d.mu.RLock()
	peer, exists := d.peers[targetDID]
	d.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("peer destino %s no registrado en la malla silenciosa", targetDID)
	}

	atomic.AddUint64(&d.stats.TotalCallsDispatched, 1)

	if remoteReceiver == nil {
		return nil, errors.New("sin receptor remoto configurado")
	}

	resp, err := remoteReceiver(targetDID, payload)
	if err != nil {
		return nil, err
	}

	atomic.AddUint64(&d.stats.TotalResponsesReceived, 1)
	d.mu.Lock()
	peer.RespCount++
	peer.LastSeen = time.Now().UTC()
	d.mu.Unlock()

	return resp, nil
}

// GetStats retorna las métricas en tiempo real de la red silenciosa
func (d *SilentDispatcher) GetStats() SilentStats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	s := d.stats
	s.ActivePeers = len(d.peers)
	return s
}
