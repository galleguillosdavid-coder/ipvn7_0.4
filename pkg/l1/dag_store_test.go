package l1

import (
	"testing"

	"ipvn7/pkg/l0"
)

func TestDAGStoreAndDTN(t *testing.T) {
	idA, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("Error generando idA: %v", err)
	}
	idB, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("Error generando idB: %v", err)
	}

	storeA := NewDAGStore(idA)
	storeB := NewDAGStore(idB)

	// 1. Guardar bloque en storeA dirigido a nodo B (que está offline)
	payload := []byte("mensaje_asincrono_dtn_bloque_1")
	block1, err := storeA.PutBlock(payload, nil, idB.DID())
	if err != nil {
		t.Fatalf("Error al insertar bloque en storeA: %v", err)
	}

	if block1.AuthorDID != idA.DID() {
		t.Fatalf("Esperado autor %s, obtenido %s", idA.DID(), block1.AuthorDID)
	}

	// Verificar que está encolado en la cola DTN para nodo B
	pending := storeA.GetPendingQueue(idB.DID())
	if len(pending) != 1 || pending[0].CID != block1.CID {
		t.Fatalf("Esperado 1 paquete encolado en DTN, obtenido %d", len(pending))
	}

	// 2. Simular reconexión del nodo B: transferir el bloque a storeB
	err = storeB.IngestRemoteBlock(block1)
	if err != nil {
		t.Fatalf("Error al ingerir bloque en nodo B: %v", err)
	}

	// Verificar que nodo B puede leerlo por CID
	retrieved, found := storeB.GetBlock(block1.CID)
	if !found {
		t.Fatalf("Bloque no encontrado en storeB")
	}
	if string(retrieved.Data) != string(payload) {
		t.Fatalf("Datos corruptos: esperado %s, obtenido %s", string(payload), string(retrieved.Data))
	}

	// 3. Confirmar entrega y limpiar cola en nodo A
	storeA.ClearPending(idB.DID())
	if len(storeA.GetPendingQueue(idB.DID())) != 0 {
		t.Fatalf("La cola DTN debió quedar vacía tras entrega confirmada")
	}

	statsA := storeA.Stats()
	if statsA.BundlesDelivered != 1 {
		t.Fatalf("Esperado 1 bundle entregado, obtenido %d", statsA.BundlesDelivered)
	}
}
