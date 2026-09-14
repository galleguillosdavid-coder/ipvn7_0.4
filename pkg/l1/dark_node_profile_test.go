package l1

import (
	"testing"
)

func TestDarkNodeMacroOrchestrator(t *testing.T) {
	fw := NewZTNAFirewall(true)
	pacer := NewPacketPacer(DefaultPacerConfig())
	orchestrator := NewDarkNodeOrchestrator(fw, nil, pacer)

	status := orchestrator.GetStatus()
	if status["mode"] != string(ModeStandardMesh) {
		t.Fatalf("Modo inicial esperado %s, obtenido %v", ModeStandardMesh, status["mode"])
	}
	if status["is_dark_node"] != false {
		t.Fatalf("is_dark_node inicial debe ser false")
	}

	// 1. Conmutar a Dark Node con un solo clic (Macro-Acción)
	newMode, err := orchestrator.ToggleMode()
	if err != nil {
		t.Fatalf("Error conmutando a Dark Node: %v", err)
	}
	if newMode != ModeDarkNodeEgressOnly {
		t.Fatalf("Esperado %s, obtenido %s", ModeDarkNodeEgressOnly, newMode)
	}

	darkStatus := orchestrator.GetStatus()
	if darkStatus["is_dark_node"] != true {
		t.Fatalf("is_dark_node debe ser true tras toggle")
	}
	if darkStatus["listeners_muted"] != true {
		t.Fatalf("listeners deben estar silenciados")
	}
	if darkStatus["sphinx_circuits_active"] != true {
		t.Fatalf("circuitos Sphinx deben estar activos")
	}
	if darkStatus["onion_hops"] != 3 {
		t.Fatalf("onion_hops debe ser 3")
	}

	// 2. Conmutar de vuelta a Standard Mesh
	restoredMode, err := orchestrator.ToggleMode()
	if err != nil {
		t.Fatalf("Error restaurando modo estándar: %v", err)
	}
	if restoredMode != ModeStandardMesh {
		t.Fatalf("Esperado %s, obtenido %s", ModeStandardMesh, restoredMode)
	}

	restoredStatus := orchestrator.GetStatus()
	if restoredStatus["is_dark_node"] != false {
		t.Fatalf("is_dark_node restaurado debe ser false")
	}
	if restoredStatus["listeners_muted"] != false {
		t.Fatalf("listeners restaurados deben estar activos")
	}
}
