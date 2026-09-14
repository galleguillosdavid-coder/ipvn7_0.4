package l1

import (
	"testing"
)

func TestXDPFastPathEngineOperations(t *testing.T) {
	engine := NewXDPFastPathEngine("eth0")
	if engine == nil {
		t.Fatalf("NewXDPFastPathEngine returned nil")
	}

	// 1. Sincronizar rutas desde Kùzu
	didA := "did:ipvn7:node-alpha-1234567890"
	didB := "did:ipvn7:node-beta-0987654321"

	engine.SyncRouteFromKuzu(didA, "10.7.1.10", 2, 14.5, 1)
	engine.SyncRouteFromKuzu(didB, "10.7.1.11", 7, 68.2, 2)

	status := engine.GetStatus()
	if status["active_routes"].(int) != 2 {
		t.Fatalf("Esperadas 2 rutas activas, obtenidas %v", status["active_routes"])
	}

	// 2. Procesar paquete no conforme (descarte rápido XDP_DROP)
	badPkt := make([]byte, 500)
	action := engine.ProcessPacketFastPath(badPkt)
	if action != XDPActionDrop {
		t.Fatalf("Paquete no conforme debió ser descartado (XDP_DROP), obtenido: %d", action)
	}

	// 3. Procesar paquete conforme (1280 bytes determinista)
	validPkt := make([]byte, 1280)
	action2 := engine.ProcessPacketFastPath(validPkt)
	if action2 != XDPActionRedirect {
		t.Fatalf("Paquete válido con rutas debió ser redirigido (XDP_REDIRECT), obtenido: %d", action2)
	}

	// 4. Validar estadísticas atómicas
	finalStatus := engine.GetStatus()
	if finalStatus["rx_packets"].(uint64) != 2 {
		t.Fatalf("Rx packets incorrecto: %v", finalStatus["rx_packets"])
	}
	if finalStatus["xdp_dropped"].(uint64) != 1 {
		t.Fatalf("Descartes incorrectos: %v", finalStatus["xdp_dropped"])
	}
	if finalStatus["xdp_redirected"].(uint64) != 1 {
		t.Fatalf("Redirecciones incorrectas: %v", finalStatus["xdp_redirected"])
	}

	dump := engine.FormatBPFMapDump()
	if len(dump) == 0 {
		t.Fatalf("FormatBPFMapDump vacío")
	}
}
