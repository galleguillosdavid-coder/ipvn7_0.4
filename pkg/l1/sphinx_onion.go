// Package l1 implementa el enrutamiento cebolla Sphinx de 3 saltos
// con tramas de longitud fija estricta de 1280 bytes (Deterministic MTU Wire),
// desprendimiento iterativo de capas (peeling) y protección anti-repetición O(1).
// Conforme a genesis.md (L178) y 2.md (Sección 2).
package l1

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"golang.org/x/crypto/chacha20"
)

const (
	// SphinxPacketSize define el tamaño inmutable del wire determinista (1280 bytes)
	SphinxPacketSize = 1280

	// Topología de 3 saltos
	SphinxHopsCount = 3

	// Estructura de cabeceras sincronizadas por capas
	SphinxHopKEMSize        = 160 // 32 bytes X25519 efímero + 128 bytes ML-KEM ciphertext
	SphinxKeyHeaderSize     = SphinxHopsCount * SphinxHopKEMSize // 480 bytes
	SphinxHopDescriptorSize = 128 // Tamaño fijo de descriptor por cada salto
	SphinxRoutingTotalSize  = SphinxHopsCount * SphinxHopDescriptorSize // 384 bytes
	SphinxPayloadSize       = SphinxPacketSize - SphinxKeyHeaderSize - SphinxRoutingTotalSize // 416 bytes

	// Acciones de peeling
	SphinxActionForward = "FORWARD"
	SphinxActionDeliver = "DELIVER"
	SphinxActionDrop    = "DROP"
)

// SphinxHopNode define la identidad y claves públicas de un nodo en el circuito
type SphinxHopNode struct {
	DID          string             `json:"did"`
	Address      string             `json:"address"`
	HybridKeys   *HybridKeyPair     `json:"-"`
	X25519PubHex string             `json:"x25519_pub_hex"`
	MLKEMPubHex  string             `json:"ml_kem_pub_hex"`
}

// SphinxCircuit contiene los 3 nodos seleccionados: Guard, Middle, Exit
type SphinxCircuit struct {
	CircuitID string          `json:"circuit_id"`
	Guard     *SphinxHopNode  `json:"guard"`
	Middle    *SphinxHopNode  `json:"middle"`
	Exit      *SphinxHopNode  `json:"exit"`
	CreatedAt time.Time       `json:"created_at"`
}

// SphinxPacket representa el paquete serializado de 1280 bytes exactos
type SphinxPacket struct {
	KeyHeader      [SphinxKeyHeaderSize]byte    `json:"-"`
	RoutingHeader  [SphinxRoutingTotalSize]byte `json:"-"`
	Payload        [SphinxPayloadSize]byte      `json:"-"`
}

// PeelResult contiene el resultado de procesar un salto cebolla
type PeelResult struct {
	Action      string        `json:"action"`       // FORWARD, DELIVER, DROP
	CircuitID   string        `json:"circuit_id"`
	NextHopDID  string        `json:"next_hop_did"`
	NextPacket  *SphinxPacket `json:"-"`
	RawPayload  []byte        `json:"raw_payload,omitempty"`
	IsExit      bool          `json:"is_exit"`
	PeelTimeUs  int64         `json:"peel_time_us"`
	TagHash     string        `json:"tag_hash"`
}

// SphinxRouter gestiona la creación de circuitos, envío y peeling con protección anti-replay
type SphinxRouter struct {
	mu           sync.RWMutex
	LocalKeys    *HybridKeyPair
	replayCache  map[string]int64 // tagHash -> expiración epoch unix
	replayOrder  []string
	maxCacheSize int
}

// NewSphinxRouter inicializa el enrutador cebolla
func NewSphinxRouter(localKeys *HybridKeyPair) *SphinxRouter {
	return &SphinxRouter{
		LocalKeys:    localKeys,
		replayCache:  make(map[string]int64),
		replayOrder:  make([]string, 0, 10000),
		maxCacheSize: 10000,
	}
}

