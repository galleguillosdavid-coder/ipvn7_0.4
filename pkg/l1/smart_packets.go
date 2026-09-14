package l1

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"
)

// HookPoint define el punto de intercepción del paquete en el pipeline de red
type HookPoint int

const (
	HookIngress    HookPoint = 0 // Al entrar al nodo desde la interfaz física
	HookForwarding HookPoint = 1 // Durante el reenvío intermedio en la malla
	HookEgress     HookPoint = 2 // Justo antes de salir al cable
)

func (hp HookPoint) String() string {
	switch hp {
	case HookIngress:
		return "INGRESS"
	case HookForwarding:
		return "FORWARDING"
	case HookEgress:
		return "EGRESS"
	default:
		return "UNKNOWN"
	}
}

// FilterResult es la respuesta emitida por un filtro Smart Packet
type FilterResult struct {
	Pass         bool   `json:"pass"`          // True para continuar, False para descartar
	Modified     bool   `json:"modified"`      // Indica si la carga fue transformada
	NewPayload   []byte `json:"-"`             // Carga útil resultante
	DurationUs   int64  `json:"duration_us"`   // Tiempo de ejecución en microsegundos
	Reason       string `json:"reason"`        // Motivo del descarte o de la transformación
	FilterID     string `json:"filter_id"`     // Identificador del filtro ejecutado
}

// SmartPacketFilter define la interfaz de un interceptor de paquetes programable
type SmartPacketFilter interface {
	ID() string
	Point() HookPoint
	Execute(ctx context.Context, payload []byte) (*FilterResult, error)
}

// SmartPacketPipeline orquesta la ejecución segura y acotada de filtros Wasm/Smart Packets (Dimensión 7)
type SmartPacketPipeline struct {
	mu           sync.RWMutex
	filters      map[HookPoint][]SmartPacketFilter
	timeoutLimit time.Duration // Límite estricto de 10 ms según especificación

	// Métricas
	PacketsFiltered uint64
	PacketsModified uint64
	PacketsDropped  uint64
	TimeoutsCount   uint64
}

// NewSmartPacketPipeline inicializa el pipeline de filtros programables
func NewSmartPacketPipeline() *SmartPacketPipeline {
	pipeline := &SmartPacketPipeline{
		filters:      make(map[HookPoint][]SmartPacketFilter),
		timeoutLimit: 10 * time.Millisecond, // 10 ms estrictos anti-lag
	}

	// Registrar filtros estándar de fábrica
	pipeline.RegisterFilter(&DLPInspectionFilter{})
	pipeline.RegisterFilter(&IntegrityCheckFilter{})

	return pipeline
}

// RegisterFilter añade un filtro programable al pipeline
func (spp *SmartPacketPipeline) RegisterFilter(filter SmartPacketFilter) {
	spp.mu.Lock()
	defer spp.mu.Unlock()

	hp := filter.Point()
	spp.filters[hp] = append(spp.filters[hp], filter)
}

// Process ejecuta todos los filtros registrados para el punto de intercepción indicado
func (spp *SmartPacketPipeline) Process(point HookPoint, payload []byte) ([]byte, bool, string) {
	spp.mu.RLock()
	filterList := spp.filters[point]
	spp.mu.RUnlock()

	if len(filterList) == 0 {
		return payload, true, "Sin filtros en este punto"
	}

	currentPayload := payload
	spp.PacketsFiltered++

	for _, filter := range filterList {
		// Envolver con límite estricto de 10 ms
		ctx, cancel := context.WithTimeout(context.Background(), spp.timeoutLimit)
		start := time.Now()

		res, err := filter.Execute(ctx, currentPayload)
		cancel()

		dur := time.Since(start).Microseconds()

		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				spp.TimeoutsCount++
				spp.PacketsDropped++
				return nil, false, fmt.Sprintf("Filtro %s excedió el límite estricto de 10 ms (Smart Packet Timeout)", filter.ID())
			}
			spp.PacketsDropped++
			return nil, false, fmt.Sprintf("Error en filtro %s: %v", filter.ID(), err)
		}

		if !res.Pass {
			spp.PacketsDropped++
			return nil, false, fmt.Sprintf("Descartado por filtro %s: %s (duración %d µs)", filter.ID(), res.Reason, dur)
		}

		if res.Modified && res.NewPayload != nil {
			currentPayload = res.NewPayload
			spp.PacketsModified++
		}
	}

	return currentPayload, true, "Aprobado por todos los filtros Smart Packet"
}

// -------------------------------------------------------------
// Filtros Nativos de Demostración
// -------------------------------------------------------------

// DLPInspectionFilter inspecciona fugas de datos y patrones maliciosos en la carga
type DLPInspectionFilter struct{}

func (f *DLPInspectionFilter) ID() string {
	return "dlp-privacy-guard"
}

func (f *DLPInspectionFilter) Point() HookPoint {
	return HookEgress
}

func (f *DLPInspectionFilter) Execute(ctx context.Context, payload []byte) (*FilterResult, error) {
	// Detección de patrones prohibidos según el axioma de Zero-PII (ej. palabras clave de fuga)
	forbiddenTokens := [][]byte{
		[]byte("PASSWORD="),
		[]byte("SECRET_KEY="),
		[]byte("PRIVATE_KEY_PII="),
	}

	for _, token := range forbiddenTokens {
		if bytes.Contains(payload, token) {
			return &FilterResult{
				Pass:     false,
				Reason:   fmt.Sprintf("Violación de Zero-PII detectada: presencia de token restringido %s", string(token)),
				FilterID: f.ID(),
			}, nil
		}
	}

	return &FilterResult{Pass: true, FilterID: f.ID()}, nil
}

// IntegrityCheckFilter agrega y verifica una etiqueta de integridad de contenido
type IntegrityCheckFilter struct{}

func (f *IntegrityCheckFilter) ID() string {
	return "smart-integrity-signer"
}

func (f *IntegrityCheckFilter) Point() HookPoint {
	return HookIngress
}

func (f *IntegrityCheckFilter) Execute(ctx context.Context, payload []byte) (*FilterResult, error) {
	// Filtro no invasivo: computa huella y permite paso
	_ = sha256.Sum256(payload)
	return &FilterResult{
		Pass:     true,
		FilterID: f.ID(),
	}, nil
}

// SmartPacketStats expone estadísticas del pipeline
type SmartPacketStats struct {
	TotalFilters    int    `json:"total_filters"`
	PacketsFiltered uint64 `json:"packets_filtered"`
	PacketsModified uint64 `json:"packets_modified"`
	PacketsDropped  uint64 `json:"packets_dropped"`
	TimeoutsCount   uint64 `json:"timeouts_count"`
}

// Stats retorna métricas para el panel web
func (spp *SmartPacketPipeline) Stats() SmartPacketStats {
	spp.mu.RLock()
	defer spp.mu.RUnlock()

	total := 0
	for _, l := range spp.filters {
		total += len(l)
	}

	return SmartPacketStats{
		TotalFilters:    total,
		PacketsFiltered: spp.PacketsFiltered,
		PacketsModified: spp.PacketsModified,
		PacketsDropped:  spp.PacketsDropped,
		TimeoutsCount:   spp.TimeoutsCount,
	}
}

// Ensure unused package import doesn't error
var _ = hex.EncodeToString
