package l1

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// PacerConfig define los parámetros operativos del marcapasos de paquetes
type PacerConfig struct {
	BottleneckRateBytesPerSec uint64        // Capacidad estimada del cuello de botella
	SafetyMarginPercent       uint8         // Margen de seguridad (ej. 10%)
	MaxBurstPackets           int           // Límite estricto de ráfaga para evitar bufferbloat
	MinInterval               time.Duration // Intervalo mínimo entre paquetes consecutivos
}

// DefaultPacerConfig retorna la configuración estándar para enlaces heterogéneos
func DefaultPacerConfig() PacerConfig {
	return PacerConfig{
		BottleneckRateBytesPerSec: 10 * 1024 * 1024 / 8, // 10 Mbps en bytes/seg por defecto
		SafetyMarginPercent:       10,                  // 10% margen de seguridad
		MaxBurstPackets:           1,                   // Ráfaga unitaria (flujo constante reloj-estricto)
		MinInterval:               100 * time.Microsecond,
	}
}

// PacketPacer implementa el marcapasos de red subordinado al cuello de botella de la ruta
type PacketPacer struct {
	mu           sync.RWMutex
	config       PacerConfig
	effectiveRate uint64 // Tasa sostenible efectiva calculada (tasa - margen)
	intervalPerPacket time.Duration
	lastPacketTime atomic.Int64
	packetsSent    atomic.Uint64
	bytesSent      atomic.Uint64
	throttledCount atomic.Uint64
}

// NewPacketPacer inicializa el marcapasos de red con la configuración de flujo sostenible
func NewPacketPacer(cfg PacerConfig) *PacketPacer {
	p := &PacketPacer{
		config: cfg,
	}
	p.recalculateInterval(cfg.BottleneckRateBytesPerSec)
	return p
}

// UpdateBottleneckRate actualiza la capacidad conocida de la ruta ante advertencias de congestión (Backpressure)
func (p *PacketPacer) UpdateBottleneckRate(rateBytesPerSec uint64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.config.BottleneckRateBytesPerSec = rateBytesPerSec
	p.recalculateInterval(rateBytesPerSec)
}

func (p *PacketPacer) recalculateInterval(rateBytesPerSec uint64) {
	if rateBytesPerSec == 0 {
		rateBytesPerSec = 1024 * 128 // Mínimo 128 KB/s de seguridad
	}

	// Subordinación al cuello de botella menos el margen de seguridad
	margin := (rateBytesPerSec * uint64(p.config.SafetyMarginPercent)) / 100
	p.effectiveRate = rateBytesPerSec - margin

	// Asumiendo datagramas de 1280 bytes deterministas
	packetsPerSec := float64(p.effectiveRate) / 1280.0
	if packetsPerSec <= 0 {
		packetsPerSec = 1.0
	}

	intervalNs := float64(time.Second) / packetsPerSec
	interval := time.Duration(intervalNs)
	if interval < p.config.MinInterval {
		interval = p.config.MinInterval
	}
	p.intervalPerPacket = interval
}

// Pace espera el tiempo matemáticamente exacto antes de permitir el despacho del siguiente paquete
func (p *PacketPacer) Pace(ctx context.Context, packetSizeBytes int) error {
	p.mu.RLock()
	interval := p.intervalPerPacket
	p.mu.RUnlock()

	now := time.Now().UnixNano()
	last := p.lastPacketTime.Load()

	targetTime := last + interval.Nanoseconds()
	if now < targetTime {
		diff := time.Duration(targetTime - now)
		p.throttledCount.Add(1)

		select {
		case <-time.After(diff):
		case <-ctx.Done():
			return errors.New("envío cancelado por contexto mientras esperaba en marcapasos")
		}
	}

	p.lastPacketTime.Store(time.Now().UnixNano())
	p.packetsSent.Add(1)
	p.bytesSent.Add(uint64(packetSizeBytes))
	return nil
}

// PacerStats retorna estadísticas de rendimiento del marcapasos
type PacerStats struct {
	EffectiveRateBytesPerSec uint64        `json:"effective_rate_bps"`
	PacketIntervalUs         int64         `json:"packet_interval_us"`
	PacketsSent              uint64        `json:"packets_sent"`
	BytesSent                uint64        `json:"bytes_sent"`
	ThrottledCount           uint64        `json:"throttled_count"`
}

func (p *PacketPacer) Stats() PacerStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PacerStats{
		EffectiveRateBytesPerSec: p.effectiveRate,
		PacketIntervalUs:         p.intervalPerPacket.Microseconds(),
		PacketsSent:              p.packetsSent.Load(),
		BytesSent:                p.bytesSent.Load(),
		ThrottledCount:           p.throttledCount.Load(),
	}
}

// String formatea las métricas del marcapasos
func (p *PacketPacer) String() string {
	s := p.Stats()
	return fmt.Sprintf("PacketPacer[Rate: %d B/s, Interval: %d µs, Sent: %d, Throttled: %d]",
		s.EffectiveRateBytesPerSec, s.PacketIntervalUs, s.PacketsSent, s.ThrottledCount)
}
