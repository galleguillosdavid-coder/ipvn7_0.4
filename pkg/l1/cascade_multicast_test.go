package l1

import (
	"testing"

	"ipvn7/pkg/l0"
)

func TestCascadeMulticast_TreeAndAmplification(t *testing.T) {
	id, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("Error generando identidad: %v", err)
	}

	router := NewKleinbergRouter(id)
	engine := NewCascadeMulticastEngine(id, router)

	report := engine.Broadcast("canal-soberano-test", []byte("DATAGRAMA_PRUEBA_1280B"))
	if report == nil {
		t.Fatalf("El reporte de cascada no puede ser nulo")
	}

	if report.WirePacketSize != 1280 {
		t.Errorf("WirePacketSize esperado 1280, obtenido %d", report.WirePacketSize)
	}

	if report.HostPacketsSent > 2 {
		t.Errorf("El host emisor no debe emitir más de 2 paquetes directos, emitió %d", report.HostPacketsSent)
	}

	if report.BandwidthSavedPct < 90.0 {
		t.Errorf("El ahorro de ancho de banda debe ser > 90%%, obtenido %.2f%%", report.BandwidthSavedPct)
	}

	if report.RootNode == nil || len(report.RootNode.Children) < 2 {
		t.Errorf("La raíz del árbol debe ramificarse en al menos 2 hijos")
	}
}
