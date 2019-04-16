package forest_test

import (
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"golang.org/x/crypto/openpgp"
)

func MakeIdentityOrSkip(t *testing.T) *forest.Identity {
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
	return identity
}

func TestIdentityValidatesSelf(t *testing.T) {
	identity := MakeIdentityOrSkip(t)
	if correct, err := forest.ValidateID(identity, *identity.ID()); err != nil || !correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := forest.ValidateSignature(identity, identity); err != nil || !correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}

func TestIdentityValidationFailsWhenTampered(t *testing.T) {
	identity := MakeIdentityOrSkip(t)
	identity.Name.Value = forest.Value([]byte("whatever"))
	if correct, err := forest.ValidateID(identity, *identity.ID()); err == nil && correct {
		t.Error("ID validation succeeded on modified node", err)
	}
	if correct, err := forest.ValidateSignature(identity, identity); err == nil && correct {
		t.Error("Signature validation succeeded on modified node", err)
	}
}
