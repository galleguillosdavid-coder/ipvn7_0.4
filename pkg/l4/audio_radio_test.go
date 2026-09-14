package l4

import (
	"testing"
	"time"

	"ipvn7/pkg/l0"
)

func TestAudioRadioStationsAndStreaming(t *testing.T) {
	id, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("error generando identidad: %v", err)
	}

	mgr := NewAudioRadioManager(id)
	defer mgr.Close()

	stations := mgr.ListStations()
	if len(stations) < 3 {
		t.Fatalf("se esperaban al menos 3 estaciones, obtenidas %d", len(stations))
	}

	st, err := mgr.GetStation("radio-lofi-01")
	if err != nil {
		t.Fatalf("error obteniendo estación: %v", err)
	}

	// 1. Suscribirse a la estación
	audioChan, cleanup, err := mgr.SubscribeStation(st.ID)
	if err != nil {
		t.Fatalf("error suscribiendo: %v", err)
	}
	defer cleanup()

	// 2. Verificar que se recibe audio en el canal (sintético o broadcast)
	select {
	case chunk := <-audioChan:
		if len(chunk) == 0 {
			t.Errorf("trozo de audio recibido está vacío")
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("tiempo de espera agotado esperando audio de la estación")
	}

	// 3. Probar inyección directa de audio (micrófono)
	micData := []byte("MIC_AUDIO_OPUS_SAMPLE_48KHZ")
	err = mgr.BroadcastAudio(st.ID, micData)
	if err != nil {
		t.Fatalf("error emitiendo audio: %v", err)
	}
}
