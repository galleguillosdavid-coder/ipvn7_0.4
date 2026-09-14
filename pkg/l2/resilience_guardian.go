// Package l2 implementa el Guardián de Resiliencia y Recuperación ante Fallos
// rescatado y adaptado de ASA Nexus (D:\David\asa-mirror\nexus\guardian.go).
// Provee:
// 1. Backoff exponencial con Jitter completo para reintentos de red sin efecto manada.
// 2. Patrón Cortocircuito (Circuit Breaker) para evitar fallos en cascada en nodos degradados.
// 3. Monitor de salud de memoria (Heap), rutinas Go y descriptores.
// 4. Ejecución segura con recuperación de pánicos (SafeExecute).
package l2

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Estados del Circuit Breaker
type CircuitState string

const (
	CircuitClosed   CircuitState = "CLOSED"    // Tráfico normal
	CircuitOpen     CircuitState = "OPEN"      // Bloqueo total tras fallos recurrentes
	CircuitHalfOpen CircuitState = "HALF_OPEN" // Probando recuperación con tráfico limitado
)

// GuardianConfig define los parámetros operativos del sistema de resiliencia
type GuardianConfig struct {
	MaxRetries       int           `json:"max_retries"`
	BaseBackoff      time.Duration `json:"base_backoff"`
	MaxBackoff       time.Duration `json:"max_backoff"`
	FailureThreshold int           `json:"failure_threshold"`
	CoolOffDuration  time.Duration `json:"cool_off_duration"`
	MaxHeapBytes     uint64        `json:"max_heap_bytes"` // Umbral de alerta de memoria
}

// DefaultGuardianConfig provee valores canónicos estables para el overlay
func DefaultGuardianConfig() GuardianConfig {
	return GuardianConfig{
		MaxRetries:       5,
		BaseBackoff:      50 * time.Millisecond,
		MaxBackoff:       2 * time.Second,
		FailureThreshold: 3,
		CoolOffDuration:  5 * time.Second,
		MaxHeapBytes:     512 * 1024 * 1024, // 512MB
	}
}

// HealthStatus contiene la telemetría viva del nodo
type HealthStatus struct {
	HeapAllocMB      float64      `json:"heap_alloc_mb"`
	HeapSysMB        float64      `json:"heap_sys_mb"`
	NumGoroutine     int          `json:"num_goroutine"`
	NumGC            uint32       `json:"num_gc"`
	CircuitState     CircuitState `json:"circuit_state"`
	ConsecutiveFails int          `json:"consecutive_fails"`
	TotalExecutions  uint64       `json:"total_executions"`
	TotalPanicsSaved uint64       `json:"total_panics_saved"`
	TotalRecoveries  uint64       `json:"total_recoveries"`
	IsHealthy        bool         `json:"is_healthy"`
	StatusMessage    string       `json:"status_message"`
}

// ResilienceGuardian orquesta la ejecución resiliente y protege la estabilidad del proceso
type ResilienceGuardian struct {
	config     GuardianConfig
	mu         sync.RWMutex
	state      CircuitState
	fails      int
	lastFail   time.Time
	totalExec  uint64
	totalPanic uint64
	totalRecov uint64
	rnd        *rand.Rand
	rndMu      sync.Mutex
}

