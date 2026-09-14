package l1_test

import (
	"testing"

	"ipvn7/pkg/l0"
	"ipvn7/pkg/l1"
)

func TestSemanticCatalogPullDiscoveryAndIntersection(t *testing.T) {
	catalog := l1.NewSemanticCatalog()

	idNodeA, _ := l0.GenerateIdentity()
	idNodeB, _ := l0.GenerateIdentity()
	idNodeC, _ := l0.GenerateIdentity()

	// Nodo A: Computación en Chile
	_, err := catalog.SignAndRegister(idNodeA, []string{"geo/chile", "domain/compute", "gpu/cuda"}, "Nodo A Cómputo", 95.0)
	if err != nil {
		t.Fatalf("Registro Nodo A falló: %v", err)
	}

	// Nodo B: Almacenamiento en Chile
	_, err = catalog.SignAndRegister(idNodeB, []string{"geo/chile", "service/storage", "role/dtn"}, "Nodo B Almacén", 80.0)
	if err != nil {
		t.Fatalf("Registro Nodo B falló: %v", err)
	}

	// Nodo C: Computación en Argentina
	_, err = catalog.SignAndRegister(idNodeC, []string{"geo/argentina", "domain/compute"}, "Nodo C Cómputo", 75.0)
	if err != nil {
		t.Fatalf("Registro Nodo C falló: %v", err)
	}

	// 1. Consulta Pull: ¿Quién tiene "domain/compute"? -> Nodos A y C
	computeNodes := catalog.QueryPull([]string{"domain/compute"})
	if len(computeNodes) != 2 {
		t.Errorf("Se esperaban 2 nodos de cómputo, obtenidos: %d", len(computeNodes))
	}

	// 2. Consulta Pull: Intersección "domain/compute" + "geo/chile" -> Solo Nodo A
	chileCompute := catalog.QueryPull([]string{"domain/compute", "geo/chile"})
	if len(chileCompute) != 1 || chileCompute[0].DID != idNodeA.DID() {
		t.Errorf("Se esperaba únicamente el Nodo A en chileCompute, obtenido: %+v", chileCompute)
	}

	// 3. Consulta Pull: Tag inexistente -> 0 resultados
	unknown := catalog.QueryPull([]string{"service/quantum_teleport"})
	if len(unknown) != 0 {
		t.Errorf("Se esperaban 0 resultados para tag inexistente, obtenidos: %d", len(unknown))
	}
}
