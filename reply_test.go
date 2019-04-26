package forest_test

import (
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"golang.org/x/crypto/openpgp"
)

func MakeReplyOrSkip(t *testing.T) (*forest.Identity, *openpgp.Entity, *forest.Community, *forest.Conversation, *forest.Reply) {
	identity, privkey, community, conversation := MakeConversationOrSkip(t)
	content := QualifiedContentOrSkip(t, forest.ContentTypeUTF8String, []byte("test content"))
	metadata := QualifiedContentOrSkip(t, forest.ContentTypeUTF8String, []byte{})
	reply, err := forest.NewReply(identity, privkey, conversation, content, metadata)
	if err != nil {
		t.Error("Failed to create Conversation with valid parameters", err)
	}
	return identity, privkey, community, conversation, reply
}

func TestReplyValidatesSelf(t *testing.T) {
	identity, _, _, _, reply := MakeReplyOrSkip(t)
	validateReply(t, identity, reply)
}

func failToValidateReply(t *testing.T, author *forest.Identity, reply *forest.Reply) {
	if correct, err := forest.ValidateID(reply, *reply.ID()); err == nil && correct {
		t.Error("ID validation succeded on modified node", err)
	}
	if correct, err := forest.ValidateSignature(reply, author); err == nil && correct {
		t.Error("Signature validation succeded on modified node", err)
	}
}

func validateReply(t *testing.T, author *forest.Identity, reply *forest.Reply) {
	if correct, err := forest.ValidateID(reply, *reply.ID()); err != nil || !correct {
		t.Error("ID validation failed on unmodified node", err)
	}
	if correct, err := forest.ValidateSignature(reply, author); err != nil || !correct {
		t.Error("Signature validation failed on unmodified node", err)
	}
}

func TestReplyValidationFailsWhenTampered(t *testing.T) {
	identity, _, _, _, reply := MakeReplyOrSkip(t)
	identity.Name.Value = forest.Value([]byte("whatever"))
	failToValidateReply(t, identity, reply)
}

func ensureSerializes(t *testing.T, reply *forest.Reply) {
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

func TestReplySerializes(t *testing.T) {
	_, _, _, _, reply := MakeReplyOrSkip(t)
	ensureSerializes(t, reply)
}

func TestReplyToReplyValidates(t *testing.T) {
	identity, privkey, _, _, reply := MakeReplyOrSkip(t)
	content := QualifiedContentOrSkip(t, forest.ContentTypeUTF8String, []byte("hello"))
	metadata := QualifiedContentOrSkip(t, forest.ContentTypeUTF8String, []byte(""))
	r2, err := forest.NewReply(identity, privkey, reply, content, metadata)
	if err != nil {
		t.Error("Failed to create reply to existing reply", err)
	}
	validateReply(t, identity, r2)
}

func TestReplyToReplyFailsWhenTampered(t *testing.T) {
	identity, privkey, _, _, reply := MakeReplyOrSkip(t)
	content := QualifiedContentOrSkip(t, forest.ContentTypeUTF8String, []byte("hello"))
	metadata := QualifiedContentOrSkip(t, forest.ContentTypeUTF8String, []byte(""))
	r2, err := forest.NewReply(identity, privkey, reply, content, metadata)
	if err != nil {
		t.Error("Failed to create reply to existing reply", err)
	}
	r2.Content.Value = forest.Value([]byte("else"))
	failToValidateReply(t, identity, r2)
}

func TestReplyToReplySerializes(t *testing.T) {
	identity, privkey, _, _, reply := MakeReplyOrSkip(t)
	content := QualifiedContentOrSkip(t, forest.ContentTypeUTF8String, []byte("hello"))
	metadata := QualifiedContentOrSkip(t, forest.ContentTypeUTF8String, []byte(""))
	r2, err := forest.NewReply(identity, privkey, reply, content, metadata)
	if err != nil {
		t.Error("Failed to create reply to existing reply", err)
	}
	ensureSerializes(t, r2)
}
