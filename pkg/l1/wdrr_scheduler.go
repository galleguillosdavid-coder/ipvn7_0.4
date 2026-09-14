package l1

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// WDRRPacket representa un paquete encolado en el planificador WDRR
type WDRRPacket struct {
	ID        uint64
	Class     TrafficClass
	Payload   []byte
	Size      int
	EnqueuedAt time.Time
}

// QueueMetrics estadísticas por cola de prioridad
type QueueMetrics struct {
	Class            TrafficClass `json:"class"`
	Weight           int          `json:"weight"`
	DeficitCounter   int          `json:"deficit_counter"`
	PacketsEnqueued  uint64       `json:"packets_enqueued"`
	PacketsDequeued  uint64       `json:"packets_dequeued"`
	PacketsDropped   uint64       `json:"packets_dropped"`
	CurrentQueueLen  int          `json:"current_queue_len"`
}

// WDRRScheduler implementa el planificador Weighted Deficit Round Robin (WDRR)
// para encolamiento justo sin inanición y prevención activa de Bufferbloat (1.md Sección 11)
type WDRRScheduler struct {
	mu            sync.Mutex
	quantum       int // Cuanto base de bytes por turno (ej. 1500 bytes)
	queues        map[TrafficClass][]*WDRRPacket
	weights       map[TrafficClass]int
	deficits      map[TrafficClass]int
	maxQueueLen   int
	classesOrder  []TrafficClass
	activeTurn    int
	packetCounter uint64

	// Contadores globales
	TotalEnqueued uint64
	TotalDequeued uint64
	TotalDropped  uint64
}

// NewWDRRScheduler inicializa el planificador con pesos relativos por clase
func NewWDRRScheduler(quantum, maxQueueLen int) *WDRRScheduler {
	if quantum <= 0 {
		quantum = 1500 // MTU estándar
	}
	if maxQueueLen <= 0 {
		maxQueueLen = 512
	}

	classes := []TrafficClass{ClassControl, ClassInteractive, ClassBulk}
	weights := map[TrafficClass]int{
		ClassControl:     10, // Control/Enrutamiento: máxima cuota
		ClassInteractive: 5,  // Streaming/Terminales: cuota media
		ClassBulk:        1,  // Transferencias masivas/DAG: cuota controlada
	}
	queues := make(map[TrafficClass][]*WDRRPacket)
	deficits := make(map[TrafficClass]int)

	for _, c := range classes {
		queues[c] = make([]*WDRRPacket, 0)
		deficits[c] = 0
	}

	return &WDRRScheduler{
		quantum:      quantum,
		queues:       queues,
		weights:      weights,
		deficits:     deficits,
		maxQueueLen:  maxQueueLen,
		classesOrder: classes,
		activeTurn:   0,
	}
}

// Enqueue ingresa un paquete a la cola correspondiente según su clase
func (s *WDRRScheduler) Enqueue(class TrafficClass, payload []byte) (*WDRRPacket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	q, ok := s.queues[class]
	if !ok {
		class = ClassBulk
		q = s.queues[ClassBulk]
	}

	// Prevención de Bufferbloat / Load Shedding (Sección 37)
	if len(q) >= s.maxQueueLen {
		atomic.AddUint64(&s.TotalDropped, 1)
		return nil, errors.New("cola saturada: paquete descartado preventivamente (anti-bufferbloat)")
	}

	id := atomic.AddUint64(&s.packetCounter, 1)
	pkt := &WDRRPacket{
		ID:         id,
		Class:      class,
		Payload:    payload,
		Size:       len(payload),
		EnqueuedAt: time.Now(),
	}

	s.queues[class] = append(s.queues[class], pkt)
	atomic.AddUint64(&s.TotalEnqueued, 1)
	return pkt, nil
}

// Dequeue extrae el siguiente paquete según la lógica determinista WDRR
func (s *WDRRScheduler) Dequeue() (*WDRRPacket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rounds := 0
	numClasses := len(s.classesOrder)

	for rounds < numClasses*2 {
		class := s.classesOrder[s.activeTurn]
		q := s.queues[class]

		if len(q) == 0 {
			// Si la cola está vacía, su déficit se reinicia a 0
			s.deficits[class] = 0
			s.activeTurn = (s.activeTurn + 1) % numClasses
			rounds++
			continue
		}

		// Incrementar déficit según peso de la clase
		s.deficits[class] += s.quantum * s.weights[class]

		// Despachar paquetes mientras el déficit cubra el tamaño del paquete
		for len(s.queues[class]) > 0 {
			head := s.queues[class][0]
			if s.deficits[class] >= head.Size {
				s.deficits[class] -= head.Size
				s.queues[class] = s.queues[class][1:]
				atomic.AddUint64(&s.TotalDequeued, 1)
				return head, true
			}
			break
		}

		// Pasar el turno a la siguiente clase
		s.activeTurn = (s.activeTurn + 1) % numClasses
		rounds++
	}

	return nil, false
}

// GetMetrics retorna el estado detallado de todas las colas de servicio
func (s *WDRRScheduler) GetMetrics() []QueueMetrics {
	s.mu.Lock()
	defer s.mu.Unlock()

	res := make([]QueueMetrics, 0, len(s.classesOrder))
	for _, c := range s.classesOrder {
		res = append(res, QueueMetrics{
			Class:           c,
			Weight:          s.weights[c],
			DeficitCounter:  s.deficits[c],
			CurrentQueueLen: len(s.queues[c]),
		})
	}
	return res
}
