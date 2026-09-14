package l2

import (
	"testing"
)

func TestTransitAccountingAndTitForTat(t *testing.T) {
	ta := NewTransitAccounting()
	ta.courtesyBytes = 1000   // Límite bajo para verificar transiciones en test
	ta.courtesyPackets = 10

	didCollab := "did:ipvn7:colaborador1111111111111111111111111111111111111111111111111111"
	didParasite := "did:ipvn7:parasito222222222222222222222222222222222222222222222222222222"

	// 1. Par nuevo dentro del margen de cortesía -> TierNormal
	ta.RecordTx(didParasite, 500)
	if tier := ta.GetPeerTier(didParasite); tier != TierNormal {
		t.Fatalf("Esperado TierNormal por cortesía, obtenido %v", tier)
	}

	// 2. Par parásito excede la cortesía y no aporta tráfico de vuelta -> TierThrottled
	ta.RecordTx(didParasite, 5000)
	if tier := ta.GetPeerTier(didParasite); tier != TierThrottled {
		t.Fatalf("Esperado TierThrottled para free-rider, obtenido %v", tier)
	}

	// 3. Par colaborador aporta más de lo que consume -> TierPriority
	ta.RecordTx(didCollab, 2000)
	ta.RecordRx(didCollab, 3000) // ratio = 1.5
	if tier := ta.GetPeerTier(didCollab); tier != TierPriority {
		t.Fatalf("Esperado TierPriority para nodo colaborador, obtenido %v", tier)
	}

	stats := ta.Stats()
	if stats.PriorityPeers != 1 || stats.ThrottledPeers != 1 {
		t.Fatalf("Estadísticas de Tiers incorrectas: %+v", stats)
	}
}
