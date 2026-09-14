package l1_test

import (
	"testing"

	"ipvn7/pkg/l1"
)

func TestWDRRSchedulerFairQueuingAndPrioritization(t *testing.T) {
	// Quantum 1000 bytes, maxQueueLen 10
	sched := l1.NewWDRRScheduler(1000, 10)

	// 1. Encolar paquetes en diferentes clases
	// Bulk: 2 paquetes de 800 bytes
	_, err := sched.Enqueue(l1.ClassBulk, make([]byte, 800))
	if err != nil {
		t.Fatalf("Error encolando Bulk 1: %v", err)
	}
	_, _ = sched.Enqueue(l1.ClassBulk, make([]byte, 800))

	// Control: 1 paquete de 200 bytes
	_, err = sched.Enqueue(l1.ClassControl, make([]byte, 200))
	if err != nil {
		t.Fatalf("Error encolando Control: %v", err)
	}

	// Interactive: 1 paquete de 500 bytes
	_, err = sched.Enqueue(l1.ClassInteractive, make([]byte, 500))
	if err != nil {
		t.Fatalf("Error encolando Interactive: %v", err)
	}

	// 2. Extraer paquetes y verificar que Control se despacha primero (peso 10 vs 5 vs 1)
	pkt1, ok := sched.Dequeue()
	if !ok || pkt1.Class != l1.ClassControl {
		t.Fatalf("Se esperaba que ClassControl fuera despachado primero, obtenido: %v", pkt1.Class)
	}

	pkt2, ok := sched.Dequeue()
	if !ok || pkt2.Class != l1.ClassInteractive {
		t.Fatalf("Se esperaba que ClassInteractive fuera despachado segundo, obtenido: %v", pkt2.Class)
	}

	pkt3, ok := sched.Dequeue()
	if !ok || pkt3.Class != l1.ClassBulk {
		t.Fatalf("Se esperaba que ClassBulk fuera despachado tercero, obtenido: %v", pkt3.Class)
	}

	// 3. Probar Load Shedding / Prevención de Bufferbloat
	schedSmall := l1.NewWDRRScheduler(500, 2)
	_, _ = schedSmall.Enqueue(l1.ClassBulk, make([]byte, 100))
	_, _ = schedSmall.Enqueue(l1.ClassBulk, make([]byte, 100))
	// Tercer paquete debe ser descartado preventivamente
	_, err = schedSmall.Enqueue(l1.ClassBulk, make([]byte, 100))
	if err == nil {
		t.Errorf("Se esperaba descarte preventivo por sobrepasar maxQueueLen")
	}
}
