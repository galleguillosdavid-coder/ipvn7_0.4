// Package l1 implementa el Protocolo de Recuperación en Cascada Post-Apagón y la Cadencia
// Estocástica Desprogresiva con Jitter Descorrelacionado (Anti-Thundering Herd).
package l1

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"ipvn7/pkg/l0"
)

// Constantes de temporización estocástica
const (
	DefaultBaseInterval = 5 * time.Second
	DefaultMaxInterval  = 5 * time.Minute
	DefaultMultiplier   = 3.0
)

// RecoveryPhase representa cada fase de la cascada de recuperación
type RecoveryPhase int

const (
	PhaseColdMemory RecoveryPhase = iota + 1 // Fase 1: Memoria local / Grafo persistente
	PhaseOffGridProximity                    // Fase 2: Malla de proximidad física (BLE/LoRa/LAN)
	PhaseWANProbe                            // Fase 3: Verificación de salida WAN / STUN
	PhaseBlindRendezvous                     // Fase 4: Baliza ciega efímera de rescate
	PhaseConvergedMesh                       // Convergencia: Malla P2P operativa
)

func (p RecoveryPhase) String() string {
	switch p {
	case PhaseColdMemory:
		return "FASE_1_MEMORIA_LOCAL_KUZU"
	case PhaseOffGridProximity:
		return "FASE_2_PROXIMIDAD_OFFGRID"
	case PhaseWANProbe:
		return "FASE_3_SONDEO_WAN_STUN"
	case PhaseBlindRendezvous:
		return "FASE_4_RESCATE_BALIZA_CIEGA"
	case PhaseConvergedMesh:
		return "CONVERGENCIA_P2P_PURAMENTE_ESTABLE"
	default:
		return "DESCONOCIDA"
	}
}

// DecorrelatedJitterTimer calcula intervalos estocásticos desincronizados para evitar el colapso por estampida
type DecorrelatedJitterTimer struct {
	mu          sync.Mutex
	base        time.Duration
	max         time.Duration
	current     time.Duration
	multiplier  float64
	stepCount   uint64
	lastCalculated time.Duration
}

// NewDecorrelatedJitterTimer inicializa el temporizador estocástico
func NewDecorrelatedJitterTimer(base, max time.Duration) *DecorrelatedJitterTimer {
	if base <= 0 {
		base = DefaultBaseInterval
	}
	if max <= base {
		max = DefaultMaxInterval
	}
	return &DecorrelatedJitterTimer{
		base:       base,
		max:        max,
		current:    base,
		multiplier: DefaultMultiplier,
	}
}

// NextInterval calcula el siguiente intervalo usando Decorrelated Jitter:
// T_{i+1} = min(T_max, Uniform(T_base, T_current * 3))
func (t *DecorrelatedJitterTimer) NextInterval() time.Duration {
	t.mu.Lock()
	defer t.mu.Unlock()

	upper := float64(t.current) * t.multiplier
	if upper > float64(t.max) {
		upper = float64(t.max)
	}
	lower := float64(t.base)
	if lower > upper {
		lower = upper
	}

	// Obtener número aleatorio uniforme criptográficamente robusto
	var buf [8]byte
	_, _ = rand.Read(buf[:])
	rndFloat := float64(binary.LittleEndian.Uint64(buf[:])) / float64(^uint64(0))

	// Calcular intervalo aleatorio uniforme en [lower, upper]
	delta := upper - lower
	nextFloat := lower + rndFloat*delta
	next := time.Duration(nextFloat)

	if next < t.base {
		next = t.base
	}
	if next > t.max {
		next = t.max
	}

	t.current = next
	t.lastCalculated = next
	t.stepCount++
	return next
}

// Reset reinicia el cálculo al intervalo base
func (t *DecorrelatedJitterTimer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.current = t.base
	t.stepCount = 0
}

// Stats retorna el estado actual del temporizador
func (t *DecorrelatedJitterTimer) Stats() (last time.Duration, steps uint64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.lastCalculated, t.stepCount
}

// BlackoutRecoveryManager orquesta la recuperación autónoma en cascada
type BlackoutRecoveryManager struct {
	mu             sync.RWMutex
	Identity       *l0.Identity
	CurrentPhase   RecoveryPhase
	JitterTimer    *DecorrelatedJitterTimer
	Rendezvous     *BlindRendezvousManager
	LocalPeersSeen map[string]time.Time
	OffGridActive  bool
	IsOnlineWAN    bool
	RecoveryLogs   []string
	phaseChangeCh  chan RecoveryPhase
	stopCh         chan struct{}
	running        atomic.Bool
}

