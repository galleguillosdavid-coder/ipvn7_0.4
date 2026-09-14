package l1

import (
	"testing"
	"time"

	"ipvn7/pkg/l0"
)

func TestDecorrelatedJitterMath(t *testing.T) {
	base := 100 * time.Millisecond
	max := 2 * time.Second

	timer := NewDecorrelatedJitterTimer(base, max)

	// Ejecutar 50 iteraciones para verificar que se mantiene dentro de [base, max] y no es estático
	var prev time.Duration
	distinctCount := 0

	for i := 0; i < 50; i++ {
		interval := timer.NextInterval()

		if interval < base {
			t.Fatalf("iteración %d: intervalo %v es menor que base %v", i, interval, base)
		}
		if interval > max {
			t.Fatalf("iteración %d: intervalo %v es mayor que max %v", i, interval, max)
		}
		if i > 0 && interval != prev {
			distinctCount++
		}
		prev = interval
	}

	// La variabilidad estocástica debe producir múltiples valores distintos
	if distinctCount < 20 {
		t.Errorf("variabilidad estocástica insuficiente: solo %d valores distintos en 50 iteraciones", distinctCount)
	}

	// Probar reseteo
	timer.Reset()
	last, steps := timer.Stats()
	if steps != 0 {
		t.Errorf("steps esperado 0 tras reset, obtenido %d", steps)
	}
	_ = last
}

func TestRecoveryCascadeProgression(t *testing.T) {
	id, _ := l0.GenerateIdentity()
	store := NewMemoryBlindBeaconStore()
	rz := NewBlindRendezvousManager(id, store, "test-cascade-seed")

	manager := NewBlackoutRecoveryManager(id, rz)

	// Estado inicial: ColdMemory
	if manager.CurrentPhase != PhaseColdMemory {
		t.Errorf("fase inicial esperada %v, obtenida %v", PhaseColdMemory, manager.CurrentPhase)
	}

	// 1. Paso con solo vecinos de proximidad física (Fase 2)
	phase := manager.StepCascade(false, 3, false, 0)
	if phase != PhaseOffGridProximity {
		t.Errorf("fase esperada %v, obtenida %v", PhaseOffGridProximity, phase)
	}

	// 2. Paso sin proximidad física pero con WAN reachable (Fase 4: Baliza ciega)
	phase = manager.StepCascade(false, 0, true, 0)
	if phase != PhaseBlindRendezvous {
		t.Errorf("fase esperada %v, obtenida %v", PhaseBlindRendezvous, phase)
	}

	// 3. Paso con 2 o más pares directos (Convergencia)
	phase = manager.StepCascade(false, 0, true, 2)
	if phase != PhaseConvergedMesh {
		t.Errorf("fase esperada %v, obtenida %v", PhaseConvergedMesh, phase)
	}

	// El circuit breaker del Rendezvous debe haberse disparado automáticamente
	if !rz.IsCircuitBroken() {
		t.Errorf("el circuit breaker de la baliza ciega debió dispararse tras la convergencia")
	}

	logs := manager.GetLogs()
	if len(logs) == 0 {
		t.Errorf("deberían existir registros de auditoría de transiciones")
	}
}
