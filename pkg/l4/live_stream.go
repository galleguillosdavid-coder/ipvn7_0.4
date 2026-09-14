package l4

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"ipvn7/pkg/l0"
)

var (
	ErrStreamChannelNotFound = errors.New("canal de transmisión no encontrado")
	ErrStreamUnauthorized    = errors.New("emisor no autorizado para este canal")
	ErrStreamBufferFull      = errors.New("búfer de transmisión saturado")
)

const (
	MaxStreamRingBuffer = 30 // Últimos 30 segmentos retenidos en memoria
	MaxSubscribersPerCh = 256
)

// StreamSegment representa una trama o trozo de audio/video en tránsito
type StreamSegment struct {
	Sequence  uint64    `json:"sequence"`
	Timestamp time.Time `json:"timestamp"`
	Keyframe  bool      `json:"keyframe"`
	MimeType  string    `json:"mime_type"` // ej. "video/webm; codecs=vp8,opus"
	Data      []byte    `json:"data"`
}

// StreamChannel representa una transmisión en vivo activa en la malla soberana
type StreamChannel struct {
	ID             string    `json:"id"`
	Title          string    `json:"title"`
	BroadcasterDID string    `json:"broadcaster_did"`
	StartedAt      time.Time `json:"started_at"`
	BitrateKbps    int       `json:"bitrate_kbps"`
	ViewersCount   int32     `json:"viewers_count"`
	IsActive       bool      `json:"is_active"`
	VideoCodec     string    `json:"video_codec"`
	AudioCodec     string    `json:"audio_codec"`

	mu          sync.RWMutex
	ringBuffer  []*StreamSegment
	ringHead    int
	subscribers map[uint64]chan *StreamSegment
	subSeq      uint64
	seqCounter  uint64
}

// LiveStreamManager administra las transmisiones en vivo P2P de la malla
type LiveStreamManager struct {
	mu       sync.RWMutex
	identity *l0.Identity
	channels map[string]*StreamChannel
}

// NewLiveStreamManager crea una nueva instancia del gestor de streaming en vivo
func NewLiveStreamManager(id *l0.Identity) *LiveStreamManager {
	mgr := &LiveStreamManager{
		identity: id,
		channels: make(map[string]*StreamChannel),
	}

	// Canal demostración soberano por defecto
	mgr.RegisterChannel("live-sovereign-01", "Transmisión Soberana 4K Mesh", id.DID(), "VP9/AV1", "Opus")
	return mgr
}

// RegisterChannel crea o activa un canal de transmisión
func (m *LiveStreamManager) RegisterChannel(id, title, broadcasterDID, vCodec, aCodec string) *StreamChannel {
	m.mu.Lock()
	defer m.mu.Unlock()

	if ch, exists := m.channels[id]; exists {
		ch.mu.Lock()
		ch.Title = title
		ch.BroadcasterDID = broadcasterDID
		ch.IsActive = true
		ch.mu.Unlock()
		return ch
	}

	ch := &StreamChannel{
		ID:             id,
		Title:          title,
		BroadcasterDID: broadcasterDID,
		StartedAt:      time.Now(),
		BitrateKbps:    4500,
		IsActive:       true,
		VideoCodec:     vCodec,
		AudioCodec:     aCodec,
		ringBuffer:     make([]*StreamSegment, 0, MaxStreamRingBuffer),
		subscribers:    make(map[uint64]chan *StreamSegment),
	}
	m.channels[id] = ch
	return ch
}

// GetChannel obtiene un canal por su ID
func (m *LiveStreamManager) GetChannel(id string) (*StreamChannel, error) {
	m.mu.RUnlock()
	defer m.mu.RUnlock()

	ch, exists := m.channels[id]
	if !exists {
		return nil, ErrStreamChannelNotFound
	}
	return ch, nil
}

// ListChannels devuelve todos los canales activos
func (m *LiveStreamManager) ListChannels() []*StreamChannel {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make([]*StreamChannel, 0, len(m.channels))
	for _, ch := range m.channels {
		res = append(res, ch)
	}
	return res
}

// IngestSegment inyecta un nuevo trozo multimedia en el canal y lo difunde a los suscriptores
func (m *LiveStreamManager) IngestSegment(channelID, broadcasterDID string, keyframe bool, mimeType string, data []byte) (*StreamSegment, error) {
	m.mu.RLock()
	ch, exists := m.channels[channelID]
	m.mu.RUnlock()

	if !exists {
		return nil, ErrStreamChannelNotFound
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.BroadcasterDID != "" && ch.BroadcasterDID != broadcasterDID {
		return nil, ErrStreamUnauthorized
	}

	ch.seqCounter++
	seg := &StreamSegment{
		Sequence:  ch.seqCounter,
		Timestamp: time.Now(),
		Keyframe:  keyframe,
		MimeType:  mimeType,
		Data:      data,
	}

	// Agregar al ring buffer circular
	if len(ch.ringBuffer) < MaxStreamRingBuffer {
		ch.ringBuffer = append(ch.ringBuffer, seg)
	} else {
		ch.ringBuffer[ch.ringHead] = seg
		ch.ringHead = (ch.ringHead + 1) % MaxStreamRingBuffer
	}

	// Difundir sin bloqueo a los suscriptores
	for _, subCh := range ch.subscribers {
		select {
		case subCh <- seg:
		default:
			// Descartar si el suscriptor está saturado para preservar latencia en tiempo real
		}
	}

	return seg, nil
}

// Subscribe permite a un cliente recibir segmentos en tiempo real
func (m *LiveStreamManager) Subscribe(channelID string) (chan *StreamSegment, func(), error) {
	m.mu.RLock()
	ch, exists := m.channels[channelID]
	m.mu.RUnlock()

	if !exists {
		return nil, nil, ErrStreamChannelNotFound
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	ch.subSeq++
	subID := ch.subSeq
	streamChan := make(chan *StreamSegment, 20)
	ch.subscribers[subID] = streamChan
	atomic.AddInt32(&ch.ViewersCount, 1)

	// Entregar los últimos segmentos clave del búfer circular para arranque instantáneo
	for _, prevSeg := range ch.ringBuffer {
		if prevSeg != nil {
			select {
			case streamChan <- prevSeg:
			default:
			}
		}
	}

	cleanup := func() {
		ch.mu.Lock()
		defer ch.mu.Unlock()
		if _, ok := ch.subscribers[subID]; ok {
			delete(ch.subscribers, subID)
			close(streamChan)
			atomic.AddInt32(&ch.ViewersCount, -1)
		}
	}

	return streamChan, cleanup, nil
}
