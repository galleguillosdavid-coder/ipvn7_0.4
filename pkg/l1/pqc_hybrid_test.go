package l1

import (
	"bytes"
	"testing"
)

func TestHybridPQCKeygenSignAndVerify(t *testing.T) {
	did := "did:ipvn7:test-node-pqc"
	kp, err := GenerateHybridKeyPair(did)
	if err != nil {
		t.Fatalf("GenerateHybridKeyPair failed: %v", err)
	}

	if kp.Ed25519PubHex == "" || kp.X25519PubHex == "" || kp.MLDSAPubHex == "" || kp.MLKEMPubHex == "" {
		t.Fatalf("Claves públicas incompletas en HybridKeyPair: %+v", kp)
	}

	msg := []byte("Axiom: Zero-PII y Core Freeze inalterable en malla cuántica.")
	sig, err := kp.Sign(msg)
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if sig.Algorithm != HybridSigAlgorithm {
		t.Errorf("Algoritmo inesperado: %s", sig.Algorithm)
	}

	if !kp.Verify(msg, sig) {
		t.Fatalf("Verify rechazó firma híbrida válida!")
	}

	// Probar alteración de mensaje
	tamperedMsg := []byte("Axiom: Alteración maliciosa")
	if kp.Verify(tamperedMsg, sig) {
		t.Fatalf("Verify aceptó mensaje alterado!")
	}

	// Probar alteración de firma clásica
	sigCorrupt := *sig
	sigCorrupt.ClassicalSig = make([]byte, len(sig.ClassicalSig))
	copy(sigCorrupt.ClassicalSig, sig.ClassicalSig)
	sigCorrupt.ClassicalSig[0] ^= 0xFF
	if kp.Verify(msg, &sigCorrupt) {
		t.Fatalf("Verify aceptó firma clásica corrompida!")
	}

	// Probar alteración de firma cuántica
	sigCorruptPQC := *sig
	sigCorruptPQC.PQCSig = make([]byte, len(sig.PQCSig))
	copy(sigCorruptPQC.PQCSig, sig.PQCSig)
	sigCorruptPQC.PQCSig[0] ^= 0xFF
	if kp.Verify(msg, &sigCorruptPQC) {
		t.Fatalf("Verify aceptó firma cuántica corrompida!")
	}
}

func TestHybridPQCKEMAndAEAD(t *testing.T) {
	// Receptor (Bob)
	bobKP, err := GenerateHybridKeyPair("did:ipvn7:bob")
	if err != nil {
		t.Fatalf("Error generando clave de Bob: %v", err)
	}

	// Emisor (Alice) encapsula para Bob
	aliceKey, ciphertext, err := Encapsulate(bobKP.ClassicalKEMPub, bobKP.MLKEMPubHex)
	if err != nil {
		t.Fatalf("Encapsulate failed: %v", err)
	}

	if len(aliceKey) != SharedSecretSize {
		t.Fatalf("Tamaño de clave compartida inválido: %d", len(aliceKey))
	}

	// Bob desencapsula
	bobKey, err := bobKP.Decapsulate(ciphertext)
	if err != nil {
		t.Fatalf("Decapsulate failed: %v", err)
	}

	if !bytes.Equal(aliceKey, bobKey) {
		t.Fatalf("Claves compartidas no coinciden! Alice: %x, Bob: %x", aliceKey, bobKey)
	}

	// Probar cifrado y descifrado de carga útil con ChaCha20-Poly1305
	plaintext := []byte("Datagrama determinista ipvn7 de 1280 bytes protegido cuánticamente")
	ad := []byte("did:ipvn7:bob")

	encrypted, err := EncryptPayload(aliceKey, plaintext, ad)
	if err != nil {
		t.Fatalf("EncryptPayload failed: %v", err)
	}

	decrypted, err := DecryptPayload(bobKey, encrypted, ad)
	if err != nil {
		t.Fatalf("DecryptPayload failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Fatalf("Descifrado no coincide con el texto plano original")
	}

	// Probar alteración de datos asociados (AD)
	_, err = DecryptPayload(bobKey, encrypted, []byte("did:ipvn7:mallory"))
	if err == nil {
		t.Fatalf("DecryptPayload debió fallar por AD alterado!")
	}
}
