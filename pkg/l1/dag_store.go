package l1

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"ipvn7/pkg/l0"
)

// DAGBlock representa un bloque inmutable direccionado por contenido (CID) en el DAG
type DAGBlock struct {
	CID       string    `json:"cid"`        // Identificador criptográfico único cid:ipvn7:<sha256>
	AuthorDID string    `json:"author_did"` // DID del emisor soberano
	TargetDID string    `json:"target_did"` // DID del receptor (o vacío para broadcast/malla)
	Parents   []string  `json:"parents"`    // CIDs de los bloques predecesores causales
	Timestamp time.Time `json:"timestamp"`  // Marca temporal de creación
	Data      []byte    `json:"data"`       // Carga útil de datos
	Signature []byte    `json:"signature"`  // Firma digital Ed25519 del emisor
}

// ComputeCID calcula el hash criptográfico canónico del bloque
func ComputeCID(authorDID string, parents []string, data []byte, ts time.Time) string {
	h := sha256.New()
	h.Write([]byte(authorDID))
	for _, p := range parents {
		h.Write([]byte(p))
	}
	h.Write([]byte(fmt.Sprintf("%d", ts.UnixNano())))
	h.Write(data)
	return fmt.Sprintf("cid:ipvn7:%s", hex.EncodeToString(h.Sum(nil)))
}

// DAGStore gestiona el almacenamiento asíncrono y las colas DTN Store-and-Forward (Dimensión 4)
type DAGStore struct {
	mu           sync.RWMutex
	identity     *l0.Identity
	blocks       map[string]*DAGBlock   // CID -> Bloque
	pendingQueue map[string][]*DAGBlock // TargetDID -> Lista de bloques en cola para envío diferido
	heads        []string               // CIDs de las cabeceras activas del DAG

	// Métricas
	BlocksStored   uint64
	BundlesQueued  uint64
	BundlesDelivered uint64
}

// NewDAGStore inicializa el gestor de persistencia DAG
func NewDAGStore(id *l0.Identity) *DAGStore {
	return &DAGStore{
		identity:     id,
		blocks:       make(map[string]*DAGBlock),
		pendingQueue: make(map[string][]*DAGBlock),
		heads:        make([]string, 0),
	}
}

// PutBlock crea, firma y almacena un nuevo bloque en el DAG
func (ds *DAGStore) PutBlock(data []byte, parents []string, targetDID string) (*DAGBlock, error) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Si no se especifican padres, enlazamos a las cabeceras actuales del DAG
	if len(parents) == 0 && len(ds.heads) > 0 {
		parents = append(parents, ds.heads...)
	}

	now := time.Now()
	cid := ComputeCID(ds.identity.DID(), parents, data, now)

	// Firmar el CID con la identidad soberana Ed25519
	sig := ds.identity.Sign([]byte(cid))

	block := &DAGBlock{
		CID:       cid,
		AuthorDID: ds.identity.DID(),
		TargetDID: targetDID,
		Parents:   parents,
		Timestamp: now,
		Data:      data,
		Signature: sig,
	}

	ds.blocks[cid] = block
	ds.BlocksStored++

	// Actualizar cabeceras del DAG: este bloque se convierte en nueva cabeza
	ds.heads = []string{cid}

	// Si tiene un destinatario específico que no somos nosotros, lo encolamos para DTN
	if targetDID != "" && targetDID != ds.identity.DID() {
		ds.pendingQueue[targetDID] = append(ds.pendingQueue[targetDID], block)
		ds.BundlesQueued++
	}

	return block, nil
}

// IngestRemoteBlock valida e inserta un bloque recibido de un par remoto
func (ds *DAGStore) IngestRemoteBlock(block *DAGBlock) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Si ya existe, es idempotente
	if _, exists := ds.blocks[block.CID]; exists {
		return nil
	}

	// 1. Verificar integridad del CID
	expectedCID := ComputeCID(block.AuthorDID, block.Parents, block.Data, block.Timestamp)
	if block.CID != expectedCID {
		return errors.New("integridad violada: el CID no coincide con el hash del contenido")
	}

	// 2. Verificar firma digital Ed25519 del autor
	pubKey, err := l0.PublicKeyFromDID(block.AuthorDID)
	if err != nil {
		return fmt.Errorf("DID de autor inválido: %w", err)
	}

	if !l0.VerifySignature(pubKey, []byte(block.CID), block.Signature) {
		return errors.New("firma criptográfica Ed25519 inválida para el bloque DAG")
	}

	// 3. Almacenar bloque y registrar
	ds.blocks[block.CID] = block
	ds.BlocksStored++
	ds.heads = append(ds.heads, block.CID)

	// Si era un paquete encolado para nosotros, marcamos como entregado
	if block.TargetDID == ds.identity.DID() {
		ds.BundlesDelivered++
	}

	return nil
}

// GetBlock recupera un bloque por su CID
func (ds *DAGStore) GetBlock(cid string) (*DAGBlock, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	b, ok := ds.blocks[cid]
	return b, ok
}

// GetPendingQueue retorna los bloques pendientes de entrega para un par reconectado
func (ds *DAGStore) GetPendingQueue(targetDID string) []*DAGBlock {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	list, ok := ds.pendingQueue[targetDID]
	if !ok {
		return nil
	}
	res := make([]*DAGBlock, len(list))
	copy(res, list)
	return res
}

// ClearPending elimina los bloques de la cola tras confirmación de entrega
func (ds *DAGStore) ClearPending(targetDID string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	if list, ok := ds.pendingQueue[targetDID]; ok {
		ds.BundlesDelivered += uint64(len(list))
		delete(ds.pendingQueue, targetDID)
	}
}

// ListBlocks retorna un inventario de los últimos bloques almacenados
func (ds *DAGStore) ListBlocks(limit int) []*DAGBlock {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	res := make([]*DAGBlock, 0, len(ds.blocks))
	count := 0
	for _, b := range ds.blocks {
		res = append(res, b)
		count++
		if limit > 0 && count >= limit {
			break
		}
	}
	return res
}

// DAGStats resume las métricas del almacén DAG
type DAGStats struct {
	TotalBlocks      int    `json:"total_blocks"`
	PendingTargets   int    `json:"pending_targets"`
	BlocksStored     uint64 `json:"blocks_stored"`
	BundlesQueued    uint64 `json:"bundles_queued"`
	BundlesDelivered uint64 `json:"bundles_delivered"`
}

// Stats retorna la instantánea de métricas para la UI y telemetría
func (ds *DAGStore) Stats() DAGStats {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	return DAGStats{
		TotalBlocks:      len(ds.blocks),
		PendingTargets:   len(ds.pendingQueue),
		BlocksStored:     ds.BlocksStored,
		BundlesQueued:    ds.BundlesQueued,
		BundlesDelivered: ds.BundlesDelivered,
	}
}
