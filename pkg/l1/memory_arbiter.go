// Package l1 implementa el Árbitro Global de Memoria (GlobalMemoryArbiter)
// rescatado de ip7uin_MVP_Spec_v1.1.docx (Sección 4: "GlobalMemoryArbiter").
// Principio: Protección estricta contra ataques de denegación de servicio por agotamiento
// de memoria (DoS por OOM), particionando la RAM por clases de presupuesto y rechazando
// preventivamente excesos de cuota sin comprometer la estabilidad del enrutador de malla.
package l1

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// BudgetClass define la categoría de consumo de memoria según el spec ip7uin
type BudgetClass string

const (
	BudgetReplay   BudgetClass = "replay"   // 20% (Ventanas y cachés anti-repetición)
	BudgetQoS      BudgetClass = "qos"      // 30% (Colas de prioridad P0-P3 y Token Buckets)
	BudgetTrust    BudgetClass = "trust"    // 20% (Tablas de reputación y Web of Trust)
	BudgetBindings BudgetClass = "bindings" // 20% (Pasaportes UIN y BindingRecords)
	BudgetOther    BudgetClass = "other"    // 10% (Búferes efímeros de control)
)

// ClassQuota contiene el estado operativo de una clase de presupuesto
type ClassQuota struct {
	Percentage   float64 `json:"percentage"`
	LimitBytes   uint64  `json:"limit_bytes"`
	Allocated    uint64  `json:"allocated_bytes"`
	DroppedCount uint64  `json:"dropped_allocations"`
}

// MemoryArbiterStats contiene la telemetría viva del árbitro de memoria
type MemoryArbiterStats struct {
	TotalLimitBytes uint64                 `json:"total_limit_bytes"`
	TotalAllocated  uint64                 `json:"total_allocated_bytes"`
	UsagePercentage float64                `json:"usage_percentage"`
	TotalRejections uint64                 `json:"total_rejections"`
	Classes         map[BudgetClass]ClassQuota `json:"classes"`
}

// GlobalMemoryArbiter orquesta las cuotas de RAM para todos los subsistemas del nodo
type GlobalMemoryArbiter struct {
	mu         sync.RWMutex
	totalLimit uint64
	totalAlloc uint64
	rejections uint64
	quotas     map[BudgetClass]*ClassQuota
}

// DefaultTotalMemoryLimit define 64MB de límite por defecto para overlays ligeros
const DefaultTotalMemoryLimit = 64 * 1024 * 1024 // 64 MB

// NewGlobalMemoryArbiter inicializa el árbitro con las particiones canónicas de ip7uin
func NewGlobalMemoryArbiter(totalBytes uint64) *GlobalMemoryArbiter {
	if totalBytes == 0 {
		totalBytes = DefaultTotalMemoryLimit
	}

	quotas := make(map[BudgetClass]*ClassQuota)
	quotas[BudgetReplay] = &ClassQuota{Percentage: 0.20, LimitBytes: uint64(float64(totalBytes) * 0.20)}
	quotas[BudgetQoS] = &ClassQuota{Percentage: 0.30, LimitBytes: uint64(float64(totalBytes) * 0.30)}
	quotas[BudgetTrust] = &ClassQuota{Percentage: 0.20, LimitBytes: uint64(float64(totalBytes) * 0.20)}
	quotas[BudgetBindings] = &ClassQuota{Percentage: 0.20, LimitBytes: uint64(float64(totalBytes) * 0.20)}
	quotas[BudgetOther] = &ClassQuota{Percentage: 0.10, LimitBytes: uint64(float64(totalBytes) * 0.10)}

	return &GlobalMemoryArbiter{
		totalLimit: totalBytes,
		quotas:     quotas,
	}
}

// Reserve solicita atómicamente la reserva de bytes para una clase específica.
// Retorna true si la asignación está dentro de la cuota, o false si excede el límite.
func (a *GlobalMemoryArbiter) Reserve(class BudgetClass, bytes uint64) bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	q, exists := a.quotas[class]
	if !exists {
		// Asignar por defecto a BudgetOther si no existe
		q = a.quotas[BudgetOther]
	}

	if q.Allocated+bytes > q.LimitBytes {
		q.DroppedCount++
		atomic.AddUint64(&a.rejections, 1)
		return false
	}

	if a.totalAlloc+bytes > a.totalLimit {
		q.DroppedCount++
		atomic.AddUint64(&a.rejections, 1)
		return false
	}

	q.Allocated += bytes
	a.totalAlloc += bytes
	return true
}

// Release libera atómicamente bytes previamente reservados
func (a *GlobalMemoryArbiter) Release(class BudgetClass, bytes uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	q, exists := a.quotas[class]
	if !exists {
		q = a.quotas[BudgetOther]
	}

	if bytes > q.Allocated {
		bytes = q.Allocated
	}
	q.Allocated -= bytes

	if bytes > a.totalAlloc {
		a.totalAlloc = 0
	} else {
		a.totalAlloc -= bytes
	}
}

// GetStats retorna las métricas completas del árbitro
func (a *GlobalMemoryArbiter) GetStats() MemoryArbiterStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	cls := make(map[BudgetClass]ClassQuota)
	for k, v := range a.quotas {
		cls[k] = *v
	}

	pct := 0.0
	if a.totalLimit > 0 {
		pct = (float64(a.totalAlloc) / float64(a.totalLimit)) * 100.0
	}

	return MemoryArbiterStats{
		TotalLimitBytes: a.totalLimit,
		TotalAllocated:  a.totalAlloc,
		UsagePercentage: float64(int(pct*100)) / 100.0,
		TotalRejections: atomic.LoadUint64(&a.rejections),
		Classes:         cls,
	}
}

// FormatSummary produce una síntesis en texto legible para logs o CLI
func (a *GlobalMemoryArbiter) FormatSummary() string {
	s := a.GetStats()
	return fmt.Sprintf("RAM Malla: %.2f%% usada (%d / %d KB) · Rechazos preventivos DoS: %d",
		s.UsagePercentage, s.TotalAllocated/1024, s.TotalLimitBytes/1024, s.TotalRejections)
}
