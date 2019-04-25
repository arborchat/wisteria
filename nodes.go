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
func (n commonNode) ID() *QualifiedHash {
	return &QualifiedHash{
		Descriptor: n.IDDesc,
		Value:      n.id,
	}
}

func (n *commonNode) presignSerializationOrder() []BidirectionalBinaryMarshaler {
	order := []BidirectionalBinaryMarshaler{
		&n.SchemaVersion,
		&n.Type,
	}
	order = append(order, &n.Parent)
	order = append(order, n.IDDesc.serializationOrder()...)
	order = append(order, &n.Depth)
	order = append(order, &n.Metadata)
	order = append(order, &n.SignatureAuthority)
	return order
}

func (n *commonNode) postsignSerializationOrder() []BidirectionalBinaryMarshaler {
	return []BidirectionalBinaryMarshaler{&n.Signature}
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

func (n *commonNode) Equals(n2 *commonNode) bool {
	return n.Type.Equals(&n2.Type) &&
		n.SchemaVersion.Equals(&n2.SchemaVersion) &&
		n.Parent.Equals(&n2.Parent) &&
		n.IDDesc.Equals(&n2.IDDesc) &&
		n.Depth.Equals(&n2.Depth) &&
		n.Metadata.Equals(&n2.Metadata) &&
		n.SignatureAuthority.Equals(&n2.SignatureAuthority) &&
		n.Signature.Equals(&n2.Signature)
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

func (i *Identity) nodeSpecificSerializationOrder() []BidirectionalBinaryMarshaler {
	return []BidirectionalBinaryMarshaler{&i.Name, &i.PublicKey}
}

func (i *Identity) serializationOrder() []BidirectionalBinaryMarshaler {
	order := i.commonNode.presignSerializationOrder()
	order = append(order, i.nodeSpecificSerializationOrder()...)
	order = append(order, i.commonNode.postsignSerializationOrder()...)
	return order
}

func (i Identity) MarshalSignedData() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := MarshalAllInto(buf, asMarshaler(i.presignSerializationOrder())...); err != nil {
		return nil, err
	}
	if err := MarshalAllInto(buf, asMarshaler(i.nodeSpecificSerializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (i Identity) Signature() *QualifiedSignature {
	return &i.commonNode.Signature
}

func (i Identity) SignatureIdentityHash() *QualifiedHash {
	return &i.commonNode.SignatureAuthority
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
	_, err := UnmarshalAll(b, asUnmarshaler(i.serializationOrder())...)
	if err != nil {
		return err
	}
	idBytes, err := computeID(i)
	if err != nil {
		return err
	}
	i.id = Value(idBytes)
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

/*func (i *Identity) MarshalText() ([]byte, error) {
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
}*/

func (i *Identity) Equals(i2 *Identity) bool {
	return i.commonNode.Equals(&i2.commonNode) &&
		i.Name.Equals(&i2.Name) &&
		i.PublicKey.Equals(&i2.PublicKey)
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

func (c *Community) nodeSpecificSerializationOrder() []BidirectionalBinaryMarshaler {
	return []BidirectionalBinaryMarshaler{&c.Name}
}

func (c *Community) serializationOrder() []BidirectionalBinaryMarshaler {
	order := c.commonNode.presignSerializationOrder()
	order = append(order, c.nodeSpecificSerializationOrder()...)
	order = append(order, c.commonNode.postsignSerializationOrder()...)
	return order
}

func (c Community) MarshalSignedData() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := MarshalAllInto(buf, asMarshaler(c.presignSerializationOrder())...); err != nil {
		return nil, err
	}
	if err := MarshalAllInto(buf, asMarshaler(c.nodeSpecificSerializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c Community) Signature() *QualifiedSignature {
	return &c.commonNode.Signature
}

func (c Community) SignatureIdentityHash() *QualifiedHash {
	return &c.commonNode.SignatureAuthority
}

func (c Community) IsIdentity() bool {
	return false
}

func (c Community) HashDescriptor() *HashDescriptor {
	return &c.commonNode.IDDesc
}

func (c Community) MarshalBinary() ([]byte, error) {
	signed, err := c.MarshalSignedData()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(signed)
	if err := MarshalAllInto(buf, asMarshaler(c.postsignSerializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnmarshalCommunity(b []byte) (*Community, error) {
	c := newCommunity()
	if err := c.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Community) UnmarshalBinary(b []byte) error {
	_, err := UnmarshalAll(b, asUnmarshaler(c.serializationOrder())...)
	if err != nil {
		return err
	}
	idBytes, err := computeID(c)
	if err != nil {
		return err
	}
	c.id = Value(idBytes)
	return nil
}

func (c *Community) Equals(c2 *Community) bool {
	return c.commonNode.Equals(&c2.commonNode) &&
		c.Name.Equals(&c2.Name)
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
