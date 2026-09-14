// Package l1 implementa la Separación Estricta de Dominios Criptográficos
// rescatada de ip7uin_MVP_Spec_v1.1.docx (Secciones 2 y 7).
// Principio: Inyección obligatoria de contextos canónicos en el AAD (Additional Authenticated Data)
// de cada operación AEAD (ChaCha20-Poly1305 / AES-GCM), impidiendo ataques de sustitución
// cruzada donde tramas de control o descubrimiento sean reinterpretadas como datos de usuario.
package l1

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
)

// Dominios canónicos cerrados según especificación
const (
	DomainDataV2      = "ip7:data:v2"      // Carga útil de usuario y aplicaciones
	DomainControlV1   = "ip7:control:v1"   // Señalización, gestión de enlaces y enrutamiento
	DomainHandshakeV1 = "ip7:handshake:v1" // Establecimiento de sesión y KEM
	DomainDiscoveryV1 = "ip7:discovery:v1" // Consultas silenciosas UIN y Petnames
)

// BuildDomainAAD construye la estructura binaria canónica para el AAD de cifrado AEAD
func BuildDomainAAD(domain string, originDID string, seq uint64) []byte {
	domBytes := []byte(domain)
	didBytes := []byte(originDID)

	buf := make([]byte, 2+len(domBytes)+2+len(didBytes)+8)
	offset := 0

	binary.BigEndian.PutUint16(buf[offset:offset+2], uint16(len(domBytes)))
	offset += 2
	copy(buf[offset:offset+len(domBytes)], domBytes)
	offset += len(domBytes)

	binary.BigEndian.PutUint16(buf[offset:offset+2], uint16(len(didBytes)))
	offset += 2
	copy(buf[offset:offset+len(didBytes)], didBytes)
	offset += len(didBytes)

	binary.BigEndian.PutUint64(buf[offset:offset+8], seq)
	return buf
}

// ComputeDomainHash calcula el digest determinista de un contexto de dominio
func ComputeDomainHash(domain string) [32]byte {
	return sha256.Sum256([]byte(fmt.Sprintf("ipvn7:domain:%s", domain)))
}
