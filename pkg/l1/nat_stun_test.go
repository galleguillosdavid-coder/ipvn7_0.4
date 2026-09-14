package l1

import (
	"encoding/binary"
	"net"
	"testing"
)

func TestSTUNHeaderGenerationAndParsing(t *testing.T) {
	req, txID, err := BuildSTUNBindingRequest()
	if err != nil {
		t.Fatalf("BuildSTUNBindingRequest failed: %v", err)
	}

	if len(req) != 20 {
		t.Fatalf("Expected 20 bytes STUN header, got %d", len(req))
	}

	msgType := binary.BigEndian.Uint16(req[0:2])
	if msgType != STUNBindingRequest {
		t.Errorf("Expected message type 0x%04x, got 0x%04x", STUNBindingRequest, msgType)
	}

	cookie := binary.BigEndian.Uint32(req[4:8])
	if cookie != STUNMagicCookie {
		t.Errorf("Expected magic cookie 0x%08x, got 0x%08x", STUNMagicCookie, cookie)
	}

	if len(txID) != 12 {
		t.Fatalf("Expected 12 bytes txID, got %d", len(txID))
	}
}

func TestSTUNMockResponseParsing(t *testing.T) {
	// Build a simulated STUN Binding Success Response with XOR-MAPPED-ADDRESS
	// Magic Cookie: 0x2112A442
	// Target Port: 12345 (0x3039) -> XOR with (0x2112) = 0x112B
	// Target IP: 203.0.113.195 (0xCB0071C3) -> XOR with (0x2112A442) = 0xEA12D581
	txID := [12]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12}

	buf := make([]byte, 20+8+4) // 20 header + 4 attr header + 8 attr body
	binary.BigEndian.PutUint16(buf[0:2], STUNBindingResponse)
	binary.BigEndian.PutUint16(buf[2:4], 12) // Message length of attributes
	binary.BigEndian.PutUint32(buf[4:8], STUNMagicCookie)
	copy(buf[8:20], txID[:])

	// Attribute: XOR-MAPPED-ADDRESS (0x0020)
	binary.BigEndian.PutUint16(buf[20:22], STUNAttrXorMappedAddress)
	binary.BigEndian.PutUint16(buf[22:24], 8) // Length: 8 bytes
	buf[24] = 0                              // reserved
	buf[25] = STUNFamilyIPv4                  // IPv4
	binary.BigEndian.PutUint16(buf[26:28], 12345^0x2112)

	ipBytes := net.ParseIP("203.0.113.195").To4()
	xorIP := binary.BigEndian.Uint32(ipBytes) ^ STUNMagicCookie
	binary.BigEndian.PutUint32(buf[28:32], xorIP)

	ip, port, err := ParseSTUNBindingResponse(buf, txID)
	if err != nil {
		t.Fatalf("ParseSTUNBindingResponse failed: %v", err)
	}

	if ip.String() != "203.0.113.195" {
		t.Errorf("Expected IP 203.0.113.195, got %s", ip.String())
	}
	if port != 12345 {
		t.Errorf("Expected port 12345, got %d", port)
	}
}
