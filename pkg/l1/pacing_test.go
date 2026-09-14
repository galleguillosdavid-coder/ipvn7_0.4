package l1_test

import (
	"context"
	"testing"
	"time"

	"ipvn7/pkg/l1"
)

func TestPacketPacer(t *testing.T) {
	cfg := l1.PacerConfig{
		BottleneckRateBytesPerSec: 1024 * 128, // 128 KB/s
		SafetyMarginPercent:       10,
		MaxBurstPackets:           1,
		MinInterval:               1 * time.Millisecond,
	}

	pacer := l1.NewPacketPacer(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Enviar 5 paquetes a través del marcapasos
	start := time.Now()
	for i := 0; i < 5; i++ {
		if err := pacer.Pace(ctx, 1280); err != nil {
			t.Fatalf("Pace falló en iteración %d: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	stats := pacer.Stats()
	if stats.PacketsSent != 5 {
		t.Errorf("PacketsSent esperados: 5, obtenidos: %d", stats.PacketsSent)
	}
	if elapsed < 40*time.Millisecond {
		t.Errorf("Pacing demasiado rápido (%v), debió regular el tiempo entre paquetes", elapsed)
	}

	// Probar backpressure / actualización de tasa
	pacer.UpdateBottleneckRate(1024 * 64)
	newStats := pacer.Stats()
	if newStats.EffectiveRateBytesPerSec >= stats.EffectiveRateBytesPerSec {
		t.Errorf("Tasa efectiva no disminuyó tras backpressure: %d vs %d",
			newStats.EffectiveRateBytesPerSec, stats.EffectiveRateBytesPerSec)
	}
}

func TestPeerHealthFSMAndMakeBeforeBreak(t *testing.T) {
	// 1. Probar cálculo de puntajes
	scoreHealthy, stateHealthy := l1.CalculateHealthScore(1.0, 5.0, 0.0)
	if stateHealthy != l1.HealthStateHealthy || scoreHealthy < 80.0 {
		t.Errorf("Esperado estado Healthy, obtenido %d (score: %f)", stateHealthy, scoreHealthy)
	}

	scoreDegraded, stateDegraded := l1.CalculateHealthScore(0.8, 35.0, 0.06)
	if stateDegraded != l1.HealthStateDegraded {
		t.Errorf("Esperado estado Degraded, obtenido %d (score: %f)", stateDegraded, scoreDegraded)
	}

	scoreUnstable, stateUnstable := l1.CalculateHealthScore(0.5, 70.0, 0.22)
	if stateUnstable != l1.HealthStateUnstable {
		t.Errorf("Esperado estado Unstable, obtenido %d (score: %f)", stateUnstable, scoreUnstable)
	}
}

func TestSemanticDiscoveryPull(t *testing.T) {
	registry := l1.NewSemanticRegistry("did:ipvn7:local-alice")

	// Registrar silenciosamente capacidades
	registry.RegisterCapability("print/raw", "did:ipvn7:printer-bob", "192.168.1.55:9100", 50.0, 10*time.Minute)
	registry.RegisterCapability("storage/dag", "did:ipvn7:storage-node-charlie", "192.168.1.80:7777", 200.0, 10*time.Minute)

	// Consultar capacidad por intención (pull)
	query := l1.IntentQuery{
		TargetTag:    "print/raw",
		RequiredMbps: 10.0,
		Scope:        l1.ScopeLocal,
	}

	provider, err := registry.QueryIntent(query)
	if err != nil {
		t.Fatalf("QueryIntent falló: %v", err)
	}

	if provider.ProviderDID != "did:ipvn7:printer-bob" {
		t.Errorf("Proveedor incorrecto retornado: %s", provider.ProviderDID)
	}

	// Consultar capacidad inexistente
	queryNone := l1.IntentQuery{
		TargetTag: "nonexistent/service",
	}
	_, errNone := registry.QueryIntent(queryNone)
	if errNone == nil {
		t.Errorf("Consulta de capacidad inexistente debió fallar")
	}
}
