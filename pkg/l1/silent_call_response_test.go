package l1

import (
	"strings"
	"testing"
	"time"
)

func TestSilentDispatcher_RegistrationAndNoiseRejection(t *testing.T) {
	d := NewSilentDispatcher("did:ipv7:localnode777")

	// 1. Noise Rejection (<15ns)
	start := time.Now()
	for i := 0; i < 1000; i++ {
		if !d.FastRejectNoise(true) {
			t.Fatal("Broadcast traffic must be rejected immediately")
		}
	}
	elapsed := time.Since(start)
	avgNs := float64(elapsed.Nanoseconds()) / 1000.0
	t.Logf("Promedio FastRejectNoise: %.2f ns/op", avgNs)

	if d.FastRejectNoise(false) {
		t.Fatal("Unicast traffic must NOT be rejected")
	}

	// 2. Peer Registration
	d.RegisterPeer("did:ipv7:peerA1", "192.168.1.106:7777", true)
	peer, exists := d.LookupPeer("did:ipv7:peerA1")
	if !exists {
		t.Fatal("Peer did:ipv7:peerA1 should exist")
	}
	if !peer.Verified || peer.Endpoint != "192.168.1.106:7777" {
		t.Errorf("Unexpected peer data: %+v", peer)
	}

	stats := d.GetStats()
	if stats.ActivePeers != 1 {
		t.Errorf("Expected 1 active peer, got %d", stats.ActivePeers)
	}
	if stats.DroppedBroadcastNoise != 1000 {
		t.Errorf("Expected 1000 dropped broadcast packets, got %d", stats.DroppedBroadcastNoise)
	}
}

func TestSilentDispatcher_CallResponseFlow(t *testing.T) {
	d := NewSilentDispatcher("did:ipv7:serverNode")
	d.RegisterPeer("did:ipv7:clientNode", "10.0.0.2:7777", true)

	// Set custom handler
	d.SetCallHandler(func(srcDID string, payload []byte) ([]byte, error) {
		return []byte("REPLY:" + string(payload)), nil
	})

	// Process inbound call
	resp, err := d.ProcessInboundCall("did:ipv7:clientNode", []byte("PING_PAYLOAD"))
	if err != nil {
		t.Fatalf("ProcessInboundCall failed: %v", err)
	}

	if string(resp) != "REPLY:PING_PAYLOAD" {
		t.Errorf("Expected 'REPLY:PING_PAYLOAD', got '%s'", string(resp))
	}

	stats := d.GetStats()
	if stats.TotalCallsReceived != 1 || stats.TotalResponsesSent != 1 {
		t.Errorf("Unexpected stats: %+v", stats)
	}
}

func TestSilentDispatcher_HotPathSemaphore(t *testing.T) {
	d := NewSilentDispatcher("did:ipv7:host")
	d.RegisterPeer("did:ipv7:flooder", "10.0.0.99:7777", true)

	// Consume tokens rapidly
	successCount := 0
	dropCount := 0
	for i := 0; i < 600; i++ {
		_, err := d.ProcessInboundCall("did:ipv7:flooder", []byte("TEST"))
		if err == nil {
			successCount++
		} else {
			if strings.Contains(err.Error(), "límite de tasa") || strings.Contains(err.Error(), "semáforo") {
				dropCount++
			}
		}
	}

	t.Logf("Hot-path Semaphore: %d aceptadas, %d rate-limited", successCount, dropCount)
	if dropCount == 0 {
		t.Error("Expected rate limiting drops after 500 requests")
	}
}
