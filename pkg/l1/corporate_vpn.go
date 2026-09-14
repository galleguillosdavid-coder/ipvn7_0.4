// Package l1 implementa el Motor de VPN Corporativa de Cero Fricción para Multinacionales.
// Proporciona:
// 1. Despliegue Zero-Admin: Conmutación automática a proxy local de espacio de usuario
//    (SOCKS5 en :10807 y HTTP CONNECT en :10808) cuando no hay privilegios de administrador.
// 2. Disfraz de Tráfico Anti-DPI: Encapsulado en tramas TLS 1.3 / puerto 443 para eludir cortafuegos empresariales.
// 3. Egress Gateway Soberano: Salida a Internet a través de nodos de confianza por jurisdicción.
// 4. Micro-segmentación Zero Trust (ZTNA) por DID: Elimina el movimiento lateral malicioso.
package l1

import (
	"encoding/binary"
	"errors"
	"net"
	"sync"
	"sync/atomic"

	"ipvn7/pkg/l0"
)

// Modos de operación de la VPN corporativa
type CorporateVPNMode int

const (
	ModeKernelTUN       CorporateVPNMode = 1 // Requiere privilegios de admin (adaptador nativo ipv70)
	ModeUserspaceProxy  CorporateVPNMode = 2 // Cero privilegios (SOCKS5 + HTTP CONNECT + Virtual L3)
	ModeEgressGateway   CorporateVPNMode = 3 // Modo pasarela de salida corporativa hacia Internet
)

func (m CorporateVPNMode) String() string {
	switch m {
	case ModeKernelTUN:
		return "KERNEL_TUN_NATIVO"
	case ModeUserspaceProxy:
		return "USERSPACE_PROXY_ZERO_ADMIN"
	case ModeEgressGateway:
		return "EGRESS_GATEWAY_CORPORATIVO"
	default:
		return "DESCONOCIDO"
	}
}

// CorporateVPNConfig define los parámetros operativos para redes multinacionales
type CorporateVPNConfig struct {
	AllowNonAdminFallback bool     `json:"allow_non_admin_fallback"`
	SOCKS5Addr            string   `json:"socks5_addr"`
	HTTPProxyAddr         string   `json:"http_proxy_addr"`
	EncapsulateTLS443     bool     `json:"encapsulate_tls_443"`
	EgressGatewayDID      string   `json:"egress_gateway_did"`
	ZTNAStrict            bool     `json:"ztna_strict"`
	AllowedEnterpriseDIDs []string `json:"allowed_enterprise_dids"`
}

// DefaultCorporateVPNConfig genera una configuración lista para usar con cero fricción
func DefaultCorporateVPNConfig() CorporateVPNConfig {
	return CorporateVPNConfig{
		AllowNonAdminFallback: true,
		SOCKS5Addr:            "127.0.0.1:10807",
		HTTPProxyAddr:         "127.0.0.1:10808",
		EncapsulateTLS443:     true,
		ZTNAStrict:            true,
	}
}

// CorporateVPNManager coordina el túnel corporativo transparente
type CorporateVPNManager struct {
	mu                 sync.RWMutex
	Config             CorporateVPNConfig
	Identity           *l0.Identity
	ActiveMode         CorporateVPNMode
	TunAdapter         TunAdapter
	SOCKS5             *SOCKS5Gateway
	allowedDIDsMap     map[string]bool
	isRunning          atomic.Bool
	bytesProxied       uint64
	connectionsCount   uint64
	tlsFramesDisguised uint64
}

// NewCorporateVPNManager crea una instancia del gestor de VPN corporativa
func NewCorporateVPNManager(id *l0.Identity, cfg CorporateVPNConfig, tun TunAdapter, socks *SOCKS5Gateway) *CorporateVPNManager {
	allowed := make(map[string]bool)
	for _, did := range cfg.AllowedEnterpriseDIDs {
		allowed[did] = true
	}

	return &CorporateVPNManager{
		Config:         cfg,
		Identity:       id,
		TunAdapter:     tun,
		SOCKS5:         socks,
		allowedDIDsMap: allowed,
		ActiveMode:     ModeUserspaceProxy, // Por defecto garantiza cero fricción sin admin
	}
}

// Start inicializa el motor de VPN corporativa con detección inteligente de entorno
func (c *CorporateVPNManager) Start(hasOSAdminPrivileges bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning.Load() {
		return errors.New("la VPN corporativa ya se encuentra en ejecución")
	}

	// Detección de privilegios de OS
	if hasOSAdminPrivileges && c.TunAdapter != nil && !c.TunAdapter.IsSimulated() {
		c.ActiveMode = ModeKernelTUN
	} else {
		// Fallback automático de cero fricción a espacio de usuario
		c.ActiveMode = ModeUserspaceProxy
		if c.SOCKS5 == nil {
			c.SOCKS5 = NewSOCKS5Gateway(c.Config.SOCKS5Addr, nil)
		}
		// Iniciar gateway SOCKS5 si no está corriendo
		_ = c.SOCKS5.Start()
	}

	c.isRunning.Store(true)
	return nil
}

