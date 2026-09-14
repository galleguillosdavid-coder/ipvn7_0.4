package l1_test

import (
	"crypto/ed25519"
	"net"
	"testing"

	"ipvn7/pkg/l0"
	"ipvn7/pkg/l1"
)

func TestKleinbergRouterAndRings(t *testing.T) {
	localID, _ := l0.GenerateIdentity()
	router := l1.NewKleinbergRouter(localID)

	// Agregar pares distribuidos a lo largo del espacio de claves (diferentes anillos)
	totalAdded := 0
	for ringByte := 0; ringByte < 24; ringByte++ {
		// Generar clave con distancia controlada respecto a localID
		craftedPub := make([]byte, 32)
		copy(craftedPub, localID.PublicKey)
		craftedPub[ringByte%32] ^= 0x80 // Invertir bit para variar distancia XOR

		did := l0.DIDFromPublicKey(ed25519.PublicKey(craftedPub))
		addr := &net.UDPAddr{IP: net.ParseIP("192.168.1.100"), Port: 7000 + ringByte}
		if err := router.AddOrUpdatePeer(did, addr, float64(ringByte*2)); err == nil {
			totalAdded++
		}
	}

	peers := router.GetAllPeers()
	if len(peers) == 0 {
		t.Fatalf("No se registraron pares en la tabla de enrutamiento")
	}

	// Probar búsqueda voraz de siguiente salto hacia un par existente
	targetPeer := peers[0]
	nextHop, err := router.FindNextHop(targetPeer.DID)
	if err != nil {
		t.Fatalf("FindNextHop falló: %v", err)
	}
	if nextHop.DID != targetPeer.DID {
		t.Errorf("FindNextHop debió resolver directamente el destino conocido")
	}

	// Probar búsqueda voraz hacia un DID desconocido
	unknownID, _ := l0.GenerateIdentity()
	bestHop, err := router.FindNextHop(unknownID.DID())
	if err != nil {
		t.Fatalf("FindNextHop hacia nodo desconocido falló: %v", err)
	}
	if bestHop == nil {
		t.Errorf("FindNextHop debió seleccionar el mejor salto aproximado")
	}
}

func TestSimulatedTunAdapter(t *testing.T) {
	id, _ := l0.GenerateIdentity()
	tun := l1.NewSimulatedTunAdapter(id)
	defer tun.Close()

	if tun.MTU() != 1280 {
		t.Errorf("MTU esperada: 1280, obtenida: %d", tun.MTU())
	}

	packetData := []byte("Paquete sintético TUN")
	if err := tun.InjectPacket(packetData); err != nil {
		t.Fatalf("InjectPacket falló: %v", err)
	}

	readData, err := tun.ReadPacket()
	if err != nil {
		t.Fatalf("ReadPacket falló: %v", err)
	}

	if string(readData) != string(packetData) {
		t.Errorf("Datos leídos no coinciden con datos inyectados")
	}
}
