package l1

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"ipvn7/pkg/l0"
)

func TestBlindBeaconPublishAndDiscover(t *testing.T) {
	id, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("error generando identidad: %v", err)
	}

	store := NewMemoryBlindBeaconStore()
	manager := NewBlindRendezvousManager(id, store, "test-mesh-cluster-alpha")

	endpoint := "192.168.1.105:7001"
	envelope, err := manager.PublishBlindBeacon(endpoint, 0)
	if err != nil {
		t.Fatalf("error publicando baliza ciega: %v", err)
	}

	// 1. Verificar Zero-PII: El DID o la IP no deben aparecer en texto plano en el sobre cifrado
	if bytes.Contains(envelope.Ciphertext, []byte(id.DID())) {
		t.Fatalf("violación Zero-PII: el DID apareció en texto plano en el sobre cifrado")
	}
	if bytes.Contains(envelope.Ciphertext, []byte(endpoint)) {
		t.Fatalf("violación Zero-PII: la IP del endpoint apareció en texto plano en el sobre cifrado")
	}

	// 2. Descubrir la baliza con otro receptor
	receiverID, _ := l0.GenerateIdentity()
	receiverManager := NewBlindRendezvousManager(receiverID, store, "test-mesh-cluster-alpha")

	discovered, err := receiverManager.DiscoverPeer(0)
	if err != nil {
		t.Fatalf("error descubriendo baliza ciega: %v", err)
	}

	if discovered.DID != id.DID() {
		t.Errorf("DID esperado %s, obtenido %s", id.DID(), discovered.DID)
	}
	if discovered.Endpoint != endpoint {
		t.Errorf("Endpoint esperado %s, obtenido %s", endpoint, discovered.Endpoint)
	}
}

func TestBlindBeaconConsumeAndBurn(t *testing.T) {
	id, _ := l0.GenerateIdentity()
	store := NewMemoryBlindBeaconStore()
	manager := NewBlindRendezvousManager(id, store, "seed-burn-test")

	_, err := manager.PublishBlindBeacon("10.0.0.2:7001", 1)
	if err != nil {
		t.Fatalf("error publicando: %v", err)
	}

	// Verificar existencia
	topicID := DeriveTopicID(time.Now(), 1, "seed-burn-test")
	if _, err := store.GetBeacon(topicID); err != nil {
		t.Fatalf("la baliza debería existir antes de la quema")
	}

	// Consumir y quemar
	if err := manager.ConsumeAndBurn(1); err != nil {
		t.Fatalf("error en ConsumeAndBurn: %v", err)
	}

	// Debe haber desaparecido
	if _, err := store.GetBeacon(topicID); err == nil {
		t.Errorf("la baliza debería haber sido eliminada tras la quema")
	}
}

func TestBlindBeaconCircuitBreaker(t *testing.T) {
	id, _ := l0.GenerateIdentity()
	store := NewMemoryBlindBeaconStore()
	manager := NewBlindRendezvousManager(id, store, "circuit-seed")

	_, err := manager.PublishBlindBeacon("192.168.1.50:7001", 0)
	if err != nil {
		t.Fatalf("error publicando: %v", err)
	}

	// Conectar primer par directo
	if manager.NotifyDirectPeerConnected() {
		t.Errorf("el circuit breaker no debería saltar con 1 solo par")
	}
	if manager.IsCircuitBroken() {
		t.Errorf("no debería estar en estado roto")
	}

	// Conectar segundo par directo (umbral MinPeerTripThreshold = 2)
	tripped := manager.NotifyDirectPeerConnected()
	if !tripped {
		t.Errorf("el circuit breaker debió saltar al alcanzar 2 pares directos")
	}
	if !manager.IsCircuitBroken() {
		t.Errorf("debe reportar IsCircuitBroken = true")
	}

	// Intentar publicar de nuevo debe ser rechazado
	_, err = manager.PublishBlindBeacon("192.168.1.50:7001", 0)
	if err == nil || !strings.Contains(err.Error(), "circuit breaker activo") {
		t.Errorf("debe rechazar la publicación cuando el circuit breaker está activo, error: %v", err)
	}
}

func TestDeriveTopicIDZeroKnowledge(t *testing.T) {
	t1 := time.Date(2026, 9, 14, 10, 15, 0, 0, time.UTC)
	t2 := time.Date(2026, 9, 14, 10, 45, 0, 0, time.UTC)
	t3 := time.Date(2026, 9, 14, 11, 00, 0, 0, time.UTC)

	topic1 := DeriveTopicID(t1, 0, "seed-zk")
	topic2 := DeriveTopicID(t2, 0, "seed-zk")
	topic3 := DeriveTopicID(t3, 0, "seed-zk")

	if topic1 != topic2 {
		t.Errorf("los tópicos dentro de la misma época horaria deben coincidir: %s != %s", topic1, topic2)
	}
	if topic1 == topic3 {
		t.Errorf("los tópicos en épocas horarias distintas deben ser diferentes")
	}
}
