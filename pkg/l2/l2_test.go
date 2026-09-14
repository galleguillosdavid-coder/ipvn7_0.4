package l2_test

import (
	"strings"
	"testing"

	"ipvn7/pkg/l2"
)

func TestTelemetryRingBuffer(t *testing.T) {
	rb := l2.NewTelemetryRingBuffer()

	// Registrar eventos
	rb.RecordEvent(l2.EventTxPacket, 1024, 0, 0x1234)
	rb.RecordEvent(l2.EventTxPacket, 512, 0, 0x1234)
	rb.RecordEvent(l2.EventRxPacket, 2048, 1500, 0x5678)
	rb.RecordEvent(l2.EventDrop, 0, 0, 0x9999)

	snapshot := rb.Snapshot()
	if snapshot.PacketsTx != 2 {
		t.Errorf("PacketsTx esperados: 2, obtenidos: %d", snapshot.PacketsTx)
	}
	if snapshot.PacketsRx != 1 {
		t.Errorf("PacketsRx esperados: 1, obtenidos: %d", snapshot.PacketsRx)
	}
	if snapshot.BytesTx != 1536 {
		t.Errorf("BytesTx esperados: 1536, obtenidos: %d", snapshot.BytesTx)
	}
	if snapshot.PacketsDropped != 1 {
		t.Errorf("PacketsDropped esperados: 1, obtenidos: %d", snapshot.PacketsDropped)
	}
	if snapshot.AvgLatencyMs <= 0 {
		t.Errorf("AvgLatencyMs debió ser mayor a 0: %f", snapshot.AvgLatencyMs)
	}

	// Comprobar reporte OpenMetrics
	om := rb.ToOpenMetrics()
	if !strings.Contains(om, "ipvn7_packets_total{direction=\"tx\"} 2") {
		t.Errorf("Reporte OpenMetrics incompleto o incorrecto:\n%s", om)
	}
}
