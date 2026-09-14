package l1

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
)

// MultipathStrategy define la política de selección de interfaces heterogéneas (Dimensión 8)
type MultipathStrategy int

const (
	StrategyLowestLatency       MultipathStrategy = 0 // Envía exclusivamente por el enlace con menor RTT
	StrategyBalanced            MultipathStrategy = 1 // Distribución ponderada de carga entre enlaces activos
	StrategyCriticalDuplication MultipathStrategy = 2 // Transmisión redundante concurrente (cero pérdida y latencia ultra-baja)
)

func (s MultipathStrategy) String() string {
	switch s {
	case StrategyLowestLatency:
		return "LOWEST_LATENCY"
	case StrategyBalanced:
		return "BALANCED"
	case StrategyCriticalDuplication:
		return "CRITICAL_DUPLICATION"
	default:
		return "UNKNOWN"
	}
}

// PhysicalInterface representa un enlace físico de red disponible en el host
type PhysicalInterface struct {
	Name      string  `json:"name"`       // ej. "eth0", "wlan0", "cellular5g"
	Type      string  `json:"type"`       // "ETHERNET", "WIFI", "CELLULAR", "SAT"
	LocalAddr string  `json:"local_addr"` // Dirección IP local en esa interfaz
	LatencyMs float64 `json:"latency_ms"` // RTT medido hacia la malla
	LossRate  float64 `json:"loss_rate"`  // Tasa de pérdida de paquetes (0.0 a 1.0)
	Weight    int     `json:"weight"`     // Ponderación de capacidad relativa
	Active    bool    `json:"active"`     // Estado operacional del enlace
}

// MultipathScheduler orquesta la selección y duplicación de caminos físicos
type MultipathScheduler struct {
	mu                 sync.RWMutex
	interfaces         map[string]*PhysicalInterface
	rrCounter          uint64
	defaultStrategy    MultipathStrategy

	// Métricas de observabilidad
	PacketsRouted      uint64
	PacketsDuplicated  uint64
	FailoverEvents     uint64
}

// NewMultipathScheduler inicializa el coordinador multipath
func NewMultipathScheduler() *MultipathScheduler {
	ms := &MultipathScheduler{
		interfaces:      make(map[string]*PhysicalInterface),
		defaultStrategy: StrategyLowestLatency,
	}

	// Registrar interfaces por defecto según entorno
	ms.RegisterInterface(&PhysicalInterface{
		Name:      "primary-wlan",
		Type:      "WIFI",
		LocalAddr: "192.168.1.198",
		LatencyMs: 2.5,
		LossRate:  0.001,
		Weight:    10,
		Active:    true,
	})

	ms.RegisterInterface(&PhysicalInterface{
		Name:      "secondary-eth",
		Type:      "ETHERNET",
		LocalAddr: "10.0.0.2",
		LatencyMs: 4.8,
		LossRate:  0.000,
		Weight:    20,
		Active:    true,
	})

	return ms
}

// RegisterInterface añade o actualiza una interfaz en el planificador
func (ms *MultipathScheduler) RegisterInterface(iface *PhysicalInterface) {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.interfaces[iface.Name] = iface
}

// UpdateInterfaceMetrics actualiza las condiciones observadas del enlace
func (ms *MultipathScheduler) UpdateInterfaceMetrics(name string, latencyMs, lossRate float64, active bool) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	iface, ok := ms.interfaces[name]
	if !ok {
		return fmt.Errorf("interfaz %s no encontrada", name)
	}

	if iface.Active != active {
		ms.FailoverEvents++
	}

	iface.LatencyMs = latencyMs
	iface.LossRate = lossRate
	iface.Active = active
	return nil
}

// SelectPaths determina por cuáles interfaces debe emitirse el datagrama según la estrategia
func (ms *MultipathScheduler) SelectPaths(strategy MultipathStrategy) ([]*PhysicalInterface, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	activeList := make([]*PhysicalInterface, 0, len(ms.interfaces))
	for _, iface := range ms.interfaces {
		if iface.Active {
			activeList = append(activeList, iface)
		}
	}

	if len(activeList) == 0 {
		return nil, errors.New("no hay interfaces de red físicas activas disponibles")
	}

	atomic.AddUint64(&ms.PacketsRouted, 1)

	switch strategy {
	case StrategyLowestLatency:
		// Encontrar la interfaz con menor LatencyMs
		best := activeList[0]
		for _, iface := range activeList[1:] {
			if iface.LatencyMs < best.LatencyMs {
				best = iface
			}
		}
		return []*PhysicalInterface{best}, nil

	case StrategyCriticalDuplication:
		// Emitir copia concurrente por todas las interfaces activas (mínimo 2 si están disponibles)
		atomic.AddUint64(&ms.PacketsDuplicated, uint64(len(activeList)-1))
		return activeList, nil

	case StrategyBalanced:
		// Round-Robin balanceado
		idx := atomic.AddUint64(&ms.rrCounter, 1) % uint64(len(activeList))
		return []*PhysicalInterface{activeList[idx]}, nil

	default:
		return []*PhysicalInterface{activeList[0]}, nil
	}
}

// GetAllInterfaces retorna una copia de todas las interfaces registradas
func (ms *MultipathScheduler) GetAllInterfaces() []*PhysicalInterface {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	res := make([]*PhysicalInterface, 0, len(ms.interfaces))
	for _, iface := range ms.interfaces {
		res = append(res, iface)
	}
	return res
}

// MultipathStats resume las métricas para el panel web
type MultipathStats struct {
	TotalInterfaces   int    `json:"total_interfaces"`
	ActiveInterfaces  int    `json:"active_interfaces"`
	PacketsRouted     uint64 `json:"packets_routed"`
	PacketsDuplicated uint64 `json:"packets_duplicated"`
	FailoverEvents    uint64 `json:"failover_events"`
}

// Stats genera la instantánea de observabilidad
func (ms *MultipathScheduler) Stats() MultipathStats {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	activeCount := 0
	for _, iface := range ms.interfaces {
		if iface.Active {
			activeCount++
		}
	}

	return MultipathStats{
		TotalInterfaces:   len(ms.interfaces),
		ActiveInterfaces:  activeCount,
		PacketsRouted:     atomic.LoadUint64(&ms.PacketsRouted),
		PacketsDuplicated: atomic.LoadUint64(&ms.PacketsDuplicated),
		FailoverEvents:    ms.FailoverEvents,
	}
}
