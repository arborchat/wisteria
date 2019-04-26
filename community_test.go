package forest_test

import (
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"golang.org/x/crypto/openpgp"
)

func MakeCommunityOrSkip(t *testing.T) (*forest.Identity, *openpgp.Entity, *forest.Community) {
	identity, privkey := MakeIdentityOrSkip(t)
	name, err := forest.NewQualifiedContent(forest.ContentTypeUTF8String, []byte("Test Name"))
	if err != nil {
		t.Skip("Failed to qualify username", err)
	}
	metadata, err := forest.NewQualifiedContent(forest.ContentTypeUTF8String, []byte{})
	if err != nil {
		t.Skip("Failed to qualify metadata", err)
	}
	community, err := forest.NewCommunity(identity, privkey, name, metadata)
	if err != nil {
		t.Error("Failed to create Community with valid parameters", err)
	}
	return identity, privkey, community
}

func TestCommunityValidatesSelf(t *testing.T) {
	identity, _, community := MakeCommunityOrSkip(t)
	if correct, err := forest.ValidateID(community, *community.ID()); err != nil || !correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := forest.ValidateSignature(community, identity); err != nil || !correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}

func TestCommunityValidationFailsWhenTampered(t *testing.T) {
	identity, _, community := MakeCommunityOrSkip(t)
	community.Name.Value = forest.Value([]byte("whatever"))
	if correct, err := forest.ValidateID(community, *community.ID()); err == nil && correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := forest.ValidateSignature(community, identity); err == nil && correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}

func TestCommunitySerialize(t *testing.T) {
	_, _, community := MakeCommunityOrSkip(t)
	buf, err := community.MarshalBinary()
	if err != nil {
		t.Error("Failed to serialize identity", err)
	}
	c2, err := forest.UnmarshalCommunity(buf)
	if err != nil {
		t.Error("Failed to deserialize identity", err)
	}
	if !community.Equals(c2) {
		t.Errorf("Deserialized identity should be the same as what went in, expected %v, got %v", community, c2)
	}
}
