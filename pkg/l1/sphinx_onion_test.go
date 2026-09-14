package l1

import (
	"bytes"
	"testing"
)

func TestSphinxOnionCircuitAndPeeling(t *testing.T) {
	// 1. Crear 3 pares híbridos para el circuito (Guard, Middle, Exit)
	guardKeys, err := GenerateHybridKeyPair("did:ipvn7:guard-node")
	if err != nil {
		t.Fatalf("Error creando Guard keys: %v", err)
	}
	middleKeys, err := GenerateHybridKeyPair("did:ipvn7:middle-node")
	if err != nil {
		t.Fatalf("Error creando Middle keys: %v", err)
	}
	exitKeys, err := GenerateHybridKeyPair("did:ipvn7:exit-node")
	if err != nil {
		t.Fatalf("Error creando Exit keys: %v", err)
	}

	circuit := &SphinxCircuit{
		CircuitID: "circuit-xyz-777",
		Guard: &SphinxHopNode{
			DID:          guardKeys.DID,
			Address:      "192.168.1.100:7777",
			HybridKeys:   guardKeys,
			X25519PubHex: guardKeys.X25519PubHex,
			MLKEMPubHex:  guardKeys.MLKEMPubHex,
		},
		Middle: &SphinxHopNode{
			DID:          middleKeys.DID,
			Address:      "192.168.1.101:7777",
			HybridKeys:   middleKeys,
			X25519PubHex: middleKeys.X25519PubHex,
			MLKEMPubHex:  middleKeys.MLKEMPubHex,
		},
		Exit: &SphinxHopNode{
			DID:          exitKeys.DID,
			Address:      "192.168.1.102:7777",
			HybridKeys:   exitKeys,
			X25519PubHex: exitKeys.X25519PubHex,
			MLKEMPubHex:  exitKeys.MLKEMPubHex,
		},
	}

	senderKeys, _ := GenerateHybridKeyPair("did:ipvn7:sender")
	senderRouter := NewSphinxRouter(senderKeys)

	// 2. Construir paquete Sphinx cebolla
	originalPayload := []byte("Mensaje confidencial de grado soberano a través de 3 saltos.")
	destService := "echo.service"

	packet, err := senderRouter.BuildPacket(circuit, originalPayload, destService)
	if err != nil {
		t.Fatalf("BuildPacket failed: %v", err)
	}

	// 3. Validar serialización determinista estricta a 1280 bytes
	wire := packet.Serialize()
	if len(wire) != SphinxPacketSize {
		t.Fatalf("Tamaño de wire inválido: %d bytes (esperado %d)", len(wire), SphinxPacketSize)
	}

	deserialized, err := DeserializeSphinxPacket(wire)
	if err != nil {
		t.Fatalf("DeserializeSphinxPacket failed: %v", err)
	}

	// 4. Salto 1: Guard Node
	guardRouter := NewSphinxRouter(guardKeys)
	peel1, err := guardRouter.PeelLayer(deserialized)
	if err != nil {
		t.Fatalf("Guard PeelLayer failed: %v", err)
	}
	if peel1.Action != SphinxActionForward {
		t.Fatalf("Acción de Guard inesperada: %s (esperado FORWARD)", peel1.Action)
	}
	if peel1.NextPacket == nil {
		t.Fatalf("Guard no generó el siguiente paquete cebolla")
	}
	if len(peel1.NextPacket.Serialize()) != SphinxPacketSize {
		t.Fatalf("Paquete tras Guard no preserva 1280 bytes: %d", len(peel1.NextPacket.Serialize()))
	}

	// 5. Salto 2: Middle Node
	middleRouter := NewSphinxRouter(middleKeys)
	peel2, err := middleRouter.PeelLayer(peel1.NextPacket)
	if err != nil {
		t.Fatalf("Middle PeelLayer failed: %v", err)
	}
	if peel2.Action != SphinxActionForward {
		t.Fatalf("Acción de Middle inesperada: %s (esperado FORWARD)", peel2.Action)
	}
	if peel2.NextPacket == nil {
		t.Fatalf("Middle no generó el siguiente paquete cebolla")
	}
	if len(peel2.NextPacket.Serialize()) != SphinxPacketSize {
		t.Fatalf("Paquete tras Middle no preserva 1280 bytes: %d", len(peel2.NextPacket.Serialize()))
	}

	// 6. Salto 3: Exit Node
	exitRouter := NewSphinxRouter(exitKeys)
	peel3, err := exitRouter.PeelLayer(peel2.NextPacket)
	if err != nil {
		t.Fatalf("Exit PeelLayer failed: %v", err)
	}
	if peel3.Action != SphinxActionDeliver {
		t.Fatalf("Acción de Exit inesperada: %s (esperado DELIVER)", peel3.Action)
	}
	if !peel3.IsExit {
		t.Fatalf("IsExit debió ser true en Exit Node")
	}

	// 7. Validar recuperación íntegra del payload original
	if !bytes.Equal(peel3.RawPayload, originalPayload) {
		t.Fatalf("Payload recuperado no coincide! Esperado: %s, Obtenido: %s", string(originalPayload), string(peel3.RawPayload))
	}

	// 8. Validar protección anti-repetición O(1)
	_, errReplay := guardRouter.PeelLayer(deserialized)
	if errReplay == nil {
		t.Fatalf("Guard debió rechazar el paquete repetido!")
	}
}
