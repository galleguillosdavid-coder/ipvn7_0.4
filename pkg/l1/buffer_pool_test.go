package l1

import (
	"testing"
)

func TestBufferPoolAcquireAndRecycle(t *testing.T) {
	bp := NewBufferPool()

	// 1. Probar búfer Small
	bSmall := bp.Acquire(32)
	if bSmall.Capacity() < PoolSmallSize {
		t.Fatalf("Esperado capacidad >= %d, obtenido %d", PoolSmallSize, bSmall.Capacity())
	}
	if bSmall.RefCount() != 1 {
		t.Fatalf("Esperado refCount=1, obtenido %d", bSmall.RefCount())
	}

	// Retain y Release
	bSmall.Retain()
	if bSmall.RefCount() != 2 {
		t.Fatalf("Esperado refCount=2 tras Retain(), obtenido %d", bSmall.RefCount())
	}
	bSmall.Release()
	if bSmall.RefCount() != 1 {
		t.Fatalf("Esperado refCount=1 tras primera Release(), obtenido %d", bSmall.RefCount())
	}
	bSmall.Release() // Debe reciclarse

	stats := bp.Stats()
	if stats.RecycledCount != 1 {
		t.Fatalf("Esperado RecycledCount=1, obtenido %d", stats.RecycledCount)
	}

	// 2. Probar búfer Standard
	bStd := bp.Acquire(1280)
	if bStd.Capacity() < PoolStandardSize {
		t.Fatalf("Esperado capacidad >= %d, obtenido %d", PoolStandardSize, bStd.Capacity())
	}
	bStd.Release()

	// 3. Probar búfer Jumbo
	bJumbo := bp.Acquire(20000)
	if bJumbo.Capacity() < PoolJumboSize {
		t.Fatalf("Esperado capacidad >= %d, obtenido %d", PoolJumboSize, bJumbo.Capacity())
	}
	bJumbo.Release()

	finalStats := bp.Stats()
	if finalStats.RecycledCount != 3 {
		t.Fatalf("Esperado 3 búferes reciclados, obtenido %d", finalStats.RecycledCount)
	}
}
