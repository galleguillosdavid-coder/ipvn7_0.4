package l2

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// SeverityLevel clasifica la criticidad funcional de una anomalía
type SeverityLevel string

const (
	SeverityCritical SeverityLevel = "CRITICAL"
	SeverityHigh     SeverityLevel = "HIGH"
	SeverityMedium   SeverityLevel = "MEDIUM"
	SeverityLow      SeverityLevel = "LOW"
)

// ErrorFingerprint compacta millones de eventos bajo una única firma digital (2.md Líneas 729-738)
type ErrorFingerprint struct {
	Signature   string        `json:"signature"`    // Hash SHA-256 (primeros 12 caracteres hex)
	ErrorType   string        `json:"error_type"`   // ej. BUFFER_OVERFLOW, ZTNA_UNAUTHORIZED_PROBE
	Subsystem   string        `json:"subsystem"`    // ej. L1_TRANSPORT, L1_FIREWALL, L1_QOS
	RootCause   string        `json:"root_cause"`   // Causa raíz probable
	Severity    SeverityLevel `json:"severity"`     // Criticidad
	Count       uint64        `json:"count"`        // Contador atómico
	FirstSeen   time.Time     `json:"first_seen"`
	LastSeen    time.Time     `json:"last_seen"`
	lastSeenRaw int64         // UnixNano atómico para actualización lock-free
}

// AnomalyFingerprinter implementa la telemetría de fallas silenciosa y agregada por prioridad (L2)
// Cumple estrictamente con genesis.md Línea 177: descarte y agregación O(1) sin presión a la red.
type AnomalyFingerprinter struct {
	mu           sync.RWMutex
	fingerprints map[string]*ErrorFingerprint
	totalCount   uint64
}

// NewAnomalyFingerprinter inicializa el agregador de huellas digitales de anomalías
func NewAnomalyFingerprinter() *AnomalyFingerprinter {
	return &AnomalyFingerprinter{
		fingerprints: make(map[string]*ErrorFingerprint),
		totalCount:   0,
	}
}

// ComputeSignature genera una huella criptográfica determinista
func ComputeSignature(errType, subsystem, rootCause string) string {
	h := sha256.New()
	h.Write([]byte(errType))
	h.Write([]byte(":"))
	h.Write([]byte(subsystem))
	h.Write([]byte(":"))
	h.Write([]byte(rootCause))
	sum := h.Sum(nil)
	return hex.EncodeToString(sum[:6]) // 12 caracteres hex únicos
}

// RecordAnomaly registra o compacta un evento de fallo en tiempo O(1)
func (af *AnomalyFingerprinter) RecordAnomaly(errType, subsystem, rootCause string, severity SeverityLevel) *ErrorFingerprint {
	sig := ComputeSignature(errType, subsystem, rootCause)
	now := time.Now()
	nowNano := now.UnixNano()

	// Fast-path de lectura
	af.mu.RLock()
	fp, exists := af.fingerprints[sig]
	af.mu.RUnlock()

	if exists {
		atomic.AddUint64(&fp.Count, 1)
		atomic.AddUint64(&af.totalCount, 1)
		atomic.StoreInt64(&fp.lastSeenRaw, nowNano)
		fp.LastSeen = now
		return fp
	}

	// Inserción bajo bloqueo de escritura
	af.mu.Lock()
	defer af.mu.Unlock()

	// Re-verificar tras adquirir lock
	if fp, exists = af.fingerprints[sig]; exists {
		atomic.AddUint64(&fp.Count, 1)
		atomic.AddUint64(&af.totalCount, 1)
		atomic.StoreInt64(&fp.lastSeenRaw, nowNano)
		fp.LastSeen = now
		return fp
	}

	newFp := &ErrorFingerprint{
		Signature:   sig,
		ErrorType:   errType,
		Subsystem:   subsystem,
		RootCause:   rootCause,
		Severity:    severity,
		Count:       1,
		FirstSeen:   now,
		LastSeen:    now,
		lastSeenRaw: nowNano,
	}

	af.fingerprints[sig] = newFp
	atomic.AddUint64(&af.totalCount, 1)
	return newFp
}

// RecordBatch permite simular o compactar ráfagas masivas en memoria instantáneamente
func (af *AnomalyFingerprinter) RecordBatch(errType, subsystem, rootCause string, severity SeverityLevel, delta uint64) *ErrorFingerprint {
	if delta == 0 {
		delta = 1
	}
	sig := ComputeSignature(errType, subsystem, rootCause)
	now := time.Now()

	af.mu.Lock()
	defer af.mu.Unlock()

	fp, exists := af.fingerprints[sig]
	if exists {
		atomic.AddUint64(&fp.Count, delta)
		atomic.AddUint64(&af.totalCount, delta)
		fp.LastSeen = now
		return fp
	}

	newFp := &ErrorFingerprint{
		Signature:   sig,
		ErrorType:   errType,
		Subsystem:   subsystem,
		RootCause:   rootCause,
		Severity:    severity,
		Count:       delta,
		FirstSeen:   now,
		LastSeen:    now,
		lastSeenRaw: now.UnixNano(),
	}

	af.fingerprints[sig] = newFp
	atomic.AddUint64(&af.totalCount, delta)
	return newFp
}

// GetDominantAnomaly identifica la falla con mayor prioridad matemática (volumen * severidad)
func (af *AnomalyFingerprinter) GetDominantAnomaly() *ErrorFingerprint {
	af.mu.RLock()
	defer af.mu.RUnlock()

	if len(af.fingerprints) == 0 {
		return nil
	}

	var dominant *ErrorFingerprint
	var maxScore uint64

	for _, fp := range af.fingerprints {
		c := atomic.LoadUint64(&fp.Count)
		multiplier := uint64(1)
		switch fp.Severity {
		case SeverityCritical:
			multiplier = 10
		case SeverityHigh:
			multiplier = 5
		case SeverityMedium:
			multiplier = 2
		case SeverityLow:
			multiplier = 1
		}

		score := c * multiplier
		if score > maxScore || dominant == nil {
			maxScore = score
			dominant = fp
		}
	}

	return dominant
}

// GetExecutiveSummary genera el informe limpio y desprovisto de ruido técnico
func (af *AnomalyFingerprinter) GetExecutiveSummary() map[string]interface{} {
	af.mu.RLock()
	defer af.mu.RUnlock()

	total := atomic.LoadUint64(&af.totalCount)
	dominant := af.GetDominantAnomaly()

	list := make([]*ErrorFingerprint, 0, len(af.fingerprints))
	for _, fp := range af.fingerprints {
		list = append(list, fp)
	}

	// Ordenar por volumen descendente
	sort.Slice(list, func(i, j int) bool {
		return atomic.LoadUint64(&list[i].Count) > atomic.LoadUint64(&list[j].Count)
	})

	var summaryText string
	if dominant != nil {
		summaryText = fmt.Sprintf("La anomalía dominante es [%s] con %d casos registrados en el carril %s (Causa: %s).",
			dominant.ErrorType, atomic.LoadUint64(&dominant.Count), dominant.Subsystem, dominant.RootCause)
	} else {
		summaryText = "Malla operando dentro de los parámetros estables. Cero anomalías registradas."
	}

	return map[string]interface{}{
		"total_events_compressed": total,
		"distinct_signatures":     len(af.fingerprints),
		"dominant_anomaly":        dominant,
		"summary_text":            summaryText,
		"top_fingerprints":        list,
		"timestamp":               time.Now().UTC().Format(time.RFC3339),
	}
}
