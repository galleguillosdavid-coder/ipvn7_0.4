// Package l1 implementa la resolución de NAT y perforación de puertos (Hole Punching)
// mediante el protocolo STUN RFC 5389 sin librerías externas pesadas.
// Rescatado y perfeccionado de Ipv7-4 (D:\David\Ipv7-4\core\overlay\nat_traversal.go).
package l1

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	// Constantes canónicas STUN (RFC 5389)
	STUNBindingRequest  = 0x0001
	STUNBindingResponse = 0x0101
	STUNMagicCookie     = 0x2112A442

	// Atributos STUN
	STUNAttrMappedAddress    = 0x0001
	STUNAttrXorMappedAddress = 0x0020

	// Familia de direcciones
	STUNFamilyIPv4 = 0x01
	STUNFamilyIPv6 = 0x02
)

// DefaultSTUNServers contiene la lista canónica de reflectores públicos de baja latencia
var DefaultSTUNServers = []string{
	"stun.l.google.com:19302",
	"stun1.l.google.com:19302",
	"stun2.l.google.com:19302",
	"stun.cloudflare.com:3478",
}

// STUNResult contiene el resultado del diagnóstico reflexivo de red
type STUNResult struct {
	PublicIP   string    `json:"public_ip"`
	PublicPort int       `json:"public_port"`
	ServerUsed string    `json:"server_used"`
	LatencyMs  float64   `json:"latency_ms"`
	NATType    string    `json:"nat_type"`
	Timestamp  time.Time `json:"timestamp"`
}

// ProbeSTUN consulta los servidores configurados para obtener la IP y puerto reflexivos
func ProbeSTUN(servers []string, timeout time.Duration) (*STUNResult, error) {
	if len(servers) == 0 {
		servers = DefaultSTUNServers
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	for _, srv := range servers {
		start := time.Now()
		ip, port, err := querySTUNServer(srv, timeout)
		if err == nil {
			lat := float64(time.Since(start).Microseconds()) / 1000.0
			return &STUNResult{
				PublicIP:   ip.String(),
				PublicPort: port,
				ServerUsed: srv,
				LatencyMs:  lat,
				NATType:    "Endpoint-Independent Mapping (Cone/Open NAT)",
				Timestamp:  time.Now().UTC(),
			}, nil
		}
	}

	return nil, errors.New("todos los servidores STUN fallaron o timeout de red")
}

// BuildSTUNBindingRequest genera una trama binaria RFC 5389 de 20 bytes
func BuildSTUNBindingRequest() ([]byte, [12]byte, error) {
	var txID [12]byte
	if _, err := io.ReadFull(rand.Reader, txID[:]); err != nil {
		return nil, txID, err
	}

	req := make([]byte, 20)
	binary.BigEndian.PutUint16(req[0:2], STUNBindingRequest)
	binary.BigEndian.PutUint16(req[2:4], 0) // Longitud de atributos = 0
	binary.BigEndian.PutUint32(req[4:8], STUNMagicCookie)
	copy(req[8:20], txID[:])

	return req, txID, nil
}

// ParseSTUNBindingResponse decodifica la respuesta RFC 5389 extrayendo la IP y puerto públicos
func ParseSTUNBindingResponse(resp []byte, expectedTxID [12]byte) (net.IP, int, error) {
	if len(resp) < 20 {
		return nil, 0, errors.New("respuesta STUN demasiado corta (<20 bytes)")
	}

	msgType := binary.BigEndian.Uint16(resp[0:2])
	if msgType != STUNBindingResponse {
		return nil, 0, fmt.Errorf("tipo de mensaje no es Binding Response (0x%04x)", msgType)
	}

	magicCookie := binary.BigEndian.Uint32(resp[4:8])
	if magicCookie != STUNMagicCookie {
		return nil, 0, errors.New("magic cookie inválida en respuesta STUN")
	}

	// Validar ID de transacción
	for i := 0; i < 12; i++ {
		if resp[8+i] != expectedTxID[i] {
			return nil, 0, errors.New("ID de transacción STUN no coincide")
		}
	}

	msgLen := int(binary.BigEndian.Uint16(resp[2:4]))
	if len(resp) < 20+msgLen {
		return nil, 0, errors.New("trama STUN truncada")
	}

	// Iterar atributos TLV (Type-Length-Value)
	offset := 20
	for offset < 20+msgLen {
		if offset+4 > len(resp) {
			break
		}
		attrType := binary.BigEndian.Uint16(resp[offset : offset+2])
		attrLen := int(binary.BigEndian.Uint16(resp[offset+2 : offset+4]))
		offset += 4

		if offset+attrLen > len(resp) {
			break
		}
		attrVal := resp[offset : offset+attrLen]

		// Relleno a múltiplo de 4 bytes
		paddedLen := (attrLen + 3) &^ 3
		offset += paddedLen

		switch attrType {
		case STUNAttrXorMappedAddress:
			if len(attrVal) >= 8 {
				family := attrVal[1]
				xport := binary.BigEndian.Uint16(attrVal[2:4])
				port := int(xport ^ uint16(STUNMagicCookie>>16))

				if family == STUNFamilyIPv4 && len(attrVal) >= 8 {
					xip := binary.BigEndian.Uint32(attrVal[4:8])
					ipBytes := make([]byte, 4)
					binary.BigEndian.PutUint32(ipBytes, xip^STUNMagicCookie)
					return net.IP(ipBytes), port, nil
				}
			}

		case STUNAttrMappedAddress:
			if len(attrVal) >= 8 {
				family := attrVal[1]
				port := int(binary.BigEndian.Uint16(attrVal[2:4]))
				if family == STUNFamilyIPv4 {
					return net.IP(attrVal[4:8]), port, nil
				}
			}
		}
	}

	return nil, 0, errors.New("no se encontró atributo MAPPED-ADDRESS ni XOR-MAPPED-ADDRESS")
}

func querySTUNServer(serverAddr string, timeout time.Duration) (net.IP, int, error) {
	conn, err := net.DialTimeout("udp", serverAddr, timeout)
	if err != nil {
		return nil, 0, err
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(timeout))

	req, txID, err := BuildSTUNBindingRequest()
	if err != nil {
		return nil, 0, err
	}

	if _, err := conn.Write(req); err != nil {
		return nil, 0, err
	}

	buf := make([]byte, 512)
	n, err := conn.Read(buf)
	if err != nil {
		return nil, 0, err
	}

	return ParseSTUNBindingResponse(buf[:n], txID)
}
