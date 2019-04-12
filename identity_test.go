package forest_test

import (
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"golang.org/x/crypto/openpgp"
)

func TestIdentityValidatesSelf(t *testing.T) {
	builder := forest.IdentityBuilder{}
	privkey, err := openpgp.NewEntity("forest-test", "comment", "email@email.io", nil)
	if err != nil {
		t.Skip("Failed to create private key", err)
	}
	username, err := forest.NewQualifiedContent(forest.ContentTypeUTF8String, []byte("Test Name"))
	if err != nil {
		t.Skip("Failed to qualify username", err)
	}
	metadata, err := forest.NewQualifiedContent(forest.ContentTypeUTF8String, []byte{})
	if err != nil {
		t.Skip("Failed to qualify metadata", err)
	}
	identity, err := builder.New(privkey, username, metadata)
	if err != nil {
		t.Error("Failed to create Identity with valid parameters", err)
	}
	if correct, err := identity.ValidateID(); err != nil || !correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := identity.ValidateSignatureFor(identity); err != nil || !correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}
