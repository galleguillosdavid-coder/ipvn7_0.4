package l2

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestResilienceGuardian_PanicRecovery(t *testing.T) {
	cfg := DefaultGuardianConfig()
	g := NewResilienceGuardian(cfg)

	ctx := context.Background()
	err := g.SafeExecute(ctx, "risky_operation", func(ctx context.Context) error {
		panic("falla catastrófica de memoria o puntero nulo")
	})

	if err == nil {
		t.Fatal("SafeExecute must return an error when panic occurs")
	}

	if !strings.Contains(err.Error(), "pánico interceptado") {
		t.Errorf("Unexpected error message: %v", err)
	}

	status := g.GetHealthStatus()
	if status.TotalPanicsSaved != 1 {
		t.Errorf("Expected 1 panic saved, got %d", status.TotalPanicsSaved)
	}
}

func TestResilienceGuardian_CircuitBreaker(t *testing.T) {
	cfg := GuardianConfig{
		MaxRetries:       3,
		BaseBackoff:      5 * time.Millisecond,
		MaxBackoff:       20 * time.Millisecond,
		FailureThreshold: 2,
		CoolOffDuration:  50 * time.Millisecond,
		MaxHeapBytes:     512 * 1024 * 1024,
	}
	g := NewResilienceGuardian(cfg)
	ctx := context.Background()

	// Provocar 2 fallos consecutivos
	for i := 0; i < 2; i++ {
		_ = g.SafeExecute(ctx, "fail_op", func(ctx context.Context) error {
			return errors.New("error temporal")
		})
	}

	// Tercera llamada debe ser bloqueada inmediatamente por el circuit breaker
	err := g.SafeExecute(ctx, "fail_op", func(ctx context.Context) error {
		return nil
	})

	if err == nil || !strings.Contains(err.Error(), "ABIERTO") {
		t.Fatalf("Circuit breaker should be OPEN, got err: %v", err)
	}

	status := g.GetHealthStatus()
	if status.CircuitState != CircuitOpen || status.IsHealthy {
		t.Errorf("Expected CircuitOpen and unhealthy, got %+v", status)
	}

	// Esperar cool-off para half-open
	time.Sleep(60 * time.Millisecond)

	// Siguiente llamada debe probar y al tener éxito cerrar el circuito
	err = g.SafeExecute(ctx, "recover_op", func(ctx context.Context) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Recovery execution should succeed, got: %v", err)
	}

	status = g.GetHealthStatus()
	if status.CircuitState != CircuitClosed || !status.IsHealthy {
		t.Errorf("Circuit should be CLOSED and healthy, got %+v", status)
	}
}

func TestResilienceGuardian_RetryWithBackoff(t *testing.T) {
	cfg := GuardianConfig{
		MaxRetries:       4,
		BaseBackoff:      5 * time.Millisecond,
		MaxBackoff:       25 * time.Millisecond,
		FailureThreshold: 5,
		CoolOffDuration:  100 * time.Millisecond,
		MaxHeapBytes:     512 * 1024 * 1024,
	}
	g := NewResilienceGuardian(cfg)
	ctx := context.Background()

	attempts := 0
	err := g.SafeExecuteWithRetry(ctx, "eventual_success", func(ctx context.Context) error {
		attempts++
		if attempts < 3 {
			return errors.New("fallo temporal transitorio")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("SafeExecuteWithRetry failed: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	status := g.GetHealthStatus()
	if status.TotalRecoveries != 1 {
		t.Errorf("Expected 1 recovery counted, got %d", status.TotalRecoveries)
	}
}
