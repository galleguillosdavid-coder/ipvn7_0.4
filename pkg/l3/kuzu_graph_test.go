package l3

import (
	"strings"
	"testing"
)

func TestKuzuGraphEnginePFOCypherAndAudit(t *testing.T) {
	kg := NewKuzuGraphEngine()

	// 1. Validar sembrado canónico PFO
	tree := kg.GetPFOTree()
	if tree["total_principles"].(int) != 5 {
		t.Fatalf("Esperados 5 axiomas canónicos, obtenidos %v", tree["total_principles"])
	}
	if tree["total_functions"].(int) < 10 {
		t.Fatalf("Esperadas al menos 10 funciones canónicas, obtenidas %v", tree["total_functions"])
	}

	// 2. Registrar pares y aristas XOR
	didAlice := "did:ipvn7:alice_node_ed25519"
	didBob := "did:ipvn7:bob_node_ed25519"

	kg.UpsertPeer(didAlice, "mldsa_pubkey_alice", "router", false)
	kg.UpsertPeer(didBob, "mldsa_pubkey_bob", "edge", false)
	kg.UpsertXORLink(didAlice, didBob, "0000ffff0000", 2.34, 0, "WiFi-5")

	// 3. Registrar observación empírica
	obsID := kg.AddObservation("L1_ZTNA_FIREWALL", "fp_ztna_100k_accept", 100000.0)
	if !strings.HasPrefix(obsID, "obs:") {
		t.Fatalf("ID de observación debe comenzar con obs:, obtenido %s", obsID)
	}

	// 4. Ejecutar consulta Cypher para topología
	resXOR, err := kg.ExecuteCypher("MATCH (p:Peer)-[l:XOR_LINK]->(m:Peer) RETURN p.did, l.degree, l.latency_ms, l.rf_band, m.did")
	if err != nil {
		t.Fatalf("Error en consulta Cypher XOR: %v", err)
	}
	if resXOR.RowCount != 1 {
		t.Fatalf("Esperada 1 arista en resultado, obtenidas %d", resXOR.RowCount)
	}
	if resXOR.Rows[0]["p.did"] != didAlice || resXOR.Rows[0]["m.did"] != didBob {
		t.Fatalf("DIDs de enlace no coinciden con los registrados")
	}

	// 5. Ejecutar consulta Cypher de axiomas
	resAx, err := kg.ExecuteCypher("MATCH (a:AxiomPrinciple) RETURN a.principle_id, a.statement")
	if err != nil {
		t.Fatalf("Error en consulta Cypher Axiomas: %v", err)
	}
	if resAx.RowCount != 5 {
		t.Fatalf("Esperados 5 axiomas en Cypher, obtenidos %d", resAx.RowCount)
	}

	// 6. Prueba de Auditoría Anti-Contradicciones
	// Caso legítimo:
	collides, msg := kg.AuditDirectiveContradiction("Añadir filtro de inspección con timeout de 10ms en L1")
	if collides {
		t.Fatalf("Directiva legítima no debería colisionar: %s", msg)
	}

	// Caso contradictorio 1: Violación Zero-PII
	collidesPII, msgPII := kg.AuditDirectiveContradiction("Guardar IP real en base de datos para análisis")
	if !collidesPII || !strings.Contains(msgPII, "AXIOM_ZERO_PII") {
		t.Fatalf("Debe detectar violación de Zero-PII: %s", msgPII)
	}

	// Caso contradictorio 2: Violación Core Freeze
	collidesCore, msgCore := kg.AuditDirectiveContradiction("Modify pkg/l0 to include extra headers")
	if !collidesCore || !strings.Contains(msgCore, "AXIOM_STRICT_CORE_FREEZE") {
		t.Fatalf("Debe detectar violación de Core Freeze: %s", msgCore)
	}
}
