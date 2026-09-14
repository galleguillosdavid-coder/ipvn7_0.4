package l4

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

// PetnameRecord representa un alias mnemotécnico local en el sistema dDNS (Dimensión 2)
type PetnameRecord struct {
	Name        string    `json:"name"`         // Nombre memorable (ej. notebook.ipv7)
	DID         string    `json:"did"`          // Identificador soberano destino did:ipvn7:...
	VirtualIPv6 string    `json:"virtual_ipv6"` // Dirección IPv6 soberana asociada fd07::/64
	VirtualIPv4 string    `json:"virtual_ipv4"` // Dirección IPv4 sintética 10.7.0.0/16
	ContextRoot string    `json:"context_root"` // Raíz de contexto local (opcional)
	AssignedAt  time.Time `json:"assigned_at"`  // Fecha de asignación
	Comment     string    `json:"comment"`      // Nota humana explicativa
}

// PetnameResolver administra la libreta de direcciones descentralizada sin ICANN
type PetnameResolver struct {
	mu           sync.RWMutex
	records      map[string]*PetnameRecord // clave de búsqueda -> Registro
	reverseIndex map[string]string         // DID -> Nombre canónico
}

// NewPetnameResolver inicializa el motor de nombres locales dDNS
func NewPetnameResolver() *PetnameResolver {
	return &PetnameResolver{
		records:      make(map[string]*PetnameRecord),
		reverseIndex: make(map[string]string),
	}
}

// normalizeName estandariza el formato del nombre asegurando sufijo .ipv7
func normalizeName(name string) string {
	clean := strings.ToLower(strings.TrimSpace(name))
	if !strings.HasSuffix(clean, ".ipv7") {
		clean = clean + ".ipv7"
	}
	return clean
}

// buildLookupKey calcula la clave interna considerando el contexto si existe
func buildLookupKey(name, contextRoot string) string {
	cleanName := normalizeName(name)
	if contextRoot != "" {
		h := sha256.Sum256([]byte(contextRoot))
		prefix := hex.EncodeToString(h[:4])
		return fmt.Sprintf("%s.%s", prefix, strings.TrimPrefix(cleanName, prefix+"."))
	}
	return cleanName
}

// RegisterPetname registra o actualiza un alias mnemotécnico en la libreta local
func (pr *PetnameResolver) RegisterPetname(name, did, virtualV6, virtualV4, contextRoot, comment string) (*PetnameRecord, error) {
	if did == "" {
		return nil, errors.New("el DID de destino no puede estar vacío")
	}

	key := buildLookupKey(name, contextRoot)

	pr.mu.Lock()
	defer pr.mu.Unlock()

	rec := &PetnameRecord{
		Name:        key,
		DID:         did,
		VirtualIPv6: virtualV6,
		VirtualIPv4: virtualV4,
		ContextRoot: contextRoot,
		AssignedAt:  time.Now(),
		Comment:     comment,
	}

	pr.records[key] = rec
	pr.reverseIndex[did] = key
	return rec, nil
}

// Resolve busca un nombre en la libreta y retorna el registro correspondiente
func (pr *PetnameResolver) Resolve(name, contextRoot string) (*PetnameRecord, bool) {
	key := buildLookupKey(name, contextRoot)

	pr.mu.RLock()
	defer pr.mu.RUnlock()

	rec, ok := pr.records[key]
	return rec, ok
}

// ReverseLookup busca el nombre mnemotécnico asignado a un DID
func (pr *PetnameResolver) ReverseLookup(did string) (string, bool) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	name, ok := pr.reverseIndex[did]
	return name, ok
}

// DeletePetname elimina un registro de la libreta local
func (pr *PetnameResolver) DeletePetname(name, contextRoot string) bool {
	key := buildLookupKey(name, contextRoot)

	pr.mu.Lock()
	defer pr.mu.Unlock()

	rec, ok := pr.records[key]
	if !ok {
		return false
	}

	delete(pr.reverseIndex, rec.DID)
	delete(pr.records, key)
	return true
}

// ListRecords retorna una lista de todos los nombres registrados
func (pr *PetnameResolver) ListRecords() []*PetnameRecord {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	res := make([]*PetnameRecord, 0, len(pr.records))
	for _, rec := range pr.records {
		res = append(res, rec)
	}
	return res
}

// ExportHostsFormat genera el formato estándar del archivo /etc/hosts o hosts de Windows
func (pr *PetnameResolver) ExportHostsFormat() string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("# === ipvn7 Sovereign dDNS Local Table ===\n")
	for _, rec := range pr.records {
		if rec.VirtualIPv4 != "" {
			sb.WriteString(fmt.Sprintf("%-16s %s\n", rec.VirtualIPv4, rec.Name))
		}
		if rec.VirtualIPv6 != "" {
			sb.WriteString(fmt.Sprintf("%-40s %s\n", rec.VirtualIPv6, rec.Name))
		}
	}
	return sb.String()
}

// PetnameStats expone estadísticas del resolvedor
type PetnameStats struct {
	TotalNames      int `json:"total_names"`
	ContextualNames int `json:"contextual_names"`
}

// Stats retorna la instantánea para observabilidad
func (pr *PetnameResolver) Stats() PetnameStats {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	ctxCount := 0
	for _, r := range pr.records {
		if r.ContextRoot != "" {
			ctxCount++
		}
	}

	return PetnameStats{
		TotalNames:      len(pr.records),
		ContextualNames: ctxCount,
	}
}
