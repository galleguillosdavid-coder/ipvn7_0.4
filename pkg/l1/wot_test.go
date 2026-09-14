package l1

import (
	"testing"
	"time"

	"ipvn7/pkg/l0"
)

func TestWebOfTrustAndAttenuatedReputation(t *testing.T) {
	wot := NewWebOfTrust()

	// Creamos una cadena de 4 nodos: A -> B -> C -> D
	idA, _ := l0.GenerateIdentity()
	idB, _ := l0.GenerateIdentity()
	idC, _ := l0.GenerateIdentity()
	idD, _ := l0.GenerateIdentity()

	// 1. A avala a B (100% confianza)
	_, err := wot.SignAndIssueVouch(idA, idB.DID(), 1.0, "nodo amigo de confianza", 24*time.Hour)
	if err != nil {
		t.Fatalf("Error emitiendo aval A->B: %v", err)
	}

	// 2. B avala a C (90% confianza)
	_, err = wot.SignAndIssueVouch(idB, idC.DID(), 0.9, "colega de investigacion", 24*time.Hour)
	if err != nil {
		t.Fatalf("Error emitiendo aval B->C: %v", err)
	}

	// 3. C avala a D (100% confianza)
	_, err = wot.SignAndIssueVouch(idC, idD.DID(), 1.0, "infraestructura mesh", 24*time.Hour)
	if err != nil {
		t.Fatalf("Error emitiendo aval C->D: %v", err)
	}

	// Verificación de reputación desde A:
	// A -> B: 1 salto. score = 1.0 * (0.85^1) * 100 = 85.0
	scoreB, hopsB := wot.CalculateReputation(idA.DID(), idB.DID())
	if hopsB != 1 || scoreB != 85.0 {
		t.Fatalf("Esperado hops=1, score=85.0 para B, obtenido hops=%d, score=%.2f", hopsB, scoreB)
	}

	// A -> C: 2 saltos. score = 1.0 * 0.9 * (0.85^2) * 100 = 0.9 * 0.7225 * 100 = 65.025 (~65.03)
	scoreC, hopsC := wot.CalculateReputation(idA.DID(), idC.DID())
	if hopsC != 2 || scoreC < 64.0 || scoreC > 66.0 {
		t.Fatalf("Esperado hops=2, score ~65.0 para C, obtenido hops=%d, score=%.2f", hopsC, scoreC)
	}

	// A -> D: 3 saltos (límite máximo)
	scoreD, hopsD := wot.CalculateReputation(idA.DID(), idD.DID())
	if hopsD != 3 || scoreD <= 0 {
		t.Fatalf("Esperado hops=3, score > 0 para D, obtenido hops=%d, score=%.2f", hopsD, scoreD)
	}

	// Nodo no conectado E debe tener reputación 0 y hops -1
	idE, _ := l0.GenerateIdentity()
	scoreE, hopsE := wot.CalculateReputation(idA.DID(), idE.DID())
	if hopsE != -1 || scoreE != 0 {
		t.Fatalf("Esperado nodo desconocido con score=0 y hops=-1, obtenido score=%.2f, hops=%d", scoreE, hopsE)
	}
}
