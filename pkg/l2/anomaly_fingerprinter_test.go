package l2

import (
	"testing"
)

func TestAnomalyFingerprinterCompressionAndDominance(t *testing.T) {
	af := NewAnomalyFingerprinter()

	// 1. Inyectar 100 eventos de Tipo A (Leve)
	for i := 0; i < 100; i++ {
		af.RecordAnomaly("DNS_TIMEOUT_RETRY", "L4_DDNS", "Latencia esporádica en resolución de alias", SeverityLow)
	}

	// 2. Inyectar ráfaga de 1.000.000 eventos de Tipo B (Crítico) usando batch
	af.RecordBatch("BUFFER_OVERFLOW_SATURATION", "L1_TRANSPORT", "Colas intermedias saturadas por bufferbloat", SeverityCritical, 1000000)

	// 3. Inyectar 500 eventos de Tipo C (Medio)
	af.RecordBatch("UNAUTHORIZED_PROBE_ZTNA", "L1_FIREWALL", "Escaneo no autorizado desde identidad foránea", SeverityMedium, 500)

	summary := af.GetExecutiveSummary()

	total := summary["total_events_compressed"].(uint64)
	if total != 1000600 {
		t.Fatalf("Total esperado 1000600, obtenido %d", total)
	}

	distinct := summary["distinct_signatures"].(int)
	if distinct != 3 {
		t.Fatalf("Esperadas 3 firmas únicas, obtenidas %d", distinct)
	}

	dominant := af.GetDominantAnomaly()
	if dominant == nil {
		t.Fatalf("Anomalía dominante no debe ser nil")
	}

	if dominant.ErrorType != "BUFFER_OVERFLOW_SATURATION" {
		t.Fatalf("Esperada anomalía dominante BUFFER_OVERFLOW_SATURATION, obtenida %s", dominant.ErrorType)
	}

	if dominant.Count != 1000000 {
		t.Fatalf("Conteo de anomalía dominante esperado 1000000, obtenido %d", dominant.Count)
	}

	summaryText := summary["summary_text"].(string)
	if len(summaryText) == 0 {
		t.Fatalf("summary_text no debe estar vacío")
	}
}
