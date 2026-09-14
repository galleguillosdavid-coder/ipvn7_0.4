package l4

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// Constantes de estado del ciclo de vida de un servicio
const (
	ServiceStateDormant uint8 = 0 // En disco / no instanciado en RAM (Zero Footprint)
	ServiceStateActive  uint8 = 1 // En ejecución activa en RAM
	ServiceStatePruned  uint8 = 2 // Purgado tras inactividad
)

// ServiceDescriptor define un servicio de red bajo demanda
type ServiceDescriptor struct {
	Name        string    `json:"name"`        // ej: "chat_e2ee", "remote_desktop", "dag_store"
	Scope       uint8     `json:"scope"`       // ScopeLocal, ScopeMesh, ScopeGlobal
	State       uint8     `json:"state"`       // Dormant, Active, Pruned
	InstantiatedAt time.Time `json:"instantiated_at"`
	LastActiveAt   time.Time `json:"last_active_at"`
	ActiveSessions int       `json:"active_sessions"`
}

// ServiceLifecycleManager orquesta la carga perezosa y la poda activa
type ServiceLifecycleManager struct {
	mu          sync.RWMutex
	services    map[string]*ServiceDescriptor
	idleTimeout time.Duration
}

// NewServiceLifecycleManager inicializa el gestor de ciclo de vida con un timeout de inactividad
func NewServiceLifecycleManager(idleTimeout time.Duration) *ServiceLifecycleManager {
	return &ServiceLifecycleManager{
		services:    make(map[string]*ServiceDescriptor),
		idleTimeout: idleTimeout,
	}
}

// RegisterDormantService declara un servicio dormido (cero consumo de RAM hasta ser invocado)
func (sm *ServiceLifecycleManager) RegisterDormantService(name string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.services[name] = &ServiceDescriptor{
		Name:  name,
		State: ServiceStateDormant,
	}
}

// RequestService instancia un servicio en memoria RAM bajo demanda con un radio de alcance
func (sm *ServiceLifecycleManager) RequestService(name string, scope uint8) (*ServiceDescriptor, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	svc, found := sm.services[name]
	if !found {
		return nil, fmt.Errorf("servicio desconocido '%s'", name)
	}

	svc.Scope = scope
	svc.State = ServiceStateActive
	now := time.Now()
	svc.InstantiatedAt = now
	svc.LastActiveAt = now
	svc.ActiveSessions++

	return svc, nil
}

// TouchService actualiza el pulso de actividad del servicio para evitar su poda
func (sm *ServiceLifecycleManager) TouchService(name string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	svc, found := sm.services[name]
	if !found || svc.State != ServiceStateActive {
		return errors.New("servicio no activo")
	}
	svc.LastActiveAt = time.Now()
	return nil
}

// ReleaseService decrece el conteo de sesiones activas
func (sm *ServiceLifecycleManager) ReleaseService(name string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	svc, found := sm.services[name]
	if found && svc.ActiveSessions > 0 {
		svc.ActiveSessions--
	}
}

// PruneIdleServices destruye de la memoria RAM los servicios inactivos (Poda Activa / Zero Footprint)
func (sm *ServiceLifecycleManager) PruneIdleServices() []string {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	pruned := make([]string, 0)
	now := time.Now()

	for name, svc := range sm.services {
		if svc.State == ServiceStateActive && svc.ActiveSessions == 0 {
			if now.Sub(svc.LastActiveAt) > sm.idleTimeout {
				svc.State = ServiceStateDormant // Destrucción en RAM
				pruned = append(pruned, name)
			}
		}
	}
	return pruned
}

// GetActiveServices lista los servicios actualmente consumiendo memoria RAM
func (sm *ServiceLifecycleManager) GetActiveServices() []*ServiceDescriptor {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	res := make([]*ServiceDescriptor, 0)
	for _, svc := range sm.services {
		if svc.State == ServiceStateActive {
			sCopy := *svc
			res = append(res, &sCopy)
		}
	}
	return res
}
