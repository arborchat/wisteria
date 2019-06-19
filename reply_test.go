package forest_test

import (
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

func TestNewReply(t *testing.T) {
	identity, privkey, community := MakeCommunityOrSkip(t)
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("test content"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte{})
	reply, err := forest.As(identity, privkey).NewReply(community, content, metadata)
	if err != nil {
		t.Error("Failed to create reply with valid parameters", err)
	}
	if !reply.Parent.Equals(community.ID()) {
		t.Error("Root Reply's parent is not parent community")
	} else if !reply.ConversationID.Equals(fields.NullHash()) {
		t.Error("Root Reply's conversation is not null hash")
	} else if !reply.CommunityID.Equals(community.ID()) {
		t.Error("Root Reply's community is not owning community")
	}
}

func TestNewReplyToReply(t *testing.T) {
	identity, privkey, community, reply := MakeReplyOrSkip(t)
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("other test content"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte{})
	reply2, err := forest.As(identity, privkey).NewReply(reply, content, metadata)
	if err != nil {
		t.Error("Failed to create reply with valid parameters", err)
	}
	if !reply2.Parent.Equals(reply.ID()) {
		t.Error("Reply's parent is not parent conversation")
	} else if !reply2.ConversationID.Equals(reply.ID()) {
		t.Error("Reply's conversation is not parent conversation")
	} else if !reply2.CommunityID.Equals(community.ID()) {
		t.Error("Reply's community is not owning community")
	}
}

func MakeReplyOrSkip(t *testing.T) (*forest.Identity, forest.Signer, *forest.Community, *forest.Reply) {
	identity, privkey, community := MakeCommunityOrSkip(t)
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("test content"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte{})
	reply, err := forest.As(identity, privkey).NewReply(community, content, metadata)
	if err != nil {
		t.Error("Failed to create reply with valid parameters", err)
	}
	return identity, privkey, community, reply
}

func TestReplyValidatesSelf(t *testing.T) {
	identity, _, _, reply := MakeReplyOrSkip(t)
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
	identity, _, _, reply := MakeReplyOrSkip(t)
	reply.Content.Blob = fields.Blob([]byte("whatever"))
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
	_, _, _, reply := MakeReplyOrSkip(t)
	ensureSerializes(t, reply)
}

func TestReplyToReplyValidates(t *testing.T) {
	identity, privkey, _, reply := MakeReplyOrSkip(t)
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("hello"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte(""))
	r2, err := forest.As(identity, privkey).NewReply(reply, content, metadata)
	if err != nil {
		t.Error("Failed to create reply to existing reply", err)
	}
	validateReply(t, identity, r2)
}

func TestReplyToReplyFailsWhenTampered(t *testing.T) {
	identity, privkey, _, reply := MakeReplyOrSkip(t)
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("hello"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte(""))
	r2, err := forest.As(identity, privkey).NewReply(reply, content, metadata)
	if err != nil {
		t.Error("Failed to create reply to existing reply", err)
	}
	r2.Content.Blob = fields.Blob([]byte("else"))
	failToValidateReply(t, identity, r2)
}

func TestReplyToReplySerializes(t *testing.T) {
	identity, privkey, _, reply := MakeReplyOrSkip(t)
	content := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte("hello"))
	metadata := QualifiedContentOrSkip(t, fields.ContentTypeUTF8String, []byte(""))
	r2, err := forest.As(identity, privkey).NewReply(reply, content, metadata)
	if err != nil {
		t.Error("Failed to create reply to existing reply", err)
	}
	ensureSerializes(t, r2)
}