// BuildPacket construye un paquete cebolla determinista de 1280 bytes a través de los 3 saltos
func (sr *SphinxRouter) BuildPacket(circuit *SphinxCircuit, finalPayload []byte, destService string) (*SphinxPacket, error) {
	if circuit == nil || circuit.Guard == nil || circuit.Middle == nil || circuit.Exit == nil {
		return nil, errors.New("circuito incompleto: se requieren 3 saltos (Guard, Middle, Exit)")
	}

	hops := []*SphinxHopNode{circuit.Guard, circuit.Middle, circuit.Exit}

	// 1. Derivar claves de sesión simétricas para cada salto usando KEM híbrido
	hopKeys := make([][]byte, SphinxHopsCount)
	var keyHeader [SphinxKeyHeaderSize]byte

	for i, hop := range hops {
		sharedKey, kemCipher, err := Encapsulate(hop.HybridKeys.ClassicalKEMPub, hop.HybridKeys.MLKEMPubHex)
		if err != nil {
			return nil, fmt.Errorf("error KEM en salto %d (%s): %w", i, hop.DID, err)
		}
		hopKeys[i] = sharedKey

		// Copiar KEM efímero en la posición correspondiente del encabezado de claves
		slotStart := i * SphinxHopKEMSize
		copy(keyHeader[slotStart:slotStart+32], kemCipher.EphemeralX25519)
		copy(keyHeader[slotStart+32:slotStart+160], kemCipher.PQCCiphertext)
	}

	// 2. Preparar el bloque de carga útil (exactamente SphinxPayloadSize bytes)
	var payloadBlock [SphinxPayloadSize]byte
	if len(finalPayload) > SphinxPayloadSize-32 {
		return nil, fmt.Errorf("payload excede el límite del frame (%d > %d bytes)", len(finalPayload), SphinxPayloadSize-32)
	}

	// Codificar metadatos de destino en el inicio del payload interno
	var innerBuf bytes.Buffer
	binary.Write(&innerBuf, binary.BigEndian, uint16(len(destService)))
	innerBuf.WriteString(destService)
	binary.Write(&innerBuf, binary.BigEndian, uint32(len(finalPayload)))
	innerBuf.Write(finalPayload)

	copy(payloadBlock[:], innerBuf.Bytes())

	// Rellenar el resto del payload con bytes pseudoaleatorios deterministas (zero-leak)
	if _, err := io.ReadFull(rand.Reader, payloadBlock[innerBuf.Len():]); err != nil {
		return nil, err
	}

	// 3. Encriptar el payload en capas inversas: Exit (hop 2) -> Middle (hop 1) -> Guard (hop 0)
	for i := SphinxHopsCount - 1; i >= 0; i-- {
		streamCipher, err := chacha20.NewUnauthenticatedCipher(hopKeys[i], make([]byte, 12))
		if err != nil {
			return nil, err
		}
		streamCipher.XORKeyStream(payloadBlock[:], payloadBlock[:])
	}

	// 4. Construir descriptores de enrutamiento para cada salto
	var routingHeader [SphinxRoutingTotalSize]byte

	// Salto 2 (Exit): Próximo destino es LOCAL
	descExit := make([]byte, SphinxHopDescriptorSize)
	copy(descExit[0:32], []byte("EXIT_NODE_LOCAL_DELIVERY"))
	copy(descExit[32:48], []byte(circuit.CircuitID))
	descExit[48] = 0x01 // Flag EXIT

	// Salto 1 (Middle): Próximo destino es Exit
	descMiddle := make([]byte, SphinxHopDescriptorSize)
	copy(descMiddle[0:32], []byte(circuit.Exit.DID))
	copy(descMiddle[32:48], []byte(circuit.CircuitID))
	descMiddle[48] = 0x00 // Flag RELAY

	// Salto 0 (Guard): Próximo destino es Middle
	descGuard := make([]byte, SphinxHopDescriptorSize)
	copy(descGuard[0:32], []byte(circuit.Middle.DID))
	copy(descGuard[32:48], []byte(circuit.CircuitID))
	descGuard[48] = 0x00 // Flag RELAY

	descriptors := [][]byte{descGuard, descMiddle, descExit}

	// Cifrar cada descriptor con la clave simétrica del salto respectivo
	for i := 0; i < SphinxHopsCount; i++ {
		// Calcular MAC sobre descriptor
		mac := hmac.New(sha256.New, hopKeys[i])
		mac.Write(descriptors[i][:49])
		macSum := mac.Sum(nil)
		copy(descriptors[i][49:65], macSum[:16])

		// Cifrar con ChaCha20
		stream, err := chacha20.NewUnauthenticatedCipher(hopKeys[i], make([]byte, 12))
		if err != nil {
			return nil, err
		}
		stream.XORKeyStream(descriptors[i], descriptors[i])
		copy(routingHeader[i*SphinxHopDescriptorSize:(i+1)*SphinxHopDescriptorSize], descriptors[i])
	}

	packet := &SphinxPacket{
		KeyHeader:     keyHeader,
		RoutingHeader: routingHeader,
		Payload:       payloadBlock,
	}

	return packet, nil
}

