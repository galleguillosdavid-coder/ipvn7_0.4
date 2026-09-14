package l0

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/fxamacker/cbor/v2"
)

// Constantes del formato wire de ipvn7
const (
	MagicBytes uint32 = 0x49505637 // "IPV7"
	WireVersion uint8  = 0x01
	MaxPacketSize      = 1280 // Deterministic MTU
)

// Tipos de mensaje del protocolo
const (
	MsgTypeHandshakeInit uint8 = 1
	MsgTypeHandshakeResp uint8 = 2
	MsgTypeHandshakeAuth uint8 = 3
	MsgTypeData          uint8 = 4
	MsgTypeKeepAlive     uint8 = 5
	MsgTypeRoamingUpdate uint8 = 6
	MsgTypePoWChallenge  uint8 = 7
	MsgTypePoWResponse   uint8 = 8
)

// Packet representa el datagrama unificado de red de ipvn7
type Packet struct {
	Magic     uint32 `cbor:"1,keyasint"`
	Version   uint8  `cbor:"2,keyasint"`
	Type      uint8  `cbor:"3,keyasint"`
	SourceDID string `cbor:"4,keyasint"`
	DestDID   string `cbor:"5,keyasint"`
	Timestamp int64  `cbor:"6,keyasint"`
	Sequence  uint64 `cbor:"7,keyasint"`
	Nonce     []byte `cbor:"8,keyasint"`
	Payload   []byte `cbor:"9,keyasint"`
	Signature []byte `cbor:"10,keyasint,omitempty"`
}

var cborEncMode cbor.EncMode
var cborDecMode cbor.DecMode

func init() {
	var err error
	// Modo de codificación CBOR canónico determinista (RFC 8949)
	opts := cbor.CanonicalEncOptions()
	cborEncMode, err = opts.EncMode()
	if err != nil {
		panic(fmt.Sprintf("error configurando encoder CBOR: %v", err))
	}

	decOpts := cbor.DecOptions{}
	cborDecMode, err = decOpts.DecMode()
	if err != nil {
		panic(fmt.Sprintf("error configurando decoder CBOR: %v", err))
	}
}

// NewPacket crea un paquete básico con valores predeterminados
func NewPacket(msgType uint8, srcDID, dstDID string, seq uint64, nonce, payload []byte) *Packet {
	return &Packet{
		Magic:     MagicBytes,
		Version:   WireVersion,
		Type:      msgType,
		SourceDID: srcDID,
		DestDID:   dstDID,
		Timestamp: time.Now().UnixNano(),
		Sequence:  seq,
		Nonce:     nonce,
		Payload:   payload,
	}
}

// Encode serializa el paquete a formato determinista CBOR
func (p *Packet) Encode() ([]byte, error) {
	data, err := cborEncMode.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("error serializando paquete CBOR: %w", err)
	}
	if len(data) > MaxPacketSize {
		return nil, fmt.Errorf("paquete excede MTU determinista de %d bytes (tamaño: %d)", MaxPacketSize, len(data))
	}
	return data, nil
}

// DecodePacket deserializa y valida un datagrama CBOR
func DecodePacket(data []byte) (*Packet, error) {
	if len(data) < 5 {
		return nil, errors.New("datagrama demasiado corto")
	}
	if len(data) > MaxPacketSize {
		return nil, fmt.Errorf("datagrama excede tamaño máximo permitido (%d bytes)", MaxPacketSize)
	}

	var p Packet
	if err := cborDecMode.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("error deserializando paquete CBOR: %w", err)
	}

	if p.Magic != MagicBytes {
		return nil, fmt.Errorf("magic bytes incorrectos: 0x%X (esperado 0x%X)", p.Magic, MagicBytes)
	}
	if p.Version != WireVersion {
		return nil, fmt.Errorf("versión de protocolo no soportada: %d", p.Version)
	}

	return &p, nil
}

// SignPacket firma el contenido del paquete utilizando la identidad soberana
func (p *Packet) SignPacket(id *Identity) error {
	p.Signature = nil
	data, err := p.Encode()
	if err != nil {
		return err
	}
	p.Signature = id.Sign(data)
	return nil
}

// VerifyPacketSignature verifica la firma digital del paquete
func (p *Packet) VerifyPacketSignature() (bool, error) {
	if len(p.Signature) == 0 {
		return false, errors.New("el paquete no contiene firma digital")
	}

	pub, err := PublicKeyFromDID(p.SourceDID)
	if err != nil {
		return false, err
	}

	sig := p.Signature
	p.Signature = nil
	data, err := p.Encode()
	p.Signature = sig
	if err != nil {
		return false, err
	}

	return VerifySignature(pub, data, sig), nil
}

// QuickMagicCheck comprueba de forma lock-free si los primeros 4 bytes corresponden a ipvn7
func QuickMagicCheck(b []byte) bool {
	if len(b) < 4 {
		return false
	}
	// CBOR tag 1 con magic integer puede variar en encabezado,
	// pero para fast-path en eBPF se puede verificar magic
	return bytes.Contains(b[:min(16, len(b))], []byte{0x49, 0x50, 0x56, 0x37})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
