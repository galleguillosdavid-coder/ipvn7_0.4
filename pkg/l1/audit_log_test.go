package l1_test

import (
	"testing"

	"ipvn7/pkg/l1"
)

func TestMerkleAuditLogInalterabilityAndVerification(t *testing.T) {
	log := l1.NewMerkleAuditLog()

	// 1. Verificar génesis
	if log.TotalEntries() != 1 {
		t.Fatalf("Se esperaba 1 entrada génesis, obtenido: %d", log.TotalEntries())
	}

	valid, err := log.VerifyIntegrity()
	if !valid || err != nil {
		t.Fatalf("Fallo en verificación génesis: %v", err)
	}

	// 2. Registrar eventos secuenciales
	e1, err := log.AppendEvent("ZTNA_DROP", "did:ipvn7:attacker", map[string]interface{}{"port": 7001, "action": "DROP_DEFAULT_DENY"})
	if err != nil {
		t.Fatalf("AppendEvent 1 falló: %v", err)
	}

	e2, err := log.AppendEvent("FSM_TRANSITION", "did:ipvn7:peer_a", map[string]interface{}{"from": "HEALTHY", "to": "DEGRADED", "rtt_ms": 48.2})
	if err != nil {
		t.Fatalf("AppendEvent 2 falló: %v", err)
	}

	e3, err := log.AppendEvent("COPILOT_HEAL", "did:ipvn7:self", map[string]interface{}{"action": "FAILOVER_MULTIPATH", "confidence": 0.95})
	if err != nil {
		t.Fatalf("AppendEvent 3 falló: %v", err)
	}

	// 3. Verificar encadenamiento
	if e2.PrevHash != e1.EntryHash || e3.PrevHash != e2.EntryHash {
		t.Errorf("Encadenamiento criptográfico roto entre bloques contiguos")
	}

	valid, err = log.VerifyIntegrity()
	if !valid || err != nil {
		t.Fatalf("La bitácora válida falló la verificación de integridad: %v", err)
	}

	// 4. Intentar adulteración manual (simular alteración de datos)
	e2.EventType = "ADULTERATED_EVENT"
	valid, err = log.VerifyIntegrity()
	if valid || err == nil {
		t.Fatalf("La bitácora debió detectar la alteración de datos, pero pasó la verificación!")
	}
}
