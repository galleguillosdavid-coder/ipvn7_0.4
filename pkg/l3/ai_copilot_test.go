package l3_test

import (
	"context"
	"testing"

	"ipvn7/pkg/l0"
	"ipvn7/pkg/l1"
	"ipvn7/pkg/l2"
	"ipvn7/pkg/l3"
)

func TestAICopilotAnomalyDetectionAndAutoHealing(t *testing.T) {
	_, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("Error generando identidad: %v", err)
	}

	fw := l1.NewZTNAFirewall(true)
	mp := l1.NewMultipathScheduler()
	mp.RegisterInterface(&l1.PhysicalInterface{
		Name:      "primary-wlan",
		Type:      "WIFI",
		LatencyMs: 15.0,
		LossRate:  0.001,
		Weight:    1000,
		Active:    true,
	})
	mp.RegisterInterface(&l1.PhysicalInterface{
		Name:      "secondary-eth",
		Type:      "ETHERNET",
		LatencyMs: 2.0,
		LossRate:  0.0001,
		Weight:    1000,
		Active:    true,
	})

	telemetry := l2.NewTelemetryRingBuffer()

	copilot := l3.NewAICopilotEngine(fw, mp, telemetry)

	// 1. Probar detección de anomalías
	// Caso A: Latencia normal, 0 descartes
	anomaliesA := copilot.DetectAnomalies(12.5, 0)
	if len(anomaliesA) != 0 {
		t.Errorf("Se esperaba 0 anomalías para latencia normal, pero se detectaron: %d", len(anomaliesA))
	}

	// Caso B: Latencia anómala (> 40ms)
	anomaliesB := copilot.DetectAnomalies(85.0, 0)
	if len(anomaliesB) != 1 || anomaliesB[0].Type != "LATENCY_SPIKE" {
		t.Fatalf("Se esperaba 1 anomalía LATENCY_SPIKE, se obtuvo: %+v", anomaliesB)
	}

	// Caso C: Inundación no autorizada (> 5 descartes)
	anomaliesC := copilot.DetectAnomalies(10.0, 15)
	if len(anomaliesC) != 1 || anomaliesC[0].Type != "UNAUTHORIZED_FLOOD" {
		t.Fatalf("Se esperaba 1 anomalía UNAUTHORIZED_FLOOD, se obtuvo: %+v", anomaliesC)
	}

	// 2. Diagnóstico y Auto-reparación para LATENCY_SPIKE
	ctx := context.Background()
	diagB, err := copilot.DiagnoseAndHeal(ctx, anomaliesB[0])
	if err != nil {
		t.Fatalf("DiagnoseAndHeal falló: %v", err)
	}
	if diagB.ActionType != "FAILOVER_MULTIPATH" || !diagB.ExecutedAction {
		t.Errorf("Diagnóstico LATENCY_SPIKE inválido: %+v", diagB)
	}

	// 3. Diagnóstico y Auto-reparación para UNAUTHORIZED_FLOOD
	attackerDID := "did:ipvn7:malicious_intruder_vector"
	anomaliesC[0].PeerDID = attackerDID
	diagC, err := copilot.DiagnoseAndHeal(ctx, anomaliesC[0])
	if err != nil {
		t.Fatalf("DiagnoseAndHeal falló: %v", err)
	}
	if diagC.ActionType != "ENFORCE_ZTNA" || !diagC.ExecutedAction {
		t.Errorf("Diagnóstico UNAUTHORIZED_FLOOD inválido: %+v", diagC)
	}

	// Verificar estado de auto-reparación
	decision, _ := fw.EvaluateInbound(attackerDID, 7001)
	if decision != l1.DecisionDropDefaultDeny {
		t.Errorf("El firewall debería bloquear con DecisionDropDefaultDeny tras auto-curación, resultado: %v", decision)
	}

	// 4. Verificar historial y estadísticas
	stats := copilot.Stats()
	if stats.TotalDiagnoses != 2 || stats.ActionsApplied != 2 {
		t.Errorf("Estadísticas de copiloto inconsistentes: %+v", stats)
	}

	history := copilot.GetHistory(10)
	if len(history) != 2 {
		t.Errorf("Historial esperado 2, obtenido %d", len(history))
	}
}
