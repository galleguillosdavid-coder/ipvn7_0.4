package l4

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Catálogo de 32 emojis seleccionados por máxima diferenciación visual y cultural
var SASEmojiCatalog = []string{
	"🚀", "🛡️", "🔑", "⚡", "🌟", "🔥", "💎", "🪐",
	"🎯", "🧭", "⚓", "🛸", "🌈", "🌊", "🔮", "🍀",
	"🦁", "🦅", "🐬", "🦊", "👑", "🏆", "🎁", "🎨",
	"🌲", "🍄", "🍎", "🍉", "☕", "🎷", "🎸", "🚲",
}

// SASCode contiene los dos componentes de validación visual humana (Dimensión 10)
type SASCode struct {
	Digits string   `json:"digits"` // Código de 6 dígitos numéricos (ej. "482915")
	Emojis []string `json:"emojis"` // Secuencia de 4 emojis visuales
}

// DeriveSAS genera el Código Corto de Autenticación a partir de dos claves públicas Ed25519
func DeriveSAS(pubA, pubB []byte) *SASCode {
	// Ordenar canónicamente las dos claves para que ambos extremos deriven exactamente lo mismo
	var combined []byte
	if bytes.Compare(pubA, pubB) < 0 {
		combined = append(append([]byte{}, pubA...), pubB...)
	} else {
		combined = append(append([]byte{}, pubB...), pubA...)
	}

	h := sha256.Sum256(combined)

	// 1. Extraer 6 dígitos numéricos usando los primeros 4 bytes
	num := (uint32(h[0])<<24 | uint32(h[1])<<16 | uint32(h[2])<<8 | uint32(h[3])) % 1000000
	digits := fmt.Sprintf("%06d", num)

	// 2. Extraer 4 emojis usando los siguientes 4 bytes
	emojis := make([]string, 4)
	catalogLen := len(SASEmojiCatalog)
	for i := 0; i < 4; i++ {
		idx := int(h[4+i]) % catalogLen
		emojis[i] = SASEmojiCatalog[idx]
	}

	return &SASCode{
		Digits: digits,
		Emojis: emojis,
	}
}

// URFrame representa una trama individual para Códigos QR Animados tipo Uniform Resource (UR)
type URFrame struct {
	Index int    `json:"index"`
	Total int    `json:"total"`
	Data  string `json:"data"` // Representación formateada "ur:ipvn7-profile/x-y/..."
}

// GenerateURChunks divide un perfil de configuración en fragmentos UR para animación QR
func GenerateURChunks(payload []byte, maxChunkSize int) []string {
	if maxChunkSize <= 0 {
		maxChunkSize = 64 // Tamaño estándar de lectura visual rápida
	}

	encoded := base64.RawURLEncoding.EncodeToString(payload)
	totalLen := len(encoded)
	totalChunks := (totalLen + maxChunkSize - 1) / maxChunkSize

	if totalChunks == 0 {
		totalChunks = 1
	}

	chunks := make([]string, totalChunks)
	for i := 0; i < totalChunks; i++ {
		start := i * maxChunkSize
		end := start + maxChunkSize
		if end > totalLen {
			end = totalLen
		}
		part := encoded[start:end]
		chunks[i] = fmt.Sprintf("ur:ipvn7-profile/%d-%d/%s", i+1, totalChunks, part)
	}

	return chunks
}

// ReconstructURChunks reconstruye el perfil original a partir de las tramas UR recolectadas
func ReconstructURChunks(frames []string) ([]byte, error) {
	if len(frames) == 0 {
		return nil, errors.New("no se recibieron tramas UR")
	}

	// Mapa para deduplicar y ordenar tramas por su índice
	parts := make(map[int]string)
	expectedTotal := -1

	for _, frame := range frames {
		partsArr := strings.Split(frame, "/")
		if len(partsArr) != 3 || partsArr[0] != "ur:ipvn7-profile" {
			return nil, fmt.Errorf("formato UR inválido en trama: %s", frame)
		}

		idxParts := strings.Split(partsArr[1], "-")
		if len(idxParts) != 2 {
			return nil, fmt.Errorf("índice de trama malformado: %s", partsArr[1])
		}

		idx, err1 := strconv.Atoi(idxParts[0])
		total, err2 := strconv.Atoi(idxParts[1])
		if err1 != nil || err2 != nil {
			return nil, errors.New("índice numérico inválido en UR")
		}

		if expectedTotal == -1 {
			expectedTotal = total
		} else if expectedTotal != total {
			return nil, errors.New("inconsistencia en el número total de tramas UR")
		}

		parts[idx] = partsArr[2]
	}

	if len(parts) != expectedTotal {
		return nil, fmt.Errorf("perfil incompleto: se tienen %d de %d tramas necesarias", len(parts), expectedTotal)
	}

	// Reensamblar en orden estricto
	var sb strings.Builder
	for i := 1; i <= expectedTotal; i++ {
		part, ok := parts[i]
		if !ok {
			return nil, fmt.Errorf("falta la trama %d", i)
		}
		sb.WriteString(part)
	}

	decoded, err := base64.RawURLEncoding.DecodeString(sb.String())
	if err != nil {
		return nil, fmt.Errorf("error decodificando carga UR Base64: %w", err)
	}

	return decoded, nil
}
