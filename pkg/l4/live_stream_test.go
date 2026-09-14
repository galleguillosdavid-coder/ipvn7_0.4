package l4

import (
	"testing"

	"ipvn7/pkg/l0"
)

func TestLiveStreamChannelAndSubscription(t *testing.T) {
	id, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("error generando identidad: %v", err)
	}

	mgr := NewLiveStreamManager(id)
	ch := mgr.RegisterChannel("test-chan-01", "Canal de Prueba", id.DID(), "VP9", "Opus")
	if ch.ID != "test-chan-01" {
		t.Fatalf("esperado canal test-chan-01, obtenido %s", ch.ID)
	}

	// 1. Suscribirse
	subCh, cleanup, err := mgr.Subscribe("test-chan-01")
	if err != nil {
		t.Fatalf("error suscribiendo al canal: %v", err)
	}
	defer cleanup()

	// 2. Ingerir un segmento
	payload := []byte("VP9_VIDEO_FRAME_BYTES_1280B")
	seg, err := mgr.IngestSegment("test-chan-01", id.DID(), true, "video/webm", payload)
	if err != nil {
		t.Fatalf("error ingiriendo segmento: %v", err)
	}

	// 3. Recibir el segmento
	received := <-subCh
	if received.Sequence != seg.Sequence {
		t.Fatalf("secuencia esperada %d, obtenida %d", seg.Sequence, received.Sequence)
	}

	if string(received.Data) != string(payload) {
		t.Fatalf("datos recibidos no coinciden con el payload enviado")
	}

	// 4. Comprobar conteo de espectadores
	if ch.ViewersCount < 1 {
		t.Errorf("ViewersCount esperado >= 1, obtenido %d", ch.ViewersCount)
	}
}

func TestLiveStreamUnauthorizedBroadcaster(t *testing.T) {
	id, _ := l0.GenerateIdentity()
	alienID, _ := l0.GenerateIdentity()

	mgr := NewLiveStreamManager(id)
	mgr.RegisterChannel("private-01", "Canal Privado", id.DID(), "AV1", "Opus")

	_, err := mgr.IngestSegment("private-01", alienID.DID(), false, "video/webm", []byte("bad"))
	if err != ErrStreamUnauthorized {
		t.Fatalf("se esperaba ErrStreamUnauthorized, obtenido %v", err)
	}
}
