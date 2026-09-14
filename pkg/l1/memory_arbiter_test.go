package l1

import (
	"testing"
)

func TestGlobalMemoryArbiter_QuotasAndRejection(t *testing.T) {
	// 1000 KB total
	// BudgetReplay: 200 KB
	// BudgetQoS: 300 KB
	total := uint64(1000 * 1024)
	arb := NewGlobalMemoryArbiter(total)

	// 1. Reserva dentro de cuota Replay (150 KB <= 200 KB)
	if !arb.Reserve(BudgetReplay, 150*1024) {
		t.Fatal("Reservation within quota should succeed")
	}

	// 2. Reserva que excede la cuota de Replay (150 + 60 = 210 KB > 200 KB)
	if arb.Reserve(BudgetReplay, 60*1024) {
		t.Fatal("Reservation exceeding class quota should be rejected")
	}

	stats := arb.GetStats()
	if stats.TotalRejections != 1 {
		t.Errorf("Expected 1 rejection, got %d", stats.TotalRejections)
	}

	// 3. Liberar 100 KB de Replay
	arb.Release(BudgetReplay, 100*1024)

	// 4. Ahora 60 KB debe tener éxito (50 + 60 = 110 KB <= 200 KB)
	if !arb.Reserve(BudgetReplay, 60*1024) {
		t.Fatal("Reservation after release should succeed")
	}

	// 5. Reserva en QoS (300 KB máx)
	if !arb.Reserve(BudgetQoS, 250*1024) {
		t.Fatal("QoS reservation should succeed")
	}

	finalStats := arb.GetStats()
	if finalStats.TotalAllocated != (110+250)*1024 {
		t.Errorf("Expected 360 KB allocated, got %d", finalStats.TotalAllocated/1024)
	}
}
