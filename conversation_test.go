package forest_test

import (
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"golang.org/x/crypto/openpgp"
)

func MakeConversationOrSkip(t *testing.T) (*forest.Identity, *openpgp.Entity, *forest.Community, *forest.Conversation) {
	identity, privkey, community := MakeCommunityOrSkip(t)
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("Test content"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte{})
	conversation, err := forest.As(identity, privkey).NewConversation(community, content, metadata)
	if err != nil {
		t.Error("Failed to create Conversation with valid parameters", err)
	}
	return identity, privkey, community, conversation
}

func TestConversationValidatesSelf(t *testing.T) {
	identity, _, _, conversation := MakeConversationOrSkip(t)
	if correct, err := forest.ValidateID(conversation, *conversation.ID()); err != nil || !correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := forest.ValidateSignature(conversation, identity); err != nil || !correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}

func TestConversationValidationFailsWhenTampered(t *testing.T) {
	identity, _, _, conversation := MakeConversationOrSkip(t)
	conversation.Content.Value = fields.Value([]byte("whatever"))
	if correct, err := forest.ValidateID(conversation, *conversation.ID()); err == nil && correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := forest.ValidateSignature(conversation, identity); err == nil && correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}

func TestConversationSerialize(t *testing.T) {
	_, _, _, conversation := MakeConversationOrSkip(t)
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
