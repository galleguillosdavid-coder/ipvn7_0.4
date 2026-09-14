package l1

import (
	"testing"

	"ipvn7/pkg/l0"
)

func TestNetworkProbeEngineLifecycle(t *testing.T) {
	id, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("error creando identidad: %v", err)
	}

	router := NewKleinbergRouter(id)
	probe := NewNetworkProbeEngine(id, router)

	if probe.IsActive() {
		t.Errorf("la sonda debería arrancar inactiva (opt-in)")
	}

	// 1. Activar la sonda
	probe.SetActive(true)
	if !probe.IsActive() {
		t.Errorf("la sonda debería estar activa")
	}

	// 2. Obtener reporte
	report := probe.GetReport()
	if !report.IsActive {
		t.Errorf("el reporte debe reflejar estado activo")
	}
	if report.HealthScore <= 0 {
		t.Errorf("HealthScore debe ser mayor a 0, obtenido %f", report.HealthScore)
	}

	// 3. Desactivar
	probe.SetActive(false)
	if probe.IsActive() {
		t.Errorf("la sonda debería estar inactiva tras SetActive(false)")
	}
}