// PeelLayer procesa un paquete cebolla en el nodo actual, pelando una capa de cifrado
func (sr *SphinxRouter) PeelLayer(packet *SphinxPacket) (*PeelResult, error) {
	start := time.Now()

	if packet == nil {
		return nil, errors.New("paquete nulo")
	}

	// 1. Control Anti-Repetición O(1) usando el hash de la primera ranura de clave
	tagHashBytes := sha256.Sum256(packet.KeyHeader[:SphinxHopKEMSize])
	tagHash := hex.EncodeToString(tagHashBytes[:])

	sr.mu.Lock()
	now := time.Now().Unix()
	if exp, exists := sr.replayCache[tagHash]; exists && exp > now {
		sr.mu.Unlock()
		return &PeelResult{
			Action:     SphinxActionDrop,
			TagHash:    tagHash,
			PeelTimeUs: time.Since(start).Microseconds(),
		}, errors.New("ataque de repetición detectado: paquete cebolla ya procesado")
	}
	sr.replayCache[tagHash] = now + 300 // TTL 5 minutos
	sr.replayOrder = append(sr.replayOrder, tagHash)
	if len(sr.replayOrder) > sr.maxCacheSize {
		oldest := sr.replayOrder[0]
		sr.replayOrder = sr.replayOrder[1:]
		delete(sr.replayCache, oldest)
	}
	sr.mu.Unlock()

	// 2. Derivar clave compartida para este salto a partir de la primera ranura de la cabecera efímera
	kemCipher := &HybridKEMCiphertext{
		Algorithm:       HybridKEMAlgorithm,
		EphemeralX25519: packet.KeyHeader[0:32],
		PQCCiphertext:   packet.KeyHeader[32:160],
		Salt:            make([]byte, 16),
	}

	hopKey, err := sr.LocalKeys.Decapsulate(kemCipher)
	if err != nil {
		return nil, fmt.Errorf("error decapsulando KEM en salto: %w", err)
	}

	// 3. Descifrar el descriptor de enrutamiento actual (primeros 128 bytes)
	descBytes := make([]byte, SphinxHopDescriptorSize)
	copy(descBytes, packet.RoutingHeader[:SphinxHopDescriptorSize])

	stream, err := chacha20.NewUnauthenticatedCipher(hopKey, make([]byte, 12))
	if err != nil {
		return nil, err
	}
	stream.XORKeyStream(descBytes, descBytes)

	// Validar MAC del descriptor
	mac := hmac.New(sha256.New, hopKey)
	mac.Write(descBytes[:49])
	expectedMAC := mac.Sum(nil)[:16]
	if !hmac.Equal(descBytes[49:65], expectedMAC) {
		return nil, errors.New("descriptor de enrutamiento corrupto o clave incorrecta (fallo de MAC)")
	}

	nextHopDID := string(bytes.Trim(descBytes[0:32], "\x00"))
	circuitID := string(bytes.Trim(descBytes[32:48], "\x00"))
	isExit := descBytes[48] == 0x01 || nextHopDID == "EXIT_NODE_LOCAL_DELIVERY"

	// 4. Pelar una capa del payload con ChaCha20
	peeledPayload := packet.Payload
	streamPayload, err := chacha20.NewUnauthenticatedCipher(hopKey, make([]byte, 12))
	if err != nil {
		return nil, err
	}
	streamPayload.XORKeyStream(peeledPayload[:], peeledPayload[:])

	// 5. Determinar acción: DELIVER (nodo de salida) o FORWARD (nodo de paso)
	if isExit {
		// Extraer carga útil y servicio destino
		var serviceLen uint16
		r := bytes.NewReader(peeledPayload[:])
		_ = binary.Read(r, binary.BigEndian, &serviceLen)
		serviceBytes := make([]byte, serviceLen)
		_, _ = r.Read(serviceBytes)

		var payloadLen uint32
		_ = binary.Read(r, binary.BigEndian, &payloadLen)
		if payloadLen > uint32(SphinxPayloadSize) {
			payloadLen = uint32(SphinxPayloadSize - int(serviceLen) - 6)
		}
		rawInner := make([]byte, payloadLen)
		_, _ = r.Read(rawInner)

		return &PeelResult{
			Action:     SphinxActionDeliver,
			CircuitID:  circuitID,
			NextHopDID: "LOCAL",
			RawPayload: rawInner,
			IsExit:     true,
			TagHash:    tagHash,
			PeelTimeUs: time.Since(start).Microseconds(),
		}, nil
	}

	// 6. Nodo intermedio: Desplazar KeyHeader y RoutingHeader y rellenar con ruido pseudoaleatorio
	var nextKeyHeader [SphinxKeyHeaderSize]byte
	copy(nextKeyHeader[:], packet.KeyHeader[SphinxHopKEMSize:])
	noiseKey := make([]byte, SphinxHopKEMSize)
	io.ReadFull(rand.Reader, noiseKey)
	copy(nextKeyHeader[SphinxKeyHeaderSize-SphinxHopKEMSize:], noiseKey)

	var nextRoutingHeader [SphinxRoutingTotalSize]byte
	copy(nextRoutingHeader[:], packet.RoutingHeader[SphinxHopDescriptorSize:])
	noiseRouting := make([]byte, SphinxHopDescriptorSize)
	io.ReadFull(rand.Reader, noiseRouting)
	copy(nextRoutingHeader[SphinxRoutingTotalSize-SphinxHopDescriptorSize:], noiseRouting)

	nextPacket := &SphinxPacket{
		KeyHeader:     nextKeyHeader,
		RoutingHeader: nextRoutingHeader,
		Payload:       peeledPayload,
	}

	return &PeelResult{
		Action:     SphinxActionForward,
		CircuitID:  circuitID,
		NextHopDID: nextHopDID,
		NextPacket: nextPacket,
		IsExit:     false,
		TagHash:    tagHash,
		PeelTimeUs: time.Since(start).Microseconds(),
	}, nil
}

