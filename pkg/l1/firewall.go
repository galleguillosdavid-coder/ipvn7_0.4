package l1

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Decision representa el veredicto del motor de cortafuegos ZTNA
type Decision int

const (
	DecisionAccept Decision = iota
	DecisionDropDefaultDeny
	DecisionDropACL
	DecisionDropRateLimit
)

func (d Decision) String() string {
	switch d {
	case DecisionAccept:
		return "ACCEPT"
	case DecisionDropDefaultDeny:
		return "DROP_DEFAULT_DENY"
	case DecisionDropACL:
		return "DROP_ACL"
	case DecisionDropRateLimit:
		return "DROP_RATE_LIMIT"
	default:
		return "UNKNOWN"
	}
}

// DIDPolicy define los permisos granulares concedidos a una identidad soberana
type DIDPolicy struct {
	DID           string    `json:"did"`
	AllowInbound  bool      `json:"allow_inbound"`
	AllowOutbound bool      `json:"allow_outbound"`
	AllowRelay    bool      `json:"allow_relay"`
	AllowedPorts  []uint16  `json:"allowed_ports"` // Si está vacío, permite todos los puertos virtuales autorizados
	CreatedAt     time.Time `json:"created_at"`
	ExpiresAt     time.Time `json:"expires_at"` // Zero time significa sin expiración
}

// ZTNAFirewall implementa el cortafuegos de micro-segmentación Zero Trust (Dimensión 1)
type ZTNAFirewall struct {
	mu          sync.RWMutex
	defaultDeny bool
	policies    map[string]*DIDPolicy

	// Contadores atómicos para observabilidad en tiempo real (L2)
	PacketsEvaluated uint64
	PacketsAccepted  uint64
	PacketsDroppedDD uint64
	PacketsDroppedACL uint64
}

// NewZTNAFirewall inicializa el firewall con la política estricta Default-Deny activada
func NewZTNAFirewall(defaultDeny bool) *ZTNAFirewall {
	return &ZTNAFirewall{
		defaultDeny: defaultDeny,
		policies:    make(map[string]*DIDPolicy),
	}
}

// SetDefaultDeny conmuta la política global de denegación por defecto
func (fw *ZTNAFirewall) SetDefaultDeny(enabled bool) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.defaultDeny = enabled
}

// IsDefaultDeny consulta si la denegación por defecto está activa
func (fw *ZTNAFirewall) IsDefaultDeny() bool {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	return fw.defaultDeny
}

// AuthorizeDID otorga permisos explícitos a un DID en la libreta de políticas
func (fw *ZTNAFirewall) AuthorizeDID(policy *DIDPolicy) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	if policy.CreatedAt.IsZero() {
		policy.CreatedAt = time.Now()
	}
	fw.policies[policy.DID] = policy
}

// RevokeDID revoca cualquier acceso previo concedido a un DID
func (fw *ZTNAFirewall) RevokeDID(did string) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	delete(fw.policies, did)
}

// GetPolicy consulta la política asignada a un DID
func (fw *ZTNAFirewall) GetPolicy(did string) (*DIDPolicy, bool) {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	p, ok := fw.policies[did]
	return p, ok
}

// GetAllPolicies retorna una copia de todas las políticas activas
func (fw *ZTNAFirewall) GetAllPolicies() []*DIDPolicy {
	fw.mu.RLock()
	defer fw.mu.RUnlock()
	res := make([]*DIDPolicy, 0, len(fw.policies))
	for _, p := range fw.policies {
		res = append(res, p)
	}
	return res
}

// EvaluateInbound dictamina si un datagrama entrante de srcDID hacia dstPort debe aceptarse
func (fw *ZTNAFirewall) EvaluateInbound(srcDID string, dstPort uint16) (Decision, string) {
	atomic.AddUint64(&fw.PacketsEvaluated, 1)

	fw.mu.RLock()
	defer fw.mu.RUnlock()

	policy, exists := fw.policies[srcDID]

	// 1. Verificación contra política Default-Deny
	if !exists {
		if fw.defaultDeny {
			atomic.AddUint64(&fw.PacketsDroppedDD, 1)
			return DecisionDropDefaultDeny, fmt.Sprintf("DID %s no figura en la lista de identidades autorizadas (Default-Deny)", srcDID)
		}
		atomic.AddUint64(&fw.PacketsAccepted, 1)
		return DecisionAccept, "Tráfico aceptado por modo permisivo"
	}

	// 2. Verificar expiración temporal de la política
	if !policy.ExpiresAt.IsZero() && time.Now().After(policy.ExpiresAt) {
		atomic.AddUint64(&fw.PacketsDroppedACL, 1)
		return DecisionDropACL, fmt.Sprintf("La política del DID %s ha expirado", srcDID)
	}

	// 3. Verificar si el tráfico entrante está habilitado
	if !policy.AllowInbound {
		atomic.AddUint64(&fw.PacketsDroppedACL, 1)
		return DecisionDropACL, fmt.Sprintf("Tráfico entrante denegado explícitamente para DID %s", srcDID)
	}

	// 4. Verificar restricción de puertos si existe
	if len(policy.AllowedPorts) > 0 {
		portAllowed := false
		for _, p := range policy.AllowedPorts {
			if p == dstPort {
				portAllowed = true
				break
			}
		}
		if !portAllowed {
			atomic.AddUint64(&fw.PacketsDroppedACL, 1)
			return DecisionDropACL, fmt.Sprintf("Puerto virtual %d no autorizado para DID %s", dstPort, srcDID)
		}
	}

	atomic.AddUint64(&fw.PacketsAccepted, 1)
	return DecisionAccept, "Datagrama validado por política ZTNA"
}

// FirewallStats estructura las métricas de rendimiento y auditoría del firewall
type FirewallStats struct {
	DefaultDeny       bool   `json:"default_deny"`
	ActivePolicies    int    `json:"active_policies"`
	PacketsEvaluated  uint64 `json:"packets_evaluated"`
	PacketsAccepted   uint64 `json:"packets_accepted"`
	PacketsDroppedDD  uint64 `json:"packets_dropped_default_deny"`
	PacketsDroppedACL uint64 `json:"packets_dropped_acl"`
}

// Stats genera una instantánea atómica para el dashboard L4 y la telemetría L2
func (fw *ZTNAFirewall) Stats() FirewallStats {
	fw.mu.RLock()
	dd := fw.defaultDeny
	count := len(fw.policies)
	fw.mu.RUnlock()

	return FirewallStats{
		DefaultDeny:       dd,
		ActivePolicies:    count,
		PacketsEvaluated:  atomic.LoadUint64(&fw.PacketsEvaluated),
		PacketsAccepted:   atomic.LoadUint64(&fw.PacketsAccepted),
		PacketsDroppedDD:  atomic.LoadUint64(&fw.PacketsDroppedDD),
		PacketsDroppedACL: atomic.LoadUint64(&fw.PacketsDroppedACL),
	}
}