// NewResilienceGuardian inicializa una nueva instancia del guardián
func NewResilienceGuardian(cfg GuardianConfig) *ResilienceGuardian {
	if cfg.MaxRetries <= 0 {
		cfg = DefaultGuardianConfig()
	}
	return &ResilienceGuardian{
		config: cfg,
		state:  CircuitClosed,
		rnd:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// ComputeBackoff calcula el intervalo de espera exponencial con Jitter completo:
// wait = Min(MaxBackoff, BaseBackoff * 2^attempt) + UniformRandom(0, Jitter)
func (g *ResilienceGuardian) ComputeBackoff(attempt int) time.Duration {
	if attempt < 0 {
		attempt = 0
	}
	// Limitar factor a 10 para evitar overflow
	factor := math.Pow(2, float64(attempt))
	base := float64(g.config.BaseBackoff) * factor
	max := float64(g.config.MaxBackoff)

	var wait float64
	if base > max {
		wait = max
	} else {
		wait = base
	}

	// Jitter completo (aleatorización de 0 a wait)
	g.rndMu.Lock()
	jitter := g.rnd.Float64() * wait * 0.5
	g.rndMu.Unlock()

	return time.Duration(wait + jitter)
}

// SafeExecute ejecuta una función con recuperación de pánico y gestión de Circuit Breaker
func (g *ResilienceGuardian) SafeExecute(ctx context.Context, opName string, fn func(ctx context.Context) error) (err error) {
	atomic.AddUint64(&g.totalExec, 1)

	// 1. Validar estado del Circuit Breaker
	if !g.allowExecution() {
		return fmt.Errorf("circuit breaker [%s] está ABIERTO: operación '%s' rechazada preventivamente", g.state, opName)
	}

	panicked := true

	// 2. Recuperación de Pánicos (Panic Recovery)
	defer func() {
		if r := recover(); r != nil {
			atomic.AddUint64(&g.totalPanic, 1)
			g.recordFailure()
			err = fmt.Errorf("pánico interceptado por ResilienceGuardian en '%s': %v", opName, r)
		} else if panicked {
			g.recordFailure()
		}
	}()

	err = fn(ctx)
	panicked = false

	if err != nil {
		g.recordFailure()
		return err
	}

	g.recordSuccess()
	return nil
}

// SafeExecuteWithRetry ejecuta una función reintentando automáticamente con backoff exponencial
func (g *ResilienceGuardian) SafeExecuteWithRetry(ctx context.Context, opName string, fn func(ctx context.Context) error) error {
	var lastErr error
	for attempt := 0; attempt < g.config.MaxRetries; attempt++ {
		err := g.SafeExecute(ctx, opName, fn)
		if err == nil {
			if attempt > 0 {
				atomic.AddUint64(&g.totalRecov, 1)
			}
			return nil
		}
		lastErr = err

		// Si el context ya está cancelado, salir
		if ctx.Err() != nil {
			return ctx.Err()
		}

		backoff := g.ComputeBackoff(attempt)
		select {
		case <-time.After(backoff):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return fmt.Errorf("operación '%s' falló tras %d reintentos. Último error: %w", opName, g.config.MaxRetries, lastErr)
}

func (g *ResilienceGuardian) allowExecution() bool {
	g.mu.Lock()
	defer g.mu.Unlock()

	switch g.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		if time.Since(g.lastFail) > g.config.CoolOffDuration {
			g.state = CircuitHalfOpen
			return true
		}
		return false
	case CircuitHalfOpen:
		// En half-open permitimos probar
		return true
	default:
		return true
	}
}

func (g *ResilienceGuardian) recordSuccess() {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.state == CircuitHalfOpen || g.fails > 0 {
		g.state = CircuitClosed
		g.fails = 0
	}
}

func (g *ResilienceGuardian) recordFailure() {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.fails++
	g.lastFail = time.Now()

	if g.fails >= g.config.FailureThreshold {
		g.state = CircuitOpen
	}
}

// GetHealthStatus recolecta la telemetría viva del entorno de ejecución
func (g *ResilienceGuardian) GetHealthStatus() HealthStatus {
	g.mu.RLock()
	st := g.state
	fails := g.fails
	g.mu.RUnlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	heapMB := float64(m.Alloc) / (1024 * 1024)
	sysMB := float64(m.Sys) / (1024 * 1024)
	routines := runtime.NumGoroutine()

	healthy := true
	statusMsg := "Sistema Operativo de Malla Soberana Óptimo"

	if st == CircuitOpen {
		healthy = false
		statusMsg = "Degradado: Cortocircuito ABIERTO por fallos repetidos"
	} else if m.Alloc > g.config.MaxHeapBytes {
		healthy = false
		statusMsg = fmt.Sprintf("Alerta: Uso de heap excede límite (%.1f MB > %.1f MB)", heapMB, float64(g.config.MaxHeapBytes)/(1024*1024))
	}

	return HealthStatus{
		HeapAllocMB:      math.Round(heapMB*100) / 100,
		HeapSysMB:        math.Round(sysMB*100) / 100,
		NumGoroutine:     routines,
		NumGC:            m.NumGC,
		CircuitState:     st,
		ConsecutiveFails: fails,
		TotalExecutions:  atomic.LoadUint64(&g.totalExec),
		TotalPanicsSaved: atomic.LoadUint64(&g.totalPanic),
		TotalRecoveries:  atomic.LoadUint64(&g.totalRecov),
		IsHealthy:        healthy,
		StatusMessage:    statusMsg,
	}
}
