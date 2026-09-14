package l4

import (
	"testing"

	"ipvn7/pkg/l0"
	"ipvn7/pkg/l1"
)

func TestChatManager_SendMessageAndHistory(t *testing.T) {
	id, err := l0.GenerateIdentity()
	if err != nil {
		t.Fatalf("Error generando identidad: %v", err)
	}

	dag := l1.NewDAGStore(id)
	cm := NewChatManager(id, dag)

	peerDID := "did:ipvn7:e821ef1a17f84e318f..."

	// 1. Verificar contactos iniciales
	contacts := cm.GetContacts()
	if len(contacts) < 2 {
		t.Errorf("Se esperaban al menos 2 contactos, obtenidos %d", len(contacts))
	}

	// 2. Enviar mensaje
	msg, err := cm.SendMessage(peerDID, "Mensaje de prueba soberano 1280B", "")
	if err != nil {
		t.Fatalf("Fallo enviando mensaje: %v", err)
	}

	if !msg.Delivered {
		t.Errorf("Se esperaba que el mensaje estuviese marcado como delivered")
	}

	if msg.Signature == "" {
		t.Errorf("El mensaje debe contener firma digital Ed25519")
	}

	// 3. Recuperar historial
	history := cm.GetHistory(peerDID)
	if len(history) < 3 { // 2 iniciales + 1 nuevo
		t.Errorf("Se esperaban al menos 3 mensajes en el historial, obtenidos %d", len(history))
	}
}
