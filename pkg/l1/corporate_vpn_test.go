package l1

import (
	"bytes"
	"testing"

	"ipvn7/pkg/l0"
)

func TestCorporateVPNZeroAdminFallback(t *testing.T) {
	id, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("error generando identidad: %v", err)
	}

	cfg := DefaultCorporateVPNConfig()
	simTun := NewSimulatedTunAdapter(id)
	vpn := NewCorporateVPNManager(id, cfg, simTun, nil)

	// Iniciar simulando entorno corporativo sin privilegios de administrador (hasOSAdminPrivileges = false)
	if err := vpn.Start(false); err != nil {
		t.Fatalf("el inicio de la VPN corporativa en modo zero-admin falló: %v", err)
	}
	defer vpn.Stop()

	// Verificar que conmutó a ModeUserspaceProxy
	if vpn.ActiveMode != ModeUserspaceProxy {
		t.Errorf("modo esperado %v, obtenido %v", ModeUserspaceProxy, vpn.ActiveMode)
	}

	stats := vpn.Stats()
	if stats["is_running"] != true {
		t.Errorf("la VPN debe reportar is_running = true")
	}

	ipv4, ipv6 := vpn.VirtualIPs()
	if ipv4 == nil || ipv6 == nil {
		t.Errorf("las IPs virtuales deben ser no nulas: ipv4=%v, ipv6=%v", ipv4, ipv6)
	}
}

func TestCorporateVPNTLSDisguise(t *testing.T) {
	id, _ := l0.GenerateIdentity()
	cfg := DefaultCorporateVPNConfig()
	vpn := NewCorporateVPNManager(id, cfg, nil, nil)

	rawMeshPacket := []byte("PAQUETE_DE_MALLA_SOBERANA_1280B_CBOR_ENCRYPTED")
	wrapped := vpn.WrapTLSDisguise(rawMeshPacket)

	// Verificar cabecera RFC 8446 TLS 1.3 ApplicationData
	if len(wrapped) != 5+len(rawMeshPacket) {
		t.Fatalf("longitud de trama TLS incorrecta: %d (esperada %d)", len(wrapped), 5+len(rawMeshPacket))
	}
	if wrapped[0] != 0x17 || wrapped[1] != 0x03 || wrapped[2] != 0x03 {
		t.Fatalf("cabecera TLS inválida: %x %x %x", wrapped[0], wrapped[1], wrapped[2])
	}

	// Desenvolver
	unwrapped, err := vpn.UnwrapTLSDisguise(wrapped)
	if err != nil {
		t.Fatalf("error desempaquetando trama TLS: %v", err)
	}

	if !bytes.Equal(unwrapped, rawMeshPacket) {
		t.Errorf("el paquete desenvuelto no coincide con el original")
	}
}

func TestCorporateVPNZTNAMicrosegmentation(t *testing.T) {
	id, _ := l0.GenerateIdentity()
	cfg := DefaultCorporateVPNConfig()
	authorizedDID := "did:ipvn7:0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"
	unauthorizedDID := "did:ipvn7:9999999999999999999999999999999999999999999999999999999999999999"

	cfg.AllowedEnterpriseDIDs = []string{authorizedDID}
	vpn := NewCorporateVPNManager(id, cfg, nil, nil)

	// El par autorizado debe pasar
	if !vpn.CheckZTNAAccess(authorizedDID, 443) {
		t.Errorf("DID autorizado fue denegado por ZTNA")
	}

	// El par no autorizado debe ser bloqueado por Default-Deny
	if vpn.CheckZTNAAccess(unauthorizedDID, 443) {
		t.Errorf("DID no autorizado fue permitido indebidamente por ZTNA")
	}

	// Autorizar dinámicamente
	vpn.AuthorizeEnterpriseDID(unauthorizedDID)
	if !vpn.CheckZTNAAccess(unauthorizedDID, 443) {
		t.Errorf("DID recién autorizado debería ser permitido")
	}
}

func TestCorporateVPNEgressGateway(t *testing.T) {
	id, _ := l0.GenerateIdentity()
	cfg := DefaultCorporateVPNConfig()
	vpn := NewCorporateVPNManager(id, cfg, nil, nil)

	gatewayDID := "did:ipvn7:frankfurt_enterprise_gw_01"
	vpn.SetEgressGateway(gatewayDID)

	stats := vpn.Stats()
	if stats["egress_gateway"] != gatewayDID {
		t.Errorf("egress gateway esperado %s, obtenido %v", gatewayDID, stats["egress_gateway"])
	}
}
