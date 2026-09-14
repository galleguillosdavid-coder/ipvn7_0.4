package l4

import (
	"strings"
	"testing"
)

func TestPetnameResolver(t *testing.T) {
	resolver := NewPetnameResolver()

	didNotebook := "did:ipvn7:bda3fed80fbbb6e52ed2bd34f2dc87a63436f301f07e7748d72c5b4a1b5c0c24"
	v6 := "fd07::404:fd44:dce7:af26"
	v4 := "10.7.4.4"

	// 1. Registrar nombre simple
	rec, err := resolver.RegisterPetname("notebook", didNotebook, v6, v4, "", "Notebook personal")
	if err != nil {
		t.Fatalf("Error registrando petname: %v", err)
	}
	if rec.Name != "notebook.ipv7" {
		t.Fatalf("Esperado 'notebook.ipv7', obtenido %s", rec.Name)
	}

	// 2. Resolver nombre
	resolved, ok := resolver.Resolve("notebook", "")
	if !ok || resolved.DID != didNotebook {
		t.Fatalf("Fallo resolviendo 'notebook': %+v", resolved)
	}

	// 3. Resolución inversa por DID
	revName, ok := resolver.ReverseLookup(didNotebook)
	if !ok || revName != "notebook.ipv7" {
		t.Fatalf("Fallo en resolución inversa: %s", revName)
	}

	// 4. Registrar nombre contextual
	ctxRoot := "red_investigacion_afe"
	recCtx, err := resolver.RegisterPetname("cluster", didNotebook, v6, v4, ctxRoot, "Cluster de computo")
	if err != nil {
		t.Fatalf("Error registrando nombre contextual: %v", err)
	}
	if !strings.Contains(recCtx.Name, ".cluster.ipv7") {
		t.Fatalf("Esperado prefijo de contexto en %s", recCtx.Name)
	}

	// 5. Exportar formato hosts
	hostsTxt := resolver.ExportHostsFormat()
	if !strings.Contains(hostsTxt, "notebook.ipv7") || !strings.Contains(hostsTxt, v4) {
		t.Fatalf("Hosts export no contiene el mapeo esperado:\n%s", hostsTxt)
	}
}
