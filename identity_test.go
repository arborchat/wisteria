package forest_test

import (
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"golang.org/x/crypto/openpgp"
)

func MakeIdentityOrSkip(t *testing.T) (*forest.Identity, *openpgp.Entity) {
	builder := forest.IdentityBuilder{}
	privkey, err := openpgp.NewEntity("forest-test", "comment", "email@email.io", nil)
	if err != nil {
		t.Skip("Failed to create private key", err)
	}
	username, err := fields.NewQualifiedContent(fields.ContentTypeUTF8String, []byte("Test Name"))
	if err != nil {
		t.Skip("Failed to qualify username", err)
	}
	metadata, err := fields.NewQualifiedContent(fields.ContentTypeUTF8String, []byte{})
	if err != nil {
		t.Skip("Failed to qualify metadata", err)
	}
	identity, err := builder.New(privkey, username, metadata)
	if err != nil {
		t.Error("Failed to create Identity with valid parameters", err)
	}
	return identity, privkey
}

func TestIdentityValidatesSelf(t *testing.T) {
	identity, _ := MakeIdentityOrSkip(t)
	if correct, err := forest.ValidateID(identity, *identity.ID()); err != nil || !correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := forest.ValidateSignature(identity, identity); err != nil || !correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}

func TestIdentityValidationFailsWhenTampered(t *testing.T) {
	identity, _ := MakeIdentityOrSkip(t)
	identity.Name.Value = fields.Value([]byte("whatever"))
	if correct, err := forest.ValidateID(identity, *identity.ID()); err == nil && correct {
		t.Error("ID validation succeeded on modified node", err)
	}
	if correct, err := forest.ValidateSignature(identity, identity); err == nil && correct {
		t.Error("Signature validation succeeded on modified node", err)
	}
}

func TestIdentitySerialize(t *testing.T) {
	identity, _ := MakeIdentityOrSkip(t)
	buf, err := identity.MarshalBinary()
	if err != nil {
		t.Error("Failed to serialize identity", err)
	}
	id2, err := forest.UnmarshalIdentity(buf)
	if err != nil {
		t.Error("Failed to deserialize identity", err)
	}
	if !identity.Equals(id2) {
		t.Errorf("Deserialized identity should be the same as what went in, expected %v, got %v", identity, id2)
	}
}
