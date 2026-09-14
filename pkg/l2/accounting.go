package l2

import (
	"sync"
	"time"
)

// ServiceTier representa los 4 niveles de servicio según la teoría de juegos Tit-for-Tat (Dimensión 5)
type ServiceTier int

const (
	TierPriority   ServiceTier = 0 // Ratio >= 1.0: Colaborador ejemplar (máxima prioridad y ancho de banda)
	TierNormal     ServiceTier = 1 // 0.5 <= Ratio < 1.0: Cooperación aceptable (tráfico estándar)
	TierBestEffort ServiceTier = 2 // 0.2 <= Ratio < 0.5: Contribución deficiente (cola secundaria)
	TierThrottled  ServiceTier = 3 // Ratio < 0.2: Nodo parásito/free-rider (estrangulamiento severo)
)

func (st ServiceTier) String() string {
	switch st {
	case TierPriority:
		return "PRIORITY"
	case TierNormal:
		return "NORMAL"
	case TierBestEffort:
		return "BEST_EFFORT"
	case TierThrottled:
		return "THROTTLED"
	default:
		return "UNKNOWN"
	}
}

// PeerAccounting registra la contabilidad estricta de tráfico de un par
type PeerAccounting struct {
	DID        string      `json:"did"`
	BytesTx    uint64      `json:"bytes_tx"` // Bytes que nosotros le hemos transmitido
	BytesRx    uint64      `json:"bytes_rx"` // Bytes que él nos ha retransmitido
	PacketsTx  uint64      `json:"packets_tx"`
	PacketsRx  uint64      `json:"packets_rx"`
	Ratio      float64     `json:"tit_for_tat_ratio"`
	Tier       ServiceTier `json:"tier"`
	LastActive time.Time   `json:"last_active"`
}

// TransitAccounting orquesta la economía algorítmica de reciprocidad
type TransitAccounting struct {
	mu               sync.RWMutex
	accounts         map[string]*PeerAccounting
	courtesyBytes    uint64 // Margen de cortesía inicial para bootstrap de nuevos pares (ej. 1 MB)
	courtesyPackets  uint64
	totalEnforced    uint64
	totalThrottled   uint64
}

// NewTransitAccounting inicializa el motor de reciprocidad Tit-for-Tat
func NewTransitAccounting() *TransitAccounting {
	return &TransitAccounting{
		accounts:        make(map[string]*PeerAccounting),
		courtesyBytes:   1048576, // 1 MB de cortesía inicial
		courtesyPackets: 1000,
	}
}

// getOrCreate obtiene o crea el registro contable de un DID
func (ta *TransitAccounting) getOrCreate(did string) *PeerAccounting {
	if acc, ok := ta.accounts[did]; ok {
		return acc
	}
	acc := &PeerAccounting{
		DID:        did,
		Tier:       TierNormal, // Inicia en nivel normal bajo margen de cortesía
		LastActive: time.Now(),
	}
	ta.accounts[did] = acc
	return acc
}

// RecordTx registra datagramas transmitidos hacia el par
func (ta *TransitAccounting) RecordTx(did string, bytes uint64) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	acc := ta.getOrCreate(did)
	acc.BytesTx += bytes
	acc.PacketsTx++
	acc.LastActive = time.Now()
	ta.recalculateTier(acc)
}

// RecordRx registra datagramas recibidos o retransmitidos por el par
func (ta *TransitAccounting) RecordRx(did string, bytes uint64) {
	ta.mu.Lock()
	defer ta.mu.Unlock()

	acc := ta.getOrCreate(did)
	acc.BytesRx += bytes
	acc.PacketsRx++
	acc.LastActive = time.Now()
	ta.recalculateTier(acc)
}

// recalculateTier recalcula el ratio Tit-for-Tat y el nivel de servicio
func (ta *TransitAccounting) recalculateTier(acc *PeerAccounting) {
	// 1. Margen de cortesía: si el par es nuevo y ha transmitido menos del margen, conceder TierNormal
	if acc.BytesTx <= ta.courtesyBytes && acc.PacketsTx <= ta.courtesyPackets {
		acc.Ratio = 1.0
		acc.Tier = TierNormal
		return
	}

	// 2. Cálculo de reciprocidad: Ratio = BytesRx / BytesTx
	if acc.BytesTx == 0 {
		acc.Ratio = 1.0
		acc.Tier = TierPriority
		return
	}

	acc.Ratio = float64(acc.BytesRx) / float64(acc.BytesTx)

	// 3. Asignación categórica de Tiers según la teoría de juegos
	if acc.Ratio >= 1.0 {
		acc.Tier = TierPriority
	} else if acc.Ratio >= 0.5 {
		acc.Tier = TierNormal
	} else if acc.Ratio >= 0.2 {
		acc.Tier = TierBestEffort
	} else {
		acc.Tier = TierThrottled
		ta.totalThrottled++
	}
	ta.totalEnforced++
}

// GetPeerTier retorna el nivel de servicio asignado al par
func (ta *TransitAccounting) GetPeerTier(did string) ServiceTier {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	if acc, ok := ta.accounts[did]; ok {
		return acc.Tier
	}
	return TierNormal // Nuevos pares gozan de cortesía
}

// GetPeerStats retorna una copia de la ficha contable de un par
func (ta *TransitAccounting) GetPeerStats(did string) (PeerAccounting, bool) {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	if acc, ok := ta.accounts[did]; ok {
		return *acc, true
	}
	return PeerAccounting{}, false
}

// GetAllAccounts retorna la lista completa de estados contables
func (ta *TransitAccounting) GetAllAccounts() []PeerAccounting {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	res := make([]PeerAccounting, 0, len(ta.accounts))
	for _, acc := range ta.accounts {
		res = append(res, *acc)
	}
	return res
}

// AccountingStats estructura el resumen para el dashboard web
type AccountingStats struct {
	TotalPeersTracked uint64 `json:"total_peers_tracked"`
	PriorityPeers     int    `json:"priority_peers"`
	NormalPeers       int    `json:"normal_peers"`
	BestEffortPeers   int    `json:"best_effort_peers"`
	ThrottledPeers    int    `json:"throttled_peers"`
	TotalThrottled    uint64 `json:"total_throttled_enforcements"`
}

// Stats genera la instantánea de observabilidad
func (ta *TransitAccounting) Stats() AccountingStats {
	ta.mu.RLock()
	defer ta.mu.RUnlock()

	var pri, norm, be, thr int
	for _, acc := range ta.accounts {
		switch acc.Tier {
		case TierPriority:
			pri++
		case TierNormal:
			norm++
		case TierBestEffort:
			be++
		case TierThrottled:
			thr++
		}
	}

	return AccountingStats{
		TotalPeersTracked: uint64(len(ta.accounts)),
		PriorityPeers:     pri,
		NormalPeers:       norm,
		BestEffortPeers:   be,
		ThrottledPeers:    thr,
		TotalThrottled:    ta.totalThrottled,
	}
}
