package l1

import (
	"bytes"
	"testing"

	"ipvn7/pkg/l0"
)

func TestSimulatedTunAdapter_LifecycleAndMTU(t *testing.T) {
	id, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("Error generando identidad L0: %v", err)
	}

	adapter := NewSimulatedTunAdapter(id)
	defer adapter.Close()

	if adapter.MTU() != DefaultMTU {
		t.Errorf("Esperaba MTU %d, obtuve %d", DefaultMTU, adapter.MTU())
	}
	if !adapter.IsSimulated() {
		t.Errorf("Esperaba adaptador simulado true")
	}
	if adapter.IPv4() == nil || adapter.IPv6() == nil {
		t.Errorf("Direcciones IP del adaptador no deben ser nulas")
	}

	// 1. Inyección y lectura de paquete estándar
	payload := []byte("DATAGRAMA_SOBERANO_1280")
	if err := adapter.InjectPacket(payload); err != nil {
		t.Fatalf("Error inyectando paquete en TUN: %v", err)
	}

	readPkt, err := adapter.ReadPacket()
	if err != nil {
		t.Fatalf("Error leyendo paquete desde TUN: %v", err)
	}
	if !bytes.Equal(readPkt, payload) {
		t.Errorf("Contenido del paquete difiere: esperaba %s, obtuve %s", payload, readPkt)
	}

	// 2. Escritura de paquete hacia el TUN
	outPayload := []byte("RESPUESTA_HACIA_OS")
	if err := adapter.WritePacket(outPayload); err != nil {
		t.Fatalf("Error escribiendo paquete al TUN: %v", err)
	}

	// 3. Verificación de MTU Enforcement (Descarte de tramas > 1280B)
	oversized := make([]byte, DefaultMTU+100)
	if err := adapter.WritePacket(oversized); err == nil {
		t.Errorf("Se esperaba error por paquete que excede el MTU determinista")
	}
}

func TestCreateTunAdapter_Fallback(t *testing.T) {
	id, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("Error generando identidad L0: %v", err)
	}

	// Solicitar nativo debe degradar a simulado sin lanzar pánicos
	adapter, err := CreateTunAdapter(id, true)
	if err != nil {
		t.Fatalf("CreateTunAdapter falló: %v", err)
	}
	if adapter == nil {
		t.Fatal("El adaptador retornado no debe ser nil")
	}
	if !adapter.IsSimulated() {
		t.Errorf("En entorno sin permisos debe caer en modo simulado")
	}
}
