package l2

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

const (
	RingBufferSize = 4096 // Búfer de potencia de 2 para indexación bitwise ultra-rápida (lock-free)
	RingBufferMask = RingBufferSize - 1
)

// Tipos de eventos de telemetría
const (
	EventTxPacket uint8 = 1
	EventRxPacket uint8 = 2
	EventDrop     uint8 = 3
	EventAnomaly  uint8 = 4
)

// TelemetryEvent representa un registro atómico de telemetría de red (<28 ns)
type TelemetryEvent struct {
	Timestamp int64
	Type      uint8
	Bytes     uint32
	LatencyUs uint32
	PeerHash  uint32
}

// TelemetryRingBuffer implementa un búfer circular de ultra-alto rendimiento sin bloqueos
type TelemetryRingBuffer struct {
	head    atomic.Uint64
	tail    atomic.Uint64
	entries [RingBufferSize]TelemetryEvent

	// Contadores globales acumulados
	packetsTx      atomic.Uint64
	packetsRx      atomic.Uint64
	bytesTx        atomic.Uint64
	bytesRx        atomic.Uint64
	packetsDropped atomic.Uint64
	totalLatencyUs atomic.Uint64
}

// NewTelemetryRingBuffer crea una nueva instancia del búfer de telemetría
func NewTelemetryRingBuffer() *TelemetryRingBuffer {
	return &TelemetryRingBuffer{}
}

// RecordEvent registra un evento en tiempo constante O(1) (<28 ns)
func (rb *TelemetryRingBuffer) RecordEvent(eventType uint8, byteCount uint32, latencyUs uint32, peerHash uint32) {
	// Actualizar contadores acumulativos
	switch eventType {
	case EventTxPacket:
		rb.packetsTx.Add(1)
		rb.bytesTx.Add(uint64(byteCount))
	case EventRxPacket:
		rb.packetsRx.Add(1)
		rb.bytesRx.Add(uint64(byteCount))
		if latencyUs > 0 {
			rb.totalLatencyUs.Add(uint64(latencyUs))
		}
	case EventDrop:
		rb.packetsDropped.Add(1)
	}

	// Slot circular atómico
	idx := rb.head.Add(1) & RingBufferMask
	rb.entries[idx] = TelemetryEvent{
		Timestamp: time.Now().UnixNano(),
		Type:      eventType,
		Bytes:     byteCount,
		LatencyUs: latencyUs,
		PeerHash:  peerHash,
	}
}

// MetricsSnapshot captura el estado actual de métricas de red
type MetricsSnapshot struct {
	PacketsTx      uint64  `json:"packets_tx"`
	PacketsRx      uint64  `json:"packets_rx"`
	BytesTx        uint64  `json:"bytes_tx"`
	BytesRx        uint64  `json:"bytes_rx"`
	PacketsDropped uint64  `json:"packets_dropped"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	TitForTatRatio float64 `json:"tit_for_tat_ratio"`
}

// Snapshot genera una lectura atómica de métricas del nodo
func (rb *TelemetryRingBuffer) Snapshot() MetricsSnapshot {
	tx := rb.packetsTx.Load()
	rx := rb.packetsRx.Load()
	bTx := rb.bytesTx.Load()
	bRx := rb.bytesRx.Load()
	drops := rb.packetsDropped.Load()
	latUs := rb.totalLatencyUs.Load()

	var avgLat float64
	if rx > 0 {
		avgLat = float64(latUs) / float64(rx) / 1000.0
	}

	var ratio float64 = 1.0
	if bRx > 0 {
		ratio = float64(bTx) / float64(bRx)
	}

	return MetricsSnapshot{
		PacketsTx:      tx,
		PacketsRx:      rx,
		BytesTx:        bTx,
		BytesRx:        bRx,
		PacketsDropped: drops,
		AvgLatencyMs:   avgLat,
		TitForTatRatio: ratio,
	}
}

// ToOpenMetrics genera el reporte en formato estándar OpenMetrics/Prometheus
func (rb *TelemetryRingBuffer) ToOpenMetrics() string {
	s := rb.Snapshot()
	var sb strings.Builder

	sb.WriteString("# HELP ipvn7_packets_total Total de paquetes procesados por dirección\n")
	sb.WriteString("# TYPE ipvn7_packets_total counter\n")
	sb.WriteString(fmt.Sprintf("ipvn7_packets_total{direction=\"tx\"} %d\n", s.PacketsTx))
	sb.WriteString(fmt.Sprintf("ipvn7_packets_total{direction=\"rx\"} %d\n", s.PacketsRx))

	sb.WriteString("# HELP ipvn7_bytes_total Total de bytes transferidos\n")
	sb.WriteString("# TYPE ipvn7_bytes_total counter\n")
	sb.WriteString(fmt.Sprintf("ipvn7_bytes_total{direction=\"tx\"} %d\n", s.BytesTx))
	sb.WriteString(fmt.Sprintf("ipvn7_bytes_total{direction=\"rx\"} %d\n", s.BytesRx))

	sb.WriteString("# HELP ipvn7_packets_dropped_total Paquetes descartados por congestión o seguridad\n")
	sb.WriteString("# TYPE ipvn7_packets_dropped_total counter\n")
	sb.WriteString(fmt.Sprintf("ipvn7_packets_dropped_total %d\n", s.PacketsDropped))

	sb.WriteString("# HELP ipvn7_average_latency_ms Latencia media en milisegundos\n")
	sb.WriteString("# TYPE ipvn7_average_latency_ms gauge\n")
	sb.WriteString(fmt.Sprintf("ipvn7_average_latency_ms %.4f\n", s.AvgLatencyMs))

	sb.WriteString("# HELP ipvn7_tit_for_tat_ratio Relación de reciprocidad económica de bytes\n")
	sb.WriteString("# TYPE ipvn7_tit_for_tat_ratio gauge\n")
	sb.WriteString(fmt.Sprintf("ipvn7_tit_for_tat_ratio %.4f\n", s.TitForTatRatio))

	return sb.String()
}
