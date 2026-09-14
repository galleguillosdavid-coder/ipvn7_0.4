package l4

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"ipvn7/pkg/l0"
)

var (
	ErrAudioStationNotFound = errors.New("estación de radio no encontrada")
)

// AudioStation representa un canal de transmisión de audio puro sin video
type AudioStation struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Genre          string    `json:"genre"`
	BitrateKbps    int       `json:"bitrate_kbps"`
	CurrentTrack   string    `json:"current_track"`
	BroadcasterDID string    `json:"broadcaster_did"`
	ListenersCount int32     `json:"listeners_count"`
	IsLive         bool      `json:"is_live"`
	StartedAt      time.Time `json:"started_at"`

	mu          sync.RWMutex
	subscribers map[uint64]chan []byte
	subSeq      uint64
}

// AudioRadioManager gestiona el ecosistema de radio y transmisión de audio P2P
type AudioRadioManager struct {
	mu        sync.RWMutex
	identity  *l0.Identity
	stations  map[string]*AudioStation
	stopChan  chan struct{}
	isRunning atomic.Bool
}

// NewAudioRadioManager inicializa el motor de audio y estaciones de radio soberanas
func NewAudioRadioManager(id *l0.Identity) *AudioRadioManager {
	mgr := &AudioRadioManager{
		identity: id,
		stations: make(map[string]*AudioStation),
		stopChan: make(chan struct{}),
	}

	// 1. Estaciones predefinidas del Ecosistema Soberano
	mgr.RegisterStation("radio-lofi-01", "Radio Soberana Jazz Lo-Fi", "Lo-Fi / Beats", 96, "Midnight Coding over Mesh", id.DID())
	mgr.RegisterStation("radio-ambient-02", "Frecuencia Cyberpunk 432Hz", "Ambient / Drone", 128, "Resonancia Post-Cuántica", id.DID())
	mgr.RegisterStation("radio-voice-03", "Walkie-Talkie Voz Comunitaria", "Voz Directa P2P", 48, "Canal de Emergencia Abierto", id.DID())

	// Iniciar generador armónico sintético en segundo plano (emula flujo de audio constante)
	mgr.startSyntheticAudioGenerator()

	return mgr
}

// RegisterStation añade una estación de audio a la malla
func (m *AudioRadioManager) RegisterStation(id, name, genre string, bitrate int, track, broadcasterDID string) *AudioStation {
	m.mu.Lock()
	defer m.mu.Unlock()

	st := &AudioStation{
		ID:             id,
		Name:           name,
		Genre:          genre,
		BitrateKbps:    bitrate,
		CurrentTrack:   track,
		BroadcasterDID: broadcasterDID,
		IsLive:         true,
		StartedAt:      time.Now(),
		subscribers:    make(map[uint64]chan []byte),
	}
	m.stations[id] = st
	return st
}

// ListStations devuelve la lista completa de estaciones disponibles
func (m *AudioRadioManager) ListStations() []*AudioStation {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make([]*AudioStation, 0, len(m.stations))
	for _, st := range m.stations {
		res = append(res, st)
	}
	return res
}

// GetStation obtiene una estación por ID
func (m *AudioRadioManager) GetStation(id string) (*AudioStation, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	st, exists := m.stations[id]
	if !exists {
		return nil, ErrAudioStationNotFound
	}
	return st, nil
}

// BroadcastAudio inyecta audio en vivo desde un emisor (micrófono del usuario)
func (m *AudioRadioManager) BroadcastAudio(stationID string, audioData []byte) error {
	st, err := m.GetStation(stationID)
	if err != nil {
		return err
	}

	st.mu.RLock()
	defer st.mu.RUnlock()

	for _, ch := range st.subscribers {
		select {
		case ch <- audioData:
		default:
		}
	}
	return nil
}

// SubscribeStation suscribe un oyente para recibir el flujo de audio binario
func (m *AudioRadioManager) SubscribeStation(stationID string) (chan []byte, func(), error) {
	st, err := m.GetStation(stationID)
	if err != nil {
		return nil, nil, err
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	st.subSeq++
	subID := st.subSeq
	audioChan := make(chan []byte, 30)
	st.subscribers[subID] = audioChan
	atomic.AddInt32(&st.ListenersCount, 1)

	cleanup := func() {
		st.mu.Lock()
		defer st.mu.Unlock()
		if _, exists := st.subscribers[subID]; exists {
			delete(st.subscribers, subID)
			close(audioChan)
			atomic.AddInt32(&st.ListenersCount, -1)
		}
	}

	return audioChan, cleanup, nil
}

// startSyntheticAudioGenerator genera armónicos matemáticos periódicos para las radios
func (m *AudioRadioManager) startSyntheticAudioGenerator() {
	if m.isRunning.Swap(true) {
		return
	}

	go func() {
		ticker := time.NewTicker(200 * time.Millisecond)
		defer ticker.Stop()

		var sampleIndex float64
		sampleRate := 44100.0
		chunkDuration := 0.2 // 200 ms

		for {
			select {
			case <-m.stopChan:
				return
			case <-ticker.C:
				// Generar 200ms de onda senoidal suave (tono armónico relajante en 432 Hz)
				pcmChunk := generateHarmonicPCM(432.0, sampleRate, chunkDuration, &sampleIndex)

				m.mu.RLock()
				for _, st := range m.stations {
					st.mu.RLock()
					for _, ch := range st.subscribers {
						select {
						case ch <- pcmChunk:
						default:
						}
					}
					st.mu.RUnlock()
				}
				m.mu.RUnlock()
			}
		}
	}()
}

// Close apaga el generador de audio
func (m *AudioRadioManager) Close() {
	if m.isRunning.Swap(false) {
		close(m.stopChan)
	}
}

// generateHarmonicPCM genera un trozo binario PCM a 16-bit mono
func generateHarmonicPCM(freq, sampleRate, duration float64, index *float64) []byte {
	numSamples := int(sampleRate * duration)
	buf := new(bytes.Buffer)

	for i := 0; i < numSamples; i++ {
		t := (*index) / sampleRate
		// Combinación armónica fundamental + sub-octava cálida
		val := 0.4*math.Sin(2*math.Pi*freq*t) + 0.2*math.Sin(2*math.Pi*(freq/2)*t)
		sample := int16(val * 32767.0)
		_ = binary.Write(buf, binary.LittleEndian, sample)
		*index++
	}
	return buf.Bytes()
}
