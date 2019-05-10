package forest_test

import (
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"golang.org/x/crypto/openpgp"
)

func TestNewReply(t *testing.T) {
	identity, privkey, community, conversation := MakeConversationOrSkip(t)
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("test content"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte{})
	reply, err := forest.As(identity, privkey).NewReply(conversation, content, metadata)
	if err != nil {
		t.Error("Failed to create Conversation with valid parameters", err)
	}
	if !reply.Parent.Equals(conversation.ID()) {
		t.Error("Reply's parent is not parent conversation")
	} else if !reply.ConversationID.Equals(&reply.Parent) {
		t.Error("Reply's conversation is not parent conversation")
	} else if !reply.CommunityID.Equals(community.ID()) {
		t.Error("Reply's community is not owning community")
	}
}

func TestNewReplyToReply(t *testing.T) {
	identity, privkey, community, conversation, reply := MakeReplyOrSkip(t)
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("other test content"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte{})
	reply2, err := forest.As(identity, privkey).NewReply(reply, content, metadata)
	if err != nil {
		t.Error("Failed to create Conversation with valid parameters", err)
	}
	if !reply2.Parent.Equals(reply.ID()) {
		t.Error("Reply's parent is not parent conversation")
	} else if !reply2.ConversationID.Equals(conversation.ID()) {
		t.Error("Reply's conversation is not parent conversation")
	} else if !reply2.CommunityID.Equals(community.ID()) {
		t.Error("Reply's community is not owning community")
	}
}

func MakeReplyOrSkip(t *testing.T) (*forest.Identity, *openpgp.Entity, *forest.Community, *forest.Conversation, *forest.Reply) {
	identity, privkey, community, conversation := MakeConversationOrSkip(t)
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("test content"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte{})
	reply, err := forest.As(identity, privkey).NewReply(conversation, content, metadata)
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
	reply.Content.Value = fields.Value([]byte("whatever"))
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
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("hello"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte(""))
	r2, err := forest.As(identity, privkey).NewReply(reply, content, metadata)
	if err != nil {
		t.Error("Failed to create reply to existing reply", err)
	}
	validateReply(t, identity, r2)
}

func TestReplyToReplyFailsWhenTampered(t *testing.T) {
	identity, privkey, _, _, reply := MakeReplyOrSkip(t)
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("hello"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte(""))
	r2, err := forest.As(identity, privkey).NewReply(reply, content, metadata)
	if err != nil {
		t.Error("Failed to create reply to existing reply", err)
	}
	r2.Content.Value = fields.Value([]byte("else"))
	failToValidateReply(t, identity, r2)
}

func TestReplyToReplySerializes(t *testing.T) {
	identity, privkey, _, _, reply := MakeReplyOrSkip(t)
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("hello"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte(""))
	r2, err := forest.As(identity, privkey).NewReply(reply, content, metadata)
	if err != nil {
		t.Error("Failed to create reply to existing reply", err)
	}
	ensureSerializes(t, r2)
}
