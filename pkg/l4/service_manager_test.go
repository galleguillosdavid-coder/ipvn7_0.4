package l4_test

import (
	"testing"
	"time"

	"ipvn7/pkg/l1"
	"ipvn7/pkg/l4"
)

func TestServiceLifecycleManager(t *testing.T) {
	sm := l4.NewServiceLifecycleManager(20 * time.Millisecond)

	// 1. Declarar servicios dormidos
	sm.RegisterDormantService("chat_e2ee")
	sm.RegisterDormantService("remote_desktop")

	active := sm.GetActiveServices()
	if len(active) != 0 {
		t.Errorf("Esperado 0 servicios activos inicialmente (Zero Footprint), obtenidos: %d", len(active))
	}

	// 2. Invocación bajo demanda con alcance
	svc, err := sm.RequestService("chat_e2ee", l1.ScopeLocal)
	if err != nil {
		t.Fatalf("RequestService falló: %v", err)
	}
	if svc.State != l4.ServiceStateActive || svc.Scope != l1.ScopeLocal {
		t.Errorf("Estado o alcance incorrecto para chat_e2ee: state=%d, scope=%d", svc.State, svc.Scope)
	}

	activeAfter := sm.GetActiveServices()
	if len(activeAfter) != 1 {
		t.Errorf("Esperado 1 servicio activo tras invocación, obtenidos: %d", len(activeAfter))
	}

	// 3. Liberación de sesión y Poda Activa por inactividad
	sm.ReleaseService("chat_e2ee")
	time.Sleep(30 * time.Millisecond)

	pruned := sm.PruneIdleServices()
	if len(pruned) != 1 || pruned[0] != "chat_e2ee" {
		t.Errorf("Poda activa no purgó chat_e2ee: %v", pruned)
	}

	activeFinal := sm.GetActiveServices()
	if len(activeFinal) != 0 {
		t.Errorf("Esperado 0 servicios tras poda activa, obtenidos: %d", len(activeFinal))
	}
}
