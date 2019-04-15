package forest

import (
	"bytes"
	"encoding"
	"io"
)

// generic node
type commonNode struct {
	// the ID is deterministically computed from the rest of the values
	id                 Value
	Type               NodeType
	SchemaVersion      Version
	Parent             QualifiedHash
	IDDesc             HashDescriptor
	Depth              TreeDepth
	Metadata           QualifiedContent
	SignatureAuthority QualifiedHash
	Signature          QualifiedSignature
}

// Compute and return the commonNode's ID as a Qualified Hash
func (n commonNode) ID() QualifiedHash {
	return QualifiedHash{
		Descriptor: descriptor(n.IDDesc),
		Value:      n.id,
	}
}

func (n *commonNode) serializationOrder() []BidirectionalBinaryMarshaler {
	return []BidirectionalBinaryMarshaler{
		&n.SchemaVersion,
		&n.Type,
		&n.Parent,
		&n.IDDesc,
		&n.Depth,
		&n.Metadata,
		&n.SignatureAuthority,
		&n.Signature,
	}
}

func (n *commonNode) presignSerializationOrder() []BidirectionalBinaryMarshaler {
	fields := n.serializationOrder()
	fields = fields[:len(fields)-1] // drop the signature
	return fields
}

func (n *commonNode) postsignSerializationOrder() []BidirectionalBinaryMarshaler {
	fields := n.serializationOrder()
	return fields[len(fields)-1:]
}

// unmarshalBinaryPreamble does the unmarshaling work for all of the common
// node fields before the node-specific fields and returns the unused data.
func (n *commonNode) unmarshalBinaryPreamble(b []byte) ([]byte, error) {
	return UnmarshalAll(b, asUnmarshaler(n.presignSerializationOrder())...)
}

// unmarshalBinarySignature does the unmarshaling work for the signature field after the
// node-specific fields and returns the unused data.
func (n *commonNode) unmarshalBinarySignature(b []byte) ([]byte, error) {
	return UnmarshalAll(b, asUnmarshaler(n.postsignSerializationOrder())...)
}

// concrete nodes
type Identity struct {
	commonNode
	Name      QualifiedContent
	PublicKey QualifiedKey
}

func newIdentity() *Identity {
	i := new(Identity)
	// define how to serialize this node type's fields
	return i
}

func (i *Identity) serializationOrder() []BidirectionalBinaryMarshaler {
	return []BidirectionalBinaryMarshaler{&i.Name, &i.PublicKey}
}

func (i Identity) MarshalSignedData() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := MarshalAllInto(buf, asMarshaler(i.presignSerializationOrder())...); err != nil {
		return nil, err
	}
	if err := MarshalAllInto(buf, asMarshaler(i.serializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (i Identity) Signature() QualifiedSignature {
	return i.commonNode.Signature
}

func (i Identity) SignatureIdentityHash() QualifiedHash {
	return i.commonNode.SignatureAuthority
}

func (i Identity) IsIdentity() bool {
	return true
}

func (i Identity) HashDescriptor() *HashDescriptor {
	return &i.commonNode.IDDesc
}

func (i Identity) MarshalBinary() ([]byte, error) {
	signed, err := i.MarshalSignedData()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(signed)
	if err := MarshalAllInto(buf, asMarshaler(i.postsignSerializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnmarshalIdentity(b []byte) (*Identity, error) {
	i := newIdentity()
	if err := i.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return i, nil
}

func (i *Identity) UnmarshalBinary(b []byte) error {
	unused, err := i.commonNode.unmarshalBinaryPreamble(b)
	if err != nil {
		return err
	}
	unused, err = UnmarshalAll(unused, asUnmarshaler(i.serializationOrder())...)
	if err != nil {
		return err
	}
	if _, err := i.commonNode.unmarshalBinarySignature(unused); err != nil {
		return err
	}
	return nil
}

func marshalTextWithPrefix(w io.Writer, prefix string, target encoding.TextMarshaler) error {
	b, err := target.MarshalText()
	if err != nil {
		return err
	}
	if _, err := w.Write([]byte(prefix)); err != nil {
		return err
	}
	if _, err := w.Write(b); err != nil {
		return err
	}
	return nil
}

func (i *Identity) MarshalText() ([]byte, error) {
	buf := new(bytes.Buffer)
	if _, err := buf.WriteString("identity {"); err != nil {
		return nil, err
	}
	id := i.ID()
	if err := marshalTextWithPrefix(buf, "\n\tID: ", id); err != nil {
		return nil, err
	}
	if err := marshalTextWithPrefix(buf, "\n\tParent: ", i.Parent); err != nil {
		return nil, err
	}
	if err := marshalTextWithPrefix(buf, "\n\tName: ", i.Name); err != nil {
		return nil, err
	}
	if err := marshalTextWithPrefix(buf, "\n\tPublicKey: ", i.PublicKey); err != nil {
		return nil, err
	}
	if err := marshalTextWithPrefix(buf, "\n\tMetadata: ", i.Metadata); err != nil {
		return nil, err
	}
	if err := marshalTextWithPrefix(buf, "\n\tSignatureAuthority: ", i.SignatureAuthority); err != nil {
		return nil, err
	}
	if err := marshalTextWithPrefix(buf, "\n\tSignature: ", i.Signature()); err != nil {
		return nil, err
	}
	if _, err := buf.WriteString("\n}"); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type Community struct {
	commonNode
	Name QualifiedContent
}

func newCommunity() *Community {
	c := new(Community)
	// define how to serialize this node type's fields
	return c
}

type Conversation struct {
	commonNode
	Content QualifiedContent
}

func newConversation() *Conversation {
	c := new(Conversation)
	// define how to serialize this node type's fields
	return c
}

type Reply struct {
	commonNode
	ConversationID QualifiedHash
	Content        QualifiedContent
}

func newReply() *Reply {
	r := new(Reply)
	// define how to serialize this node type's fields
	return r
}
