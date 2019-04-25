package forest_test

import (
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
)

func MakeConversationOrSkip(t *testing.T) (*forest.Identity, *forest.Community, *forest.Conversation) {
	identity, privkey := MakeIdentityOrSkip(t)
	_, community := MakeCommunityOrSkip(t)
	content, err := forest.NewQualifiedContent(forest.ContentTypeUTF8String, []byte("Test content"))
	if err != nil {
		t.Skip("Failed to qualify content", err)
	}
	metadata, err := forest.NewQualifiedContent(forest.ContentTypeUTF8String, []byte{})
	if err != nil {
		t.Skip("Failed to qualify metadata", err)
	}
	conversation, err := forest.NewConversation(identity, privkey, community, content, metadata)
	if err != nil {
		t.Error("Failed to create Conversation with valid parameters", err)
	}
	return identity, community, conversation
}

func TestConversationValidatesSelf(t *testing.T) {
	identity, _, conversation := MakeConversationOrSkip(t)
	if correct, err := forest.ValidateID(conversation, *conversation.ID()); err != nil || !correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := forest.ValidateSignature(conversation, identity); err != nil || !correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}

func TestConversationValidationFailsWhenTampered(t *testing.T) {
	identity, _, conversation := MakeConversationOrSkip(t)
	identity.Name.Value = forest.Value([]byte("whatever"))
	if correct, err := forest.ValidateID(conversation, *conversation.ID()); err != nil || !correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := forest.ValidateSignature(conversation, identity); err != nil || !correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}

func TestConversationSerialize(t *testing.T) {
	_, _, conversation := MakeConversationOrSkip(t)
	buf, err := conversation.MarshalBinary()
	if err != nil {
		t.Error("Failed to serialize identity", err)
	}
	c2, err := forest.UnmarshalConversation(buf)
	if err != nil {
		t.Error("Failed to deserialize identity", err)
	}
	if !conversation.Equals(c2) {
		t.Errorf("Deserialized identity should be the same as what went in, expected %v, got %v", conversation, c2)
	}
}
