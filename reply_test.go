package forest_test

import (
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"golang.org/x/crypto/openpgp"
)

func MakeReplyOrSkip(t *testing.T) (*forest.Identity, *openpgp.Entity, *forest.Community, *forest.Conversation, *forest.Reply) {
	identity, privkey, community, conversation := MakeConversationOrSkip(t)
	content, err := forest.NewQualifiedContent(forest.ContentTypeUTF8String, []byte("Test content"))
	if err != nil {
		t.Skip("Failed to qualify content", err)
	}
	metadata, err := forest.NewQualifiedContent(forest.ContentTypeUTF8String, []byte{})
	if err != nil {
		t.Skip("Failed to qualify metadata", err)
	}
	reply, err := forest.NewReply(identity, privkey, conversation, content, metadata)
	if err != nil {
		t.Error("Failed to create Conversation with valid parameters", err)
	}
	return identity, privkey, community, conversation, reply
}

func TestReplyValidatesSelf(t *testing.T) {
	identity, _, _, _, reply := MakeReplyOrSkip(t)
	if correct, err := forest.ValidateID(reply, *reply.ID()); err != nil || !correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := forest.ValidateSignature(reply, identity); err != nil || !correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}

func TestReplyValidationFailsWhenTampered(t *testing.T) {
	identity, _, _, _, reply := MakeReplyOrSkip(t)
	identity.Name.Value = forest.Value([]byte("whatever"))
	if correct, err := forest.ValidateID(reply, *reply.ID()); err != nil || !correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := forest.ValidateSignature(reply, identity); err != nil || !correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}

func TestReplySerialize(t *testing.T) {
	_, _, _, _, reply := MakeReplyOrSkip(t)
	buf, err := reply.MarshalBinary()
	if err != nil {
		t.Error("Failed to serialize identity", err)
	}
	c2, err := forest.UnmarshalReply(buf)
	if err != nil {
		t.Error("Failed to deserialize identity", err)
	}
	if !reply.Equals(c2) {
		t.Errorf("Deserialized identity should be the same as what went in, expected %v, got %v", reply, c2)
	}
}
