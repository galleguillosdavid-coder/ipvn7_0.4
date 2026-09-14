package l0_test

import (
	"bytes"
	"strings"
	"testing"

	"ipvn7/pkg/l0"
)

func TestIdentityGenerationAndDID(t *testing.T) {
	id, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("GenerateIdentity falló: %v", err)
	}

	did := id.DID()
	if !strings.HasPrefix(did, "did:ipvn7:") {
		t.Errorf("DID no tiene prefijo esperado: %s", did)
	}

	pub, err := l0.PublicKeyFromDID(did)
	if err != nil {
		t.Fatalf("PublicKeyFromDID falló: %v", err)
	}
	if !bytes.Equal(pub, id.PublicKey) {
		t.Errorf("Clave pública recuperada no coincide con original")
	}

	ipv6 := id.IPv6()
	if ipv6[0] != 0xfd || ipv6[1] != 0x07 {
		t.Errorf("IPv6 no tiene prefijo fd07::/64: %v", ipv6)
	}

	ipv4 := id.IPv4()
	if ipv4[0] != 10 || ipv4[1] != 7 {
		t.Errorf("IPv4 no tiene prefijo 10.7.x.y: %v", ipv4)
	}
}

func TestWireCBORPacketSerialization(t *testing.T) {
	id1, _ := l0.GenerateIdentity()
	id2, _ := l0.GenerateIdentity()

	nonce, _ := l0.GenerateNonce()
	payload := []byte("Hola Mundo ipvn7 - Red Soberana")

	pkt := l0.NewPacket(l0.MsgTypeData, id1.DID(), id2.DID(), 100, nonce, payload)
	if err := pkt.SignPacket(id1); err != nil {
		t.Fatalf("SignPacket falló: %v", err)
	}

	encoded, err := pkt.Encode()
	if err != nil {
		t.Fatalf("Encode falló: %v", err)
	}

	if len(encoded) > l0.MaxPacketSize {
		t.Errorf("Paquete excede MaxPacketSize: %d bytes", len(encoded))
	}

	decoded, err := l0.DecodePacket(encoded)
	if err != nil {
		t.Fatalf("DecodePacket falló: %v", err)
	}

	if decoded.SourceDID != id1.DID() || decoded.DestDID != id2.DID() {
		t.Errorf("DIDs no coinciden en paquete decodificado")
	}
	if !bytes.Equal(decoded.Payload, payload) {
		t.Errorf("Payload alterado tras decodificación")
	}

	valid, err := decoded.VerifyPacketSignature()
	if err != nil || !valid {
		t.Errorf("Verificación de firma digital falló: valid=%v, err=%v", valid, err)
	}
}

func TestNoiseHandshakeAndAEAD(t *testing.T) {
	aliceID, _ := l0.GenerateIdentity()
	bobID, _ := l0.GenerateIdentity()

	aliceHS, err := l0.NewNoiseHandshake(aliceID)
	if err != nil {
		t.Fatalf("Alice NewNoiseHandshake falló: %v", err)
	}
	bobHS, err := l0.NewNoiseHandshake(bobID)
	if err != nil {
		t.Fatalf("Bob NewNoiseHandshake falló: %v", err)
	}

	// Step 1: Alice -> Bob
	msg1, err := aliceHS.Step1Initiate()
	if err != nil {
		t.Fatalf("Step1Initiate falló: %v", err)
	}

	// Step 2: Bob procesa msg1 y responde
	msg2, bobKeys, err := bobHS.Step2Respond(msg1)
	if err != nil {
		t.Fatalf("Step2Respond falló: %v", err)
	}

	// Step 3: Alice finaliza con msg2
	aliceKeys, err := aliceHS.Step3Finalize(msg2)
	if err != nil {
		t.Fatalf("Step3Finalize falló: %v", err)
	}

	// Comprobar coincidencia simétrica de claves
	if aliceKeys.TxKey != bobKeys.RxKey {
		t.Errorf("Alice TxKey != Bob RxKey")
	}
	if aliceKeys.RxKey != bobKeys.TxKey {
		t.Errorf("Alice RxKey != Bob TxKey")
	}

	// Probar cifrado y descifrado AEAD
	nonce, _ := l0.GenerateNonce()
	plaintext := []byte("Datagrama confidencial E2EE")
	ad := []byte("metadata-no-cifrada")

	ciphertext, err := l0.EncryptPayload(aliceKeys.TxKey[:], nonce, plaintext, ad)
	if err != nil {
		t.Fatalf("EncryptPayload falló: %v", err)
	}

	decrypted, err := l0.DecryptPayload(bobKeys.RxKey[:], nonce, ciphertext, ad)
	if err != nil {
		t.Fatalf("DecryptPayload falló: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Texto descifrado no coincide con texto en claro")
	}
}

func TestAntiReplayFilter(t *testing.T) {
	filter := l0.NewAntiReplayFilter()

	if !filter.ValidateAndUpdate(1) {
		t.Errorf("Secuencia 1 debió ser aceptada")
	}
	if !filter.ValidateAndUpdate(2) {
		t.Errorf("Secuencia 2 debió ser aceptada")
	}
	if filter.ValidateAndUpdate(1) {
		t.Errorf("Secuencia 1 repetida debió ser rechazada")
	}
	if !filter.ValidateAndUpdate(50) {
		t.Errorf("Secuencia 50 debió ser aceptada")
	}
	if !filter.ValidateAndUpdate(49) {
		t.Errorf("Secuencia 49 (dentro de ventana) debió ser aceptada")
	}
	if filter.ValidateAndUpdate(49) {
		t.Errorf("Secuencia 49 repetida debió ser rechazada")
	}
}

func TestStatelessCookieGenerator(t *testing.T) {
	cg := l0.NewStatelessCookieGenerator()
	addr := "192.168.1.50:7777"
	ephPub := []byte("32bytes-ephemeral-key-alice-xxxx")

	cookie := cg.GenerateCookie(addr, ephPub)
	if len(cookie) != 16 {
		t.Fatalf("Tamaño de cookie incorrecto: %d bytes (esperado 16)", len(cookie))
	}

	if !cg.ValidateCookie(cookie, addr, ephPub) {
		t.Errorf("Cookie legítima fue rechazada")
	}

	// Dirección diferente
	if cg.ValidateCookie(cookie, "192.168.1.99:7777", ephPub) {
		t.Errorf("Cookie validada para dirección incorrecta debió fallar")
	}

	// Clave efímera diferente
	if cg.ValidateCookie(cookie, addr, []byte("different-ephemeral-key-bob-xxxx")) {
		t.Errorf("Cookie validada para clave efímera distinta debió fallar")
	}
}

