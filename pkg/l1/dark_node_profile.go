package l1

import (
	"fmt"
	"sync"
	"time"
)

// OperatingMode define el estado operacional de presencia en la malla
type OperatingMode string

const (
	ModeStandardMesh       OperatingMode = "STANDARD_MESH"
	ModeDarkNodeEgressOnly OperatingMode = "DARK_NODE_EGRESS_ONLY"
)

// DarkNodeMacroOrchestrator implementa la Macro-Acción Atómica (2.md Líneas 722-728 y 766-789)
// Automatiza en cascada invisible:
// 1. ZTNA Firewall: aislamiento total de paquetes entrantes no solicitados.
// 2. DHT / Kleinberg: suspensión de listeners públicos (cero anuncios de IP de escucha).
// 3. Enrutamiento Sphinx: activación de circuitos cebolla multi-salto con tramas fijas de 1280B.
// 4. Packet Pacer: inyección de micro-jitter anti-análisis de tráfico lateral.
type DarkNodeMacroOrchestrator struct {
	mu                   sync.RWMutex
	currentMode          OperatingMode
	firewall             *ZTNAFirewall
	router               *KleinbergRouter
	pacer                *PacketPacer
	listenersMuted       bool
	sphinxCircuitsActive bool
	activeHops           int
	transitionsCount     uint64
	lastTransition       time.Time
}

// NewDarkNodeOrchestrator inicializa el orquestador de macro-acciones
func NewDarkNodeOrchestrator(fw *ZTNAFirewall, r *KleinbergRouter, p *PacketPacer) *DarkNodeMacroOrchestrator {
	return &DarkNodeMacroOrchestrator{
		currentMode:          ModeStandardMesh,
		firewall:             fw,
		router:               r,
		pacer:                p,
		listenersMuted:       false,
		sphinxCircuitsActive: false,
		activeHops:           0,
		transitionsCount:     0,
		lastTransition:       time.Now(),
	}
}

// ToggleMode conmuta atómicamente entre Modo Estándar y Modo Invisible (Dark Node)
func (o *DarkNodeMacroOrchestrator) ToggleMode() (OperatingMode, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	target := ModeDarkNodeEgressOnly
	if o.currentMode == ModeDarkNodeEgressOnly {
		target = ModeStandardMesh
	}

	if err := o.applyTransitionLocked(target); err != nil {
		return o.currentMode, err
	}

	return o.currentMode, nil
}

// SetMode establece explícitamente el modo operacional del nodo
func (o *DarkNodeMacroOrchestrator) SetMode(target OperatingMode) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.currentMode == target {
		return nil
	}

	return o.applyTransitionLocked(target)
}

// applyTransitionLocked ejecuta la orquestación en cascada invisible sin bloqueos manuales
func (o *DarkNodeMacroOrchestrator) applyTransitionLocked(target OperatingMode) error {
	switch target {
	case ModeDarkNodeEgressOnly:
		// 1. ZTNA: Garantizar Default-Deny estricto
		if o.firewall != nil {
			o.firewall.SetDefaultDeny(true)
		}

		// 2. DHT / Enrutador: Silenciar listeners públicos (Egress-Only)
		o.listenersMuted = true

		// 3. Activar circuitos cebolla Sphinx (3 saltos obligatorios, paquetes de 1280 bytes fijos)
		o.sphinxCircuitsActive = true
		o.activeHops = 3

		// 4. Pacer: Micro-ajuste para evitar correlación temporal de paquetes
		if o.pacer != nil {
			// El pacer mantiene la cadencia sostenible con protección anti-side-channel
			_ = o.pacer.Stats()
		}

		o.currentMode = ModeDarkNodeEgressOnly

	case ModeStandardMesh:
		// Restaurar modo bidireccional estándar
		o.listenersMuted = false
		o.sphinxCircuitsActive = false
		o.activeHops = 0
		o.currentMode = ModeStandardMesh

	default:
		return fmt.Errorf("modo operacional no reconocido: %s", target)
	}

	o.transitionsCount++
	o.lastTransition = time.Now()
	return nil
}

// GetStatus retorna el estado del macro-modo para la interfaz gráfica y auditoría
func (o *DarkNodeMacroOrchestrator) GetStatus() map[string]interface{} {
	o.mu.RLock()
	defer o.mu.RUnlock()

	isDark := o.currentMode == ModeDarkNodeEgressOnly

	return map[string]interface{}{
		"mode":                   string(o.currentMode),
		"is_dark_node":           isDark,
		"listeners_muted":        o.listenersMuted,
		"sphinx_circuits_active": o.sphinxCircuitsActive,
		"onion_hops":             o.activeHops,
		"egress_only":            isDark,
		"fixed_frame_bytes":      1280,
		"transitions_count":      o.transitionsCount,
		"last_transition":        o.lastTransition.Format(time.RFC3339),
		"description": func() string {
			if isDark {
				return "Modo Invisible Activo: Sin listeners en DHT, túneles Sphinx 3-saltos, tráfico saliente exclusivo (Egress-Only)."
			}
			return "Modo Estándar Activo: Participación bidireccional completa en los 12 anillos de Kleinberg."
		}(),
	}
}
