package l1

import (
	"testing"
)

func TestMultipathScheduler(t *testing.T) {
	ms := NewMultipathScheduler()

	// 1. Probar Lowest Latency (wlan tiene 2.5 ms, eth tiene 4.8 ms)
	paths, err := ms.SelectPaths(StrategyLowestLatency)
	if err != nil {
		t.Fatalf("Error seleccionando camino: %v", err)
	}
	if len(paths) != 1 || paths[0].Name != "primary-wlan" {
		t.Fatalf("Esperado 'primary-wlan' (menor latencia), obtenido %s", paths[0].Name)
	}

	// 2. Simular degradación de wlan a 10.0 ms
	_ = ms.UpdateInterfaceMetrics("primary-wlan", 10.0, 0.05, true)
	paths, _ = ms.SelectPaths(StrategyLowestLatency)
	if paths[0].Name != "secondary-eth" {
		t.Fatalf("Esperado conmutación a 'secondary-eth' tras aumento de latencia, obtenido %s", paths[0].Name)
	}

	// 3. Probar Critical Duplication (debe retornar ambas interfaces activas)
	pathsDup, err := ms.SelectPaths(StrategyCriticalDuplication)
	if err != nil || len(pathsDup) != 2 {
		t.Fatalf("Esperado 2 interfaces duplicadas concurrentemente, obtenido %d", len(pathsDup))
	}

	stats := ms.Stats()
	if stats.PacketsRouted != 3 || stats.PacketsDuplicated != 1 {
		t.Fatalf("Estadísticas inesperadas: %+v", stats)
	}
}
