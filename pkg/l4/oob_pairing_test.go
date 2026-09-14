package l4

import (
	"bytes"
	"testing"

	"ipvn7/pkg/l0"
)

func TestOOBPairingSASAndURChunks(t *testing.T) {
	idA, _ := l0.GenerateIdentity()
	idB, _ := l0.GenerateIdentity()

	pubA := idA.PublicKey
	pubB := idB.PublicKey

	// 1. Derivación de SAS independiente en ambos extremos
	sasA := DeriveSAS(pubA, pubB)
	sasB := DeriveSAS(pubB, pubA) // El orden de los argumentos no debe alterar el resultado

	if sasA.Digits != sasB.Digits {
		t.Fatalf("Los dígitos SAS no coinciden: %s vs %s", sasA.Digits, sasB.Digits)
	}

	if len(sasA.Digits) != 6 {
		t.Fatalf("Esperado código SAS de 6 dígitos, obtenido %d dígitos: %s", len(sasA.Digits), sasA.Digits)
	}

	if len(sasA.Emojis) != 4 {
		t.Fatalf("Esperado exactamente 4 emojis, obtenido %d", len(sasA.Emojis))
	}

	for i := 0; i < 4; i++ {
		if sasA.Emojis[i] != sasB.Emojis[i] {
			t.Fatalf("Discrepancia en emoji [%d]: %s vs %s", i, sasA.Emojis[i], sasB.Emojis[i])
		}
	}
	t.Logf("SAS verificado exitosamente: %s | %v", sasA.Digits, sasA.Emojis)

	// 2. Fragmentación y Reconstrucción UR para Códigos QR Animados
	profileData := []byte(`{"did":"` + idA.DID() + `","ipv6":"` + idA.IPv6().String() + `","locator":"192.168.1.198:7777"}`)
	chunks := GenerateURChunks(profileData, 30) // Fragmentos pequeños para forzar múltiples tramas

	if len(chunks) < 2 {
		t.Fatalf("Esperado múltiples tramas UR, obtenido %d", len(chunks))
	}

	// Reconstruir en orden desordenado para probar robustez
	shuffled := []string{chunks[1], chunks[0]}
	if len(chunks) > 2 {
		shuffled = append(shuffled, chunks[2:]...)
	}

	reconstructed, err := ReconstructURChunks(shuffled)
	if err != nil {
		t.Fatalf("Error reconstruyendo tramas UR: %v", err)
	}

	if !bytes.Equal(reconstructed, profileData) {
		t.Fatalf("Perfil reconstruido no coincide con el original:\n%s vs %s", string(reconstructed), string(profileData))
	}
}
