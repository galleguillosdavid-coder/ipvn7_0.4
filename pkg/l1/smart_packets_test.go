package l1

import (
	"context"
	"testing"
	"time"
)

// HangFilter simula un filtro malicioso o lento que se cuelga
type HangFilter struct{}

func (f *HangFilter) ID() string {
	return "hang-filter-test"
}

func (f *HangFilter) Point() HookPoint {
	return HookForwarding
}

func (f *HangFilter) Execute(ctx context.Context, payload []byte) (*FilterResult, error) {
	select {
	case <-time.After(50 * time.Millisecond): // Más que el límite de 10 ms
		return &FilterResult{Pass: true}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func TestSmartPacketPipeline(t *testing.T) {
	pipeline := NewSmartPacketPipeline()

	// 1. Paquete regular en Egress debe pasar DLP
	cleanPayload := []byte("paquete_de_datos_valido_sin_fugas")
	_, pass, _ := pipeline.Process(HookEgress, cleanPayload)
	if !pass {
		t.Fatalf("Esperado que el paquete limpio pase el filtro DLP")
	}

	// 2. Paquete con token restringido PII debe ser descartado por DLP
	leakPayload := []byte("ALERTA: PASSWORD=supersecret_no_transmitir")
	_, pass, reason := pipeline.Process(HookEgress, leakPayload)
	if pass {
		t.Fatalf("Esperado que el filtro DLP descarte el paquete con PASSWORD=")
	}
	t.Logf("Descarte exitoso por DLP: %s", reason)

	// 3. Registrar filtro que excede 10 ms: debe abortarse por timeout
	pipeline.RegisterFilter(&HangFilter{})
	_, pass, reason = pipeline.Process(HookForwarding, cleanPayload)
	if pass {
		t.Fatalf("Esperado que el filtro que cuelga sea abortado por timeout")
	}
	t.Logf("Aborto por timeout exitoso: %s", reason)

	stats := pipeline.Stats()
	if stats.PacketsDropped != 2 || stats.TimeoutsCount != 1 {
		t.Fatalf("Estadísticas inesperadas: %+v", stats)
	}
}