// Serialize convierte el paquete a exactamente 1280 bytes para el wire
func (p *SphinxPacket) Serialize() []byte {
	out := make([]byte, SphinxPacketSize)
	copy(out[0:SphinxKeyHeaderSize], p.KeyHeader[:])
	copy(out[SphinxKeyHeaderSize:SphinxKeyHeaderSize+SphinxRoutingTotalSize], p.RoutingHeader[:])
	copy(out[SphinxKeyHeaderSize+SphinxRoutingTotalSize:], p.Payload[:])
	return out
}

// Deserialize reconstruye el paquete validando el límite estricto de 1280 bytes
func DeserializeSphinxPacket(wire []byte) (*SphinxPacket, error) {
	if len(wire) != SphinxPacketSize {
		return nil, fmt.Errorf("tamaño wire inválido (%d bytes): se requieren exactamente %d bytes deterministas", len(wire), SphinxPacketSize)
	}

	var p SphinxPacket
	copy(p.KeyHeader[:], wire[0:SphinxKeyHeaderSize])
	copy(p.RoutingHeader[:], wire[SphinxKeyHeaderSize:SphinxKeyHeaderSize+SphinxRoutingTotalSize])
	copy(p.Payload[:], wire[SphinxKeyHeaderSize+SphinxRoutingTotalSize:])
	return &p, nil
}
