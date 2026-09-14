package l1

import (
	"testing"
	"time"
)

func TestAntiReplayFilter_FourAttackScenarios(t *testing.T) {
	arb := NewGlobalMemoryArbiter(10 * 1024 * 1024)
	filter := NewAntiReplayFilter(arb)

	peer := "did:ipvn7:peer_test_alpha"
	session1 := uint64(1001)
	now := time.Now().Unix()

	// 1. Paquetes en orden
	if !filter.Accept(peer, session1, 1, now) {
		t.Fatal("Packet 1 must be accepted")
	}
	if !filter.Accept(peer, session1, 2, now) {
		t.Fatal("Packet 2 must be accepted")
	}
	if !filter.Accept(peer, session1, 5, now) {
		t.Fatal("Packet 5 (leap) must be accepted")
	}

	// 2. Escenario 1: Replay Inmediato (duplicado de seq 2)
	if filter.Accept(peer, session1, 2, now) {
		t.Fatal("Immediate duplicate of packet 2 must be rejected")
	}

	// 3. Escenario 4: Reorden legítimo con Jitter (paquete 3 retrasado, nunca visto)
	if !filter.Accept(peer, session1, 3, now) {
		t.Fatal("Packet 3 (reordered within window) must be accepted")
	}
	// Duplicado de paquete 3 ahora debe fallar
	if filter.Accept(peer, session1, 3, now) {
		t.Fatal("Duplicate of packet 3 must be rejected")
	}

	// 4. Escenario 2: Replay Tardío (fuera de la ventana)
	// Avanzar la ventana más allá de 1024
	if !filter.Accept(peer, session1, 2000, now) {
		t.Fatal("Packet 2000 must be accepted")
	}
	// Secuencia 4 ahora está fuera de la ventana (2000 - 4 = 1996 >= 1024)
	if filter.Accept(peer, session1, 4, now) {
		t.Fatal("Stale packet 4 outside window must be rejected")
	}

	// 5. Escenario 3: Cross-Session (mismo seq inyectado con SessionID obsoleta)
	oldSession := uint64(999)
	oldTimestamp := now - 10
	if filter.Accept(peer, oldSession, 2001, oldTimestamp) {
		t.Fatal("Cross-session packet with older timestamp/session must be rejected")
	}

	stats := filter.GetStats()
	if stats.ImmediateReplays == 0 {
		t.Error("Expected immediate replays to be recorded")
	}
	if stats.StaleReplays == 0 {
		t.Error("Expected stale replays to be recorded")
	}
	if stats.CrossSessionDrops == 0 {
		t.Error("Expected cross-session drops to be recorded")
	}
	t.Logf("Anti-Replay Stats: %+v", stats)
}
