package l1

import (
	"testing"
	"time"
)

func TestZTNAFirewall(t *testing.T) {
	fw := NewZTNAFirewall(true) // Default-Deny activado

	didA := "did:ipvn7:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	didB := "did:ipvn7:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	// 1. DID no registrado debe ser rechazado por Default-Deny
	dec, _ := fw.EvaluateInbound(didA, 8080)
	if dec != DecisionDropDefaultDeny {
		t.Fatalf("Esperado DecisionDropDefaultDeny, obtenido %v", dec)
	}

	// 2. Autorizar didA únicamente para puerto virtual 7001
	fw.AuthorizeDID(&DIDPolicy{
		DID:          didA,
		AllowInbound: true,
		AllowedPorts: []uint16{7001},
	})

	// Puerto 7001 debe ser aceptado
	dec, _ = fw.EvaluateInbound(didA, 7001)
	if dec != DecisionAccept {
		t.Fatalf("Esperado DecisionAccept en puerto 7001, obtenido %v", dec)
	}

	// Puerto 8080 para didA debe ser rechazado por ACL
	dec, _ = fw.EvaluateInbound(didA, 8080)
	if dec != DecisionDropACL {
		t.Fatalf("Esperado DecisionDropACL en puerto 8080, obtenido %v", dec)
	}

	// 3. Probar política expirada
	fw.AuthorizeDID(&DIDPolicy{
		DID:          didB,
		AllowInbound: true,
		ExpiresAt:    time.Now().Add(-1 * time.Minute), // ya expiró
	})
	dec, _ = fw.EvaluateInbound(didB, 7001)
	if dec != DecisionDropACL {
		t.Fatalf("Esperado DecisionDropACL por expiración, obtenido %v", dec)
	}

	// 4. Revocar didA
	fw.RevokeDID(didA)
	dec, _ = fw.EvaluateInbound(didA, 7001)
	if dec != DecisionDropDefaultDeny {
		t.Fatalf("Esperado DecisionDropDefaultDeny tras revocación, obtenido %v", dec)
	}

	stats := fw.Stats()
	if stats.PacketsAccepted != 1 || stats.PacketsDroppedDD != 2 || stats.PacketsDroppedACL != 2 {
		t.Fatalf("Estadísticas inesperadas: %+v", stats)
	}
}