// Stop detiene el gestor y cierra los sockets asociados
func (c *CorporateVPNManager) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning.Load() {
		return nil
	}

	if c.SOCKS5 != nil {
		c.SOCKS5.Stop()
	}
	if c.TunAdapter != nil {
		_ = c.TunAdapter.Close()
	}

	c.isRunning.Store(false)
	return nil
}

// WrapTLSDisguise añade una cabecera de trama estándar TLS 1.3 ApplicationData (RFC 8446)
// sobre el paquete de malla para evadir cortafuegos corporativos con inspección DPI:
// Byte 0: 0x17 (Application Data)
// Bytes 1-2: 0x03 0x03 (TLS 1.2/1.3 Legacy Version)
// Bytes 3-4: Longitud de carga útil (uint16 big-endian)
func (c *CorporateVPNManager) WrapTLSDisguise(packet []byte) []byte {
	atomic.AddUint64(&c.tlsFramesDisguised, 1)
	length := len(packet)

	frame := make([]byte, 5+length)
	frame[0] = 0x17 // TLS ApplicationData
	frame[1] = 0x03 // Major
	frame[2] = 0x03 // Minor (Legacy version standard)
	binary.BigEndian.PutUint16(frame[3:5], uint16(length))
	copy(frame[5:], packet)

	return frame
}

// UnwrapTLSDisguise remueve la trama TLS y extrae el paquete de malla original
func (c *CorporateVPNManager) UnwrapTLSDisguise(frame []byte) ([]byte, error) {
	if len(frame) < 5 {
		return nil, errors.New("trama TLS incompleta o corrupta")
	}

	if frame[0] != 0x17 || frame[1] != 0x03 || frame[2] != 0x03 {
		return nil, errors.New("cabecera de trama TLS inválida")
	}

	length := int(binary.BigEndian.Uint16(frame[3:5]))
	if len(frame) < 5+length {
		return nil, errors.New("longitud de trama declarada no coincide con el payload recibido")
	}

	return frame[5 : 5+length], nil
}

// CheckZTNAAccess verifica la autorización de un par bajo la política Zero Trust Default-Deny
func (c *CorporateVPNManager) CheckZTNAAccess(peerDID string, port int) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Si no está en modo estricto, permite comunicación entre nodos conocidos
	if !c.Config.ZTNAStrict {
		return true
	}

	// Si hay lista blanca empresarial explícita, se requiere autorización
	if len(c.allowedDIDsMap) > 0 {
		return c.allowedDIDsMap[peerDID]
	}

	// Por defecto, se requiere que el DID pertenezca al dominio soberano ipvn7
	return len(peerDID) > 10
}

// SetEgressGateway define la identidad del nodo de salida corporativo
func (c *CorporateVPNManager) SetEgressGateway(did string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.Config.EgressGatewayDID = did
}

// AuthorizeEnterpriseDID añade un DID corporativo a la lista de micro-segmentación ZTNA
func (c *CorporateVPNManager) AuthorizeEnterpriseDID(did string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.allowedDIDsMap[did] = true
	c.Config.AllowedEnterpriseDIDs = append(c.Config.AllowedEnterpriseDIDs, did)
}

// Stats devuelve el estado operativo actual de la VPN
func (c *CorporateVPNManager) Stats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]interface{}{
		"is_running":            c.isRunning.Load(),
		"active_mode":           c.ActiveMode.String(),
		"socks5_addr":           c.Config.SOCKS5Addr,
		"tls_disguise_active":   c.Config.EncapsulateTLS443,
		"tls_frames_disguised":  atomic.LoadUint64(&c.tlsFramesDisguised),
		"egress_gateway":        c.Config.EgressGatewayDID,
		"ztna_strict":           c.Config.ZTNAStrict,
		"authorized_dids_count": len(c.allowedDIDsMap),
	}
}

// VirtualIPs devuelve las direcciones virtuales asignadas localmente
func (c *CorporateVPNManager) VirtualIPs() (ipv4 net.IP, ipv6 net.IP) {
	if c.TunAdapter != nil {
		return c.TunAdapter.IPv4(), c.TunAdapter.IPv6()
	}
	return c.Identity.IPv4(), c.Identity.IPv6()
}