// NewBlackoutRecoveryManager crea un nuevo gestor de recuperación post-apagón
func NewBlackoutRecoveryManager(id *l0.Identity, rz *BlindRendezvousManager) *BlackoutRecoveryManager {
	return &BlackoutRecoveryManager{
		Identity:       id,
		CurrentPhase:   PhaseColdMemory,
		JitterTimer:    NewDecorrelatedJitterTimer(2*time.Second, 45*time.Second),
		Rendezvous:     rz,
		LocalPeersSeen: make(map[string]time.Time),
		phaseChangeCh:  make(chan RecoveryPhase, 16),
		stopCh:         make(chan struct{}),
	}
}

// StepCascade evalúa la cascada paso a paso según la evidencia empírica observada
func (b *BlackoutRecoveryManager) StepCascade(hasLocalHistory bool, offGridDiscovered int, wanReachable bool, directPeersCount int) RecoveryPhase {
	b.mu.Lock()
	defer b.mu.Unlock()

	prevPhase := b.CurrentPhase

	// Si ya tenemos al menos 2 pares directos conectados, la malla converge
	if directPeersCount >= MinPeerTripThreshold {
		b.CurrentPhase = PhaseConvergedMesh
		if b.Rendezvous != nil && !b.Rendezvous.IsCircuitBroken() {
			b.Rendezvous.NotifyDirectPeerConnected()
			b.Rendezvous.NotifyDirectPeerConnected()
		}
		b.logTransition(prevPhase, b.CurrentPhase, "Convergencia P2P alcanzada con éxito")
		return b.CurrentPhase
	}

	// Fase 1: Memoria local
	if hasLocalHistory && b.CurrentPhase == PhaseColdMemory {
		b.CurrentPhase = PhaseColdMemory
		b.logTransition(prevPhase, b.CurrentPhase, "Sondeando vecinos históricos en KùzuDB")
		return b.CurrentPhase
	}

	// Fase 2: Si no hay memoria local o falló el sondeo, buscar en proximidad física
	if offGridDiscovered > 0 {
		b.CurrentPhase = PhaseOffGridProximity
		b.logTransition(prevPhase, b.CurrentPhase, fmt.Sprintf("Vecinos detectados en interfaz de proximidad (BLE/LoRa/LAN): %d", offGridDiscovered))
		return b.CurrentPhase
	}

	// Fase 3: Si no hay vecinos locales pero hay conexión a Internet, validar WAN
	if wanReachable {
		b.IsOnlineWAN = true
		b.CurrentPhase = PhaseWANProbe

		// Si pasamos la prueba WAN y tenemos gestor de baliza ciega, avanzar a Fase 4
		if b.Rendezvous != nil && !b.Rendezvous.IsCircuitBroken() {
			b.CurrentPhase = PhaseBlindRendezvous
			b.logTransition(prevPhase, b.CurrentPhase, "Internet WAN activa: Activando baliza ciega efímera de rescate (EBRA)")
		} else {
			b.logTransition(prevPhase, b.CurrentPhase, "WAN alcanzable pero baliza ciega desactivada")
		}
		return b.CurrentPhase
	}

	// Fallback por defecto: permanecer en Fase 1 con reintentos estocásticos
	b.CurrentPhase = PhaseColdMemory
	b.logTransition(prevPhase, b.CurrentPhase, "Sin conectividad WAN ni proximidad: Modo espera estocástica silenciosa")
	return b.CurrentPhase
}

func (b *BlackoutRecoveryManager) logTransition(from, to RecoveryPhase, reason string) {
	if from != to {
		entry := fmt.Sprintf("[%s] Transición %s ➔ %s (%s)", time.Now().Format("15:04:05.000"), from, to, reason)
		b.RecoveryLogs = append(b.RecoveryLogs, entry)
		if len(b.RecoveryLogs) > 30 {
			b.RecoveryLogs = b.RecoveryLogs[1:]
		}
	}
}

// GetLogs devuelve el historial de transiciones
func (b *BlackoutRecoveryManager) GetLogs() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return append([]string(nil), b.RecoveryLogs...)
}

// CurrentPhaseString retorna el nombre textual de la fase actual
func (b *BlackoutRecoveryManager) CurrentPhaseString() string {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.CurrentPhase.String()
}
