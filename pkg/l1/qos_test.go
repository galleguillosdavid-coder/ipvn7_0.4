package l1

import (
	"testing"
)

func TestQoSTokenBucketAndPoW(t *testing.T) {
	qm := NewQoSManager()
	qm.burstLimit = 1000.0 // Límite bajo para forzar saturación en test
	qm.defaultRate = 100.0 // Relleno lento

	did := "did:ipvn7:testpeer1234567890abcdef"

	// 1. Tráfico de control siempre debe pasar
	allowed, ch := qm.EvaluatePacket(did, ClassControl, 2000)
	if !allowed || ch != nil {
		t.Fatalf("Esperado que ClassControl pase sin restricciones")
	}

	// 2. Consumir la capacidad de ráfaga
	allowed, _ = qm.EvaluatePacket(did, ClassInteractive, 800)
	if !allowed {
		t.Fatalf("Esperado que el primer paquete de 800 bytes pase")
	}

	// 3. Exceder capacidad: debe ser estrangulado y emitir desafío PoW
	allowed, ch = qm.EvaluatePacket(did, ClassBulk, 500)
	if allowed {
		t.Fatalf("Esperado que el paquete excedente sea estrangulado")
	}
	if ch == nil || ch.Difficulty != 16 {
		t.Fatalf("Esperado desafío PoW emitido con dificultad 16, obtenido %+v", ch)
	}

	// 4. Resolver el desafío PoW
	nonce, err := SolvePoW(ch.Challenge, ch.Difficulty)
	if err != nil {
		t.Fatalf("Error al resolver PoW: %v", err)
	}

	// 5. Verificar solución
	valid := qm.VerifyAndCreditPoW(ch.Challenge, nonce)
	if !valid {
		t.Fatalf("Verificación de PoW falló para nonce %d", nonce)
	}

	// 6. Tras resolver PoW, los tokens deben haberse repuesto y permitir tráfico
	allowed, _ = qm.EvaluatePacket(did, ClassBulk, 500)
	if !allowed {
		t.Fatalf("Esperado que el tráfico pase tras resolver PoW exitosamente")
	}

	stats := qm.Stats()
	if stats.PoWsIssued != 1 || stats.PoWsSolved != 1 {
		t.Fatalf("Estadísticas de PoW incorrectas: %+v", stats)
	}
}
