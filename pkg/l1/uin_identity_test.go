package l1

import (
	"crypto/ed25519"
	"crypto/rand"
	"testing"
	"time"
)

func TestUINIdentityManager_RootAndBindingFlow(t *testing.T) {
	mgr, err := NewUINIdentityManager(IdentityModeHybrid)
	if err != nil {
		t.Fatalf("NewUINIdentityManager failed: %v", err)
	}

	rootID := mgr.RootID()
	if len(rootID) != 32 {
		t.Fatalf("Expected 32 bytes root_id, got %d", len(rootID))
	}

	entityDID := mgr.EntityDID()
	if len(entityDID) == 0 {
		t.Fatal("EntityDID should not be empty")
	}

	// Emitir un binding para un Agente de IA (Scope AIAgent)
	agentPub, _, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	var prevKey [16]byte
	rec, err := mgr.IssueBinding(agentPub, BindingAlgoEd25519, BindingScopeAIAgent, 24*time.Hour, prevKey)
	if err != nil {
		t.Fatalf("IssueBinding failed: %v", err)
	}

	if rec.Scope != BindingScopeAIAgent {
		t.Errorf("Expected scope %d, got %d", BindingScopeAIAgent, rec.Scope)
	}

	// Validar binding criptográficamente
	if !mgr.ValidateBinding(rec) {
		t.Fatal("BindingRecord must be valid immediately after issuance")
	}

	// Comprobar pasaporte
	passport := mgr.GetPassport()
	if len(passport.ActiveBindings) != 1 {
		t.Errorf("Expected 1 active binding in passport, got %d", len(passport.ActiveBindings))
	}

	// Revocar binding
	if err := mgr.RevokeBinding(rec.KeyID); err != nil {
		t.Fatalf("RevokeBinding failed: %v", err)
	}

	// Tras revocación, ValidateBinding debe fallar
	if mgr.ValidateBinding(rec) {
		t.Fatal("BindingRecord must NOT be valid after revocation")
	}

	passportAfter := mgr.GetPassport()
	if len(passportAfter.ActiveBindings) != 0 || len(passportAfter.RevokedBindings) != 1 {
		t.Errorf("Unexpected passport state after revocation: %+v", passportAfter)
	}
}
