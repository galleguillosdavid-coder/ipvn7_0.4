package l1

import (
	"errors"
	"sync"
	"sync/atomic"
)

// Constantes de tamaño para los 3 niveles del Buffer Pool (Dimensión 12)
const (
	PoolSmallSize    = 64    // Cabeceras, tokens y datagramas de control
	PoolStandardSize = 1500  // MTU 1280 determinista + sobrecarga de transporte
	PoolJumboSize    = 65536 // Tramas gigantes de streaming y agregación
)

// PacketBuffer envuelve un búfer preasignado con conteo atómico de referencias (Zero-Copy)
type PacketBuffer struct {
	data     []byte
	length   int
	refCount int32
	poolType int // 0: Small, 1: Standard, 2: Jumbo
	pool     *BufferPool
}

// Data retorna el slice de bytes activo del búfer
func (b *PacketBuffer) Data() []byte {
	return b.data[:b.length]
}

// RawSlice retorna el slice completo con capacidad asignada
func (b *PacketBuffer) RawSlice() []byte {
	return b.data
}

// SetLength actualiza la longitud de datos válidos
func (b *PacketBuffer) SetLength(n int) error {
	if n > len(b.data) {
		return errors.New("longitud solicitada excede la capacidad del búfer preasignado")
	}
	b.length = n
	return nil
}

// Length retorna la longitud actual de datos válidos
func (b *PacketBuffer) Length() int {
	return b.length
}

// Capacidad retorna la capacidad máxima del búfer
func (b *PacketBuffer) Capacity() int {
	return len(b.data)
}

// Retain incrementa atómicamente el contador de referencias
func (b *PacketBuffer) Retain() {
	atomic.AddInt32(&b.refCount, 1)
}

// Release decrementa atómicamente el contador y devuelve el búfer al pool si llega a 0
func (b *PacketBuffer) Release() {
	if atomic.AddInt32(&b.refCount, -1) == 0 {
		b.pool.recycle(b)
	}
}

// RefCount retorna el número actual de referencias
func (b *PacketBuffer) RefCount() int32 {
	return atomic.LoadInt32(&b.refCount)
}

// BufferPool gestiona el reciclaje de memoria en 3 niveles de tamaño
type BufferPool struct {
	smallPool    sync.Pool
	standardPool sync.Pool
	jumboPool    sync.Pool

	// Métricas atómicas para observabilidad L2
	AllocationsSmall    uint64
	AllocationsStandard uint64
	AllocationsJumbo    uint64
	RecycledCount       uint64
}

// NewBufferPool inicializa el gestor de búferes de memoria
func NewBufferPool() *BufferPool {
	bp := &BufferPool{}

	bp.smallPool = sync.Pool{
		New: func() interface{} {
			atomic.AddUint64(&bp.AllocationsSmall, 1)
			return &PacketBuffer{
				data:     make([]byte, PoolSmallSize),
				poolType: 0,
				pool:     bp,
			}
		},
	}

	bp.standardPool = sync.Pool{
		New: func() interface{} {
			atomic.AddUint64(&bp.AllocationsStandard, 1)
			return &PacketBuffer{
				data:     make([]byte, PoolStandardSize),
				poolType: 1,
				pool:     bp,
			}
		},
	}

	bp.jumboPool = sync.Pool{
		New: func() interface{} {
			atomic.AddUint64(&bp.AllocationsJumbo, 1)
			return &PacketBuffer{
				data:     make([]byte, PoolJumboSize),
				poolType: 2,
				pool:     bp,
			}
		},
	}

	return bp
}

// Acquire solicita un búfer preasignado del nivel adecuado según el tamaño requerido
func (bp *BufferPool) Acquire(minSize int) *PacketBuffer {
	var buf *PacketBuffer

	if minSize <= PoolSmallSize {
		buf = bp.smallPool.Get().(*PacketBuffer)
	} else if minSize <= PoolStandardSize {
		buf = bp.standardPool.Get().(*PacketBuffer)
	} else {
		buf = bp.jumboPool.Get().(*PacketBuffer)
	}

	buf.length = minSize
	atomic.StoreInt32(&buf.refCount, 1)
	return buf
}

// recycle devuelve el búfer a su sync.Pool correspondiente
func (bp *BufferPool) recycle(b *PacketBuffer) {
	atomic.AddUint64(&bp.RecycledCount, 1)
	b.length = 0

	switch b.poolType {
	case 0:
		bp.smallPool.Put(b)
	case 1:
		bp.standardPool.Put(b)
	case 2:
		bp.jumboPool.Put(b)
	}
}

// PoolStats exporta estadísticas de uso de memoria zero-copy
type PoolStats struct {
	AllocationsSmall    uint64 `json:"allocations_small"`
	AllocationsStandard uint64 `json:"allocations_standard"`
	AllocationsJumbo    uint64 `json:"allocations_jumbo"`
	RecycledCount       uint64 `json:"recycled_count"`
}

// Stats genera una instantánea de métricas del pool
func (bp *BufferPool) Stats() PoolStats {
	return PoolStats{
		AllocationsSmall:    atomic.LoadUint64(&bp.AllocationsSmall),
		AllocationsStandard: atomic.LoadUint64(&bp.AllocationsStandard),
		AllocationsJumbo:    atomic.LoadUint64(&bp.AllocationsJumbo),
		RecycledCount:       atomic.LoadUint64(&bp.RecycledCount),
	}
}
