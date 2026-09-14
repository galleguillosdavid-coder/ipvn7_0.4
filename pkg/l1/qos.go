package l1

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// TrafficClass define la categoría de prioridad de un datagrama
type TrafficClass uint8

const (
	ClassControl     TrafficClass = 0 // Prioridad máxima: enrutamiento, FSM, roaming
	ClassInteractive TrafficClass = 1 // Prioridad media: terminales, streaming, RPC
	ClassBulk        TrafficClass = 2 // Prioridad baja: transferencias de archivos, DAG
)

func (tc TrafficClass) String() string {
	switch tc {
	case ClassControl:
		return "CONTROL"
	case ClassInteractive:
		return "INTERACTIVE"
	case ClassBulk:
		return "BULK"
	default:
		return "UNKNOWN"
	}
}

// TokenBucket implementa un limitador de tasa adaptativo por identidad
type TokenBucket struct {
	mu         sync.Mutex
	capacity   float64
	tokens     float64
	refillRate float64 // Tokens (bytes) por segundo
	lastRefill time.Time
}

// NewTokenBucket crea un cubo de fichas con capacidad y tasa de rellenado
func NewTokenBucket(capacity, refillRate float64) *TokenBucket {
	return &TokenBucket{
		capacity:   capacity,
		tokens:     capacity,
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// Consume intenta debitar n tokens. Retorna true si hay suficientes disponibles
func (tb *TokenBucket) Consume(n float64) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.lastRefill = now

	// Rellenar tokens acumulados según el tiempo transcurrido
	tb.tokens += elapsed * tb.refillRate
	if tb.tokens > tb.capacity {
		tb.tokens = tb.capacity
	}

	if tb.tokens >= n {
		tb.tokens -= n
		return true
	}

	return false
}

// AvailableTokens retorna la cantidad actual de tokens disponibles
func (tb *TokenBucket) AvailableTokens() float64 {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.tokens
}

// PoWChallenge representa un desafío computacional asimétrico emitido por el receptor
type PoWChallenge struct {
	PeerDID    string    `json:"peer_did"`
	Challenge  string    `json:"challenge"` // Hex encoded hash
	Difficulty uint8     `json:"difficulty"` // Número de bits cero requeridos (ej. 16)
	IssuedAt   time.Time `json:"issued_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// QoSManager gestiona la priorización de tráfico y desafíos dinámicos Anti-DDoS (Dimensión 3)
type QoSManager struct {
	mu           sync.RWMutex
	buckets      map[string]*TokenBucket
	activePoWs   map[string]*PoWChallenge // challenge_hex -> Challenge
	defaultRate  float64                  // Bytes por segundo (alineado con Pacer: ~1.18 MB/s)
	burstLimit   float64                  // Ráfaga permitida en bytes (~64 KB)
	powThreshold float64                  // Por debajo de este nivel de tokens, se exige PoW

	// Métricas
	TotalThrottled uint64
	PoWsIssued     uint64
	PoWsSolved     uint64
}

// NewQoSManager inicializa el orquestador de Calidad de Servicio y mitigación de abusos
func NewQoSManager() *QoSManager {
	return &QoSManager{
		buckets:      make(map[string]*TokenBucket),
		activePoWs:   make(map[string]*PoWChallenge),
		defaultRate:  1179648.0, // 9.44 Mbps sostenible según axioma de Flujo Sostenible
		burstLimit:   65536.0,   // Ráfaga de 64 KB
		powThreshold: 1000.0,    // Umbral de agotamiento severo
	}
}

// getOrCreateBucket obtiene o inicializa el cubo de fichas para un DID
func (qm *QoSManager) getOrCreateBucket(did string) *TokenBucket {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if b, ok := qm.buckets[did]; ok {
		return b
	}

	b := NewTokenBucket(qm.burstLimit, qm.defaultRate)
	qm.buckets[did] = b
	return b
}

// EvaluatePacket evalúa si un paquete de clase tc y longitud bytes puede transmitirse o procesarse
func (qm *QoSManager) EvaluatePacket(did string, tc TrafficClass, length int) (bool, *PoWChallenge) {
	// Tráfico de control nunca se estrangula
	if tc == ClassControl {
		return true, nil
	}

	tb := qm.getOrCreateBucket(did)
	allowed := tb.Consume(float64(length))

	if allowed {
		return true, nil
	}

	// Si no está permitido y los tokens están en estado crítico, emitir desafío PoW
	qm.mu.Lock()
	qm.TotalThrottled++
	qm.PoWsIssued++

	// Generar sal criptográfica aleatoria
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)

	h := sha256.Sum256([]byte(fmt.Sprintf("%s:%s:%d", did, hex.EncodeToString(salt), time.Now().UnixNano())))
	chHex := hex.EncodeToString(h[:])

	challenge := &PoWChallenge{
		PeerDID:    did,
		Challenge:  chHex,
		Difficulty: 16, // 16 bits en 0 (2 bytes cero = 65,536 hashes promedio, ~5 ms en CPU)
		IssuedAt:   time.Now(),
		ExpiresAt:  time.Now().Add(30 * time.Second),
	}
	qm.activePoWs[chHex] = challenge
	qm.mu.Unlock()

	return false, challenge
}

// VerifyAndCreditPoW valida la solución aportada por el par y recarga créditos si es correcta
func (qm *QoSManager) VerifyAndCreditPoW(challengeHex string, nonce uint64) bool {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	ch, ok := qm.activePoWs[challengeHex]
	if !ok || time.Now().After(ch.ExpiresAt) {
		delete(qm.activePoWs, challengeHex)
		return false
	}

	// Comprobar prueba de trabajo: SHA256(challengeBytes || nonceBytes)
	chBytes, err := hex.DecodeString(ch.Challenge)
	if err != nil {
		return false
	}

	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)

	data := append(chBytes, nonceBytes...)
	hash := sha256.Sum256(data)

	// Verificar si tiene al menos Difficulty bits en cero (para 16 bits: hash[0] == 0 && hash[1] == 0)
	bytesZero := int(ch.Difficulty / 8)
	for i := 0; i < bytesZero; i++ {
		if hash[i] != 0 {
			return false
		}
	}

	// Válido: bonificar al DID con ráfaga adicional en su TokenBucket
	delete(qm.activePoWs, challengeHex)
	qm.PoWsSolved++

	if b, found := qm.buckets[ch.PeerDID]; found {
		b.mu.Lock()
		b.tokens = qm.burstLimit // Restaura ráfaga completa como recompensa al peaje computacional
		b.mu.Unlock()
	}

	return true
}

// SolvePoW es una función de conveniencia que un cliente o nodo utiliza para resolver el desafío
func SolvePoW(challengeHex string, difficulty uint8) (uint64, error) {
	chBytes, err := hex.DecodeString(challengeHex)
	if err != nil {
		return 0, err
	}

	bytesZero := int(difficulty / 8)
	nonceBytes := make([]byte, 8)

	for nonce := uint64(0); ; nonce++ {
		binary.BigEndian.PutUint64(nonceBytes, nonce)
		data := append(chBytes, nonceBytes...)
		hash := sha256.Sum256(data)

		match := true
		for i := 0; i < bytesZero; i++ {
			if hash[i] != 0 {
				match = false
				break
			}
		}

		if match {
			return nonce, nil
		}
	}
}

// QoSStats expone métricas de QoS para observabilidad
type QoSStats struct {
	TotalThrottled uint64 `json:"total_throttled"`
	PoWsIssued     uint64 `json:"pows_issued"`
	PoWsSolved     uint64 `json:"pows_solved"`
	TrackedPeers   int    `json:"tracked_peers"`
}

// Stats retorna la instantánea de métricas de QoS
func (qm *QoSManager) Stats() QoSStats {
	qm.mu.RLock()
	defer qm.mu.RUnlock()

	return QoSStats{
		TotalThrottled: qm.TotalThrottled,
		PoWsIssued:     qm.PoWsIssued,
		PoWsSolved:     qm.PoWsSolved,
		TrackedPeers:   len(qm.buckets),
	}
}
