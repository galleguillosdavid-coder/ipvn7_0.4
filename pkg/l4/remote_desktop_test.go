package l4

import (
	"testing"

	"ipvn7/pkg/l0"
	"ipvn7/pkg/l1"
)

func TestRemoteDesktopManager_SASAndInput(t *testing.T) {
	id, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("Error generando identidad: %v", err)
	}

	fw := l1.NewZTNAFirewall(true)
	rdm := NewRemoteDesktopManager(id, fw)

	// 1. Verificar estado de sesión y derivación SAS
	status := rdm.GetStatus()
	if status == nil {
		t.Fatalf("Estado de sesión remoto no puede ser nulo")
	}

	if len(status.SASDigits) != 6 {
		t.Errorf("Código SAS numérico debe tener exactamente 6 dígitos, tiene %d", len(status.SASDigits))
	}

	if len(status.SASEmojis) != 4 {
		t.Errorf("SAS debe incluir 4 emojis, tiene %d", len(status.SASEmojis))
	}

	// 2. Procesar evento de entrada interactivo
	evt := &RemoteInputEvent{
		Type:      "mouse_click",
		X:         480,
		Y:         270,
		Button:    0,
		Timestamp: 1726000000,
	}

	res, err := rdm.HandleInputEvent(evt)
	if err != nil {
		t.Fatalf("Error procesando input remoto: %v", err)
	}

	if res["status"] != "PROCESSED" {
		t.Errorf("Estado esperado PROCESSED, obtenido %v", res["status"])
	}
}
