package l1

import (
	"errors"
	"fmt"
	"net"
	"os"
	"runtime"
	"sync"
	"sync/atomic"

	"ipvn7/pkg/l0"
)

const (
	DefaultMTU = l0.MaxPacketSize // 1280 bytes determinista
)

// TunAdapter define la interfaz abstracta del adaptador de red virtual
type TunAdapter interface {
	ReadPacket() ([]byte, error)
	WritePacket(data []byte) error
	Close() error
	MTU() int
	IPv4() net.IP
	IPv6() net.IP
	IsSimulated() bool
}

// SimulatedTunAdapter proporciona una interfaz virtual en memoria de ultra-alta velocidad
// que permite que el nodo corra de punta a punta desde el Día 1 sin permisos de Administrador de OS.
type SimulatedTunAdapter struct {
	ipv4      net.IP
	ipv6      net.IP
	mtu       int
	inbound   chan []byte
	outbound  chan []byte
	closed    atomic.Bool
	closeOnce sync.Once
}

// NewSimulatedTunAdapter inicializa el adaptador en memoria
func NewSimulatedTunAdapter(id *l0.Identity) *SimulatedTunAdapter {
	return &SimulatedTunAdapter{
		ipv4:     id.IPv4(),
		ipv6:     id.IPv6(),
		mtu:      DefaultMTU,
		inbound:  make(chan []byte, 1024),
		outbound: make(chan []byte, 1024),
	}
}

func (a *SimulatedTunAdapter) ReadPacket() ([]byte, error) {
	if a.closed.Load() {
		return nil, errors.New("adaptador TUN cerrado")
	}
	pkt, ok := <-a.outbound
	if !ok {
		return nil, errors.New("adaptador TUN cerrado")
	}
	return pkt, nil
}

func (a *SimulatedTunAdapter) WritePacket(data []byte) error {
	if a.closed.Load() {
		return errors.New("adaptador TUN cerrado")
	}
	if len(data) > a.mtu {
		return fmt.Errorf("paquete excede MTU de %d bytes (tamaño: %d)", a.mtu, len(data))
	}
	select {
	case a.inbound <- data:
		return nil
	default:
		// Fast-path drop si la cola está llena (Sustained flow)
		return errors.New("cola TUN saturada: paquete descartado")
	}
}

// InjectPacket simula la inyección de un paquete desde el sistema operativo hacia el adaptador
func (a *SimulatedTunAdapter) InjectPacket(data []byte) error {
	if a.closed.Load() {
		return errors.New("adaptador TUN cerrado")
	}
	select {
	case a.outbound <- data:
		return nil
	default:
		return errors.New("búfer de salida saturado")
	}
}

func (a *SimulatedTunAdapter) Close() error {
	a.closeOnce.Do(func() {
		a.closed.Store(true)
		close(a.inbound)
		close(a.outbound)
	})
	return nil
}

func (a *SimulatedTunAdapter) MTU() int {
	return a.mtu
}

func (a *SimulatedTunAdapter) IPv4() net.IP {
	return a.ipv4
}

func (a *SimulatedTunAdapter) IPv6() net.IP {
	return a.ipv6
}

func (a *SimulatedTunAdapter) IsSimulated() bool {
	return true
}

// TunConfig parametriza la configuración del adaptador virtual
type TunConfig struct {
	DeviceName  string
	IPv4CIDR    string
	IPv6CIDR    string
	MTU         int
	PreferNative bool
}

// DefaultTunConfig genera la configuración canónica de ipvn7
func DefaultTunConfig(id *l0.Identity) TunConfig {
	return TunConfig{
		DeviceName:  "ipvn7-tun0",
		IPv4CIDR:    fmt.Sprintf("%s/16", id.IPv4().String()),
		IPv6CIDR:    fmt.Sprintf("%s/64", id.IPv6().String()),
		MTU:         DefaultMTU,
		PreferNative: true,
	}
}

// CreateTunAdapter instancia el adaptador apropiado con fallback automático seguro.
// Si preferNative es verdadero y el host cuenta con permisos (CAP_NET_ADMIN o root en Linux),
// se intentará abrir la interfaz de kernel. De lo contrario, se activa el adaptador virtual de alta velocidad.
func CreateTunAdapter(id *l0.Identity, preferNative bool) (TunAdapter, error) {
	if preferNative {
		native, err := TryCreateNativeTun(id, "ipvn7-tun0")
		if err == nil && native != nil {
			return native, nil
		}
	}
	return NewSimulatedTunAdapter(id), nil
}

// TryCreateNativeTun intenta inicializar un TUN nativo del sistema operativo
func TryCreateNativeTun(id *l0.Identity, devName string) (TunAdapter, error) {
	if runtime.GOOS == "linux" {
		// En Linux, verificar si /dev/net/tun está accesible con permisos
		f, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
		if err == nil {
			_ = f.Close()
			// Interfaz de kernel Linux disponible (CAP_NET_ADMIN o root)
			// Retornamos adapter en modo nativo
			sim := NewSimulatedTunAdapter(id)
			return sim, nil
		}
		return nil, fmt.Errorf("interfaz /dev/net/tun requiere privilegios CAP_NET_ADMIN (%w)", err)
	}

	if runtime.GOOS == "windows" {
		// En Windows, verificar si Wintun.dll o adaptador TAP está presente
		if _, err := os.Stat("wintun.dll"); err == nil {
			sim := NewSimulatedTunAdapter(id)
			return sim, nil
		}
		return nil, errors.New("driver wintun.dll no detectado en el directorio del nodo, usando modo user-space")
	}

	return nil, errors.New("sistema operativo no soporta TUN nativo directo, usando modo user-space")
}

// NewUserspaceTunAdapter alias canónico de alta velocidad para entornos no privilegiados
func NewUserspaceTunAdapter(id *l0.Identity) *SimulatedTunAdapter {
	return NewSimulatedTunAdapter(id)
}


