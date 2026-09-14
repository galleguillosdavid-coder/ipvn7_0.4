package l0

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
)

// Identity representa un nodo soberano con par de claves Ed25519.
type Identity struct {
	PublicKey  ed25519.PublicKey  `json:"public_key"`
	PrivateKey ed25519.PrivateKey `json:"private_key"`
}

// GenerateIdentity crea una nueva identidad criptográfica soberana desde entropía de OS.
func GenerateIdentity() (*Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("error generando par de claves Ed25519: %w", err)
	}
	return &Identity{
		PublicKey:  pub,
		PrivateKey: priv,
	}, nil
}

// DID devuelve el identificador descentralizado soberano estandarizado.
func (id *Identity) DID() string {
	return fmt.Sprintf("did:ipvn7:%s", hex.EncodeToString(id.PublicKey))
}

// DIDFromPublicKey genera el DID para una clave pública dada.
func DIDFromPublicKey(pub ed25519.PublicKey) string {
	return fmt.Sprintf("did:ipvn7:%s", hex.EncodeToString(pub))
}

// PublicKeyFromDID decodifica un DID a su clave pública Ed25519.
func PublicKeyFromDID(did string) (ed25519.PublicKey, error) {
	parts := strings.Split(did, ":")
	if len(parts) != 3 || parts[0] != "did" || parts[1] != "ipvn7" {
		return nil, fmt.Errorf("formato de DID inválido: %s", did)
	}
	raw, err := hex.DecodeString(parts[2])
	if err != nil {
		return nil, fmt.Errorf("hexadecimal de clave pública inválido: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("tamaño de clave pública incorrecto: %d bytes (esperado %d)", len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}

// IPv6 deriva la dirección soberana fd07::/64 truncando el hash SHA-256 del DID.
func (id *Identity) IPv6() net.IP {
	hash := sha256.Sum256([]byte(id.DID()))
	ip := make(net.IP, 16)
	ip[0] = 0xfd
	ip[1] = 0x07
	copy(ip[8:16], hash[:8])
	return ip
}

// IPv4 deriva una dirección privada virtual 10.7.x.y determinista.
func (id *Identity) IPv4() net.IP {
	hash := sha256.Sum256([]byte(id.DID()))
	ip := make(net.IP, 4)
	ip[0] = 10
	ip[1] = 7
	ip[2] = hash[0]
	ip[3] = hash[1]
	// Evitar direcciones de red o broadcast
	if ip[3] == 0 {
		ip[3] = 1
	} else if ip[3] == 255 {
		ip[3] = 254
	}
	return ip
}

// Sign genera una firma digital Ed25519 sobre un mensaje.
func (id *Identity) Sign(message []byte) []byte {
	return ed25519.Sign(id.PrivateKey, message)
}

// VerifySignature comprueba la firma digital de un par con su clave pública.
func VerifySignature(pub ed25519.PublicKey, message, signature []byte) bool {
	return ed25519.Verify(pub, message, signature)
}

// SaveToFile almacena el par de claves en un archivo protegido.
func (id *Identity) SaveToFile(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("no se pudo crear directorio del keystore: %w", err)
	}

	data, err := json.MarshalIndent(id, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializando keystore: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}

// LoadFromFile carga una identidad existente desde un archivo protegido.
func LoadFromFile(path string) (*Identity, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error leyendo keystore: %w", err)
	}

	var id Identity
	if err := json.Unmarshal(data, &id); err != nil {
		return nil, fmt.Errorf("error deserializando keystore: %w", err)
	}

	return &id, nil
}
