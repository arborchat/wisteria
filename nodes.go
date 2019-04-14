package forest

import (
	"bytes"
	"encoding"
	"io"
)

type BidirectionalBinaryMarshaler interface {
	encoding.BinaryMarshaler
	encoding.BinaryUnmarshaler
}

func asMarshaler(in []BidirectionalBinaryMarshaler) []encoding.BinaryMarshaler {
	out := make([]encoding.BinaryMarshaler, len(in))
	for i, f := range in {
		out[i] = encoding.BinaryMarshaler(f)
	}
	return out
}

func asUnmarshaler(in []BidirectionalBinaryMarshaler) []encoding.BinaryUnmarshaler {
	out := make([]encoding.BinaryUnmarshaler, len(in))
	for i, f := range in {
		out[i] = encoding.BinaryUnmarshaler(f)
	}
	return out
}

func MarshalAllInto(w io.Writer, marshalers ...encoding.BinaryMarshaler) error {
	for _, marshaler := range marshalers {
		b, err := marshaler.MarshalBinary()
		if err != nil {
			return err
		}
		_, err = w.Write(b)
		if err != nil {
			return err
		}
	}
	return nil
}

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
	runningBytesConsumed := 0
	if err := n.SchemaVersion.UnmarshalBinary(b[runningBytesConsumed:sizeofVersion]); err != nil {
		return nil, err
	}
	runningBytesConsumed += sizeofVersion
	if err := n.Type.UnmarshalBinary(b[:sizeofgenericType]); err != nil {
		return nil, err
	}
	runningBytesConsumed += sizeofgenericType
	if err := n.Parent.UnmarshalBinary(b[runningBytesConsumed:]); err != nil {
		return nil, err
	}
	runningBytesConsumed += minSizeofQualifiedHash + int(n.Parent.Descriptor.Length)
	if err := n.IDDesc.UnmarshalBinary(b[runningBytesConsumed:sizeofHashDescriptor]); err != nil {
		return nil, err
	}
	runningBytesConsumed += sizeofHashDescriptor
	if err := n.Depth.UnmarshalBinary(b[runningBytesConsumed:sizeofTreeDepth]); err != nil {
		return nil, err
	}
	runningBytesConsumed += sizeofTreeDepth
	if err := n.Metadata.UnmarshalBinary(b[runningBytesConsumed:]); err != nil {
		return nil, err
	}
	runningBytesConsumed += minSizeofQualifiedContent + int(n.Metadata.Descriptor.Length)
	if err := n.SignatureAuthority.UnmarshalBinary(b[runningBytesConsumed:]); err != nil {
		return nil, err
	}
	runningBytesConsumed += minSizeofQualifiedHash + int(n.SignatureAuthority.Descriptor.Length)
	return b[runningBytesConsumed:], nil
}

// unmarshalBinarySignature does the unmarshaling work for the signature field after the
// node-specific fields and returns the unused data.
func (n *commonNode) unmarshalBinarySignature(b []byte) ([]byte, error) {
	if err := n.Signature.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return b[minSizeofQualifiedSignature+int(n.Signature.Descriptor.Length):], nil
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
	b = unused
	runningBytesConsumed := 0
	if err := i.Name.UnmarshalBinary(b[runningBytesConsumed:]); err != nil {
		return err
	}
	runningBytesConsumed += int(i.Name.Descriptor.Length)
	if err := i.PublicKey.UnmarshalBinary(b[runningBytesConsumed:]); err != nil {
		return err
	}
	runningBytesConsumed += int(i.PublicKey.Descriptor.Length)
	if _, err := i.commonNode.unmarshalBinarySignature(b[runningBytesConsumed:]); err != nil {
		return err
	}
	return nil
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
