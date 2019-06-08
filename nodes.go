package forest

import (
	"bytes"
	"fmt"

	"git.sr.ht/~whereswaldon/forest-go/fields"
)

const MaxNameLength = 256

type Validator interface {
	Validate() error
}

type Node interface {
	ID() *fields.QualifiedHash
	ParentID() *fields.QualifiedHash
	Equals(interface{}) bool
	ValidateShallow() error
	ValidateDeep(Store) error
}

// NodeTypeOf returns the NodeType of the provided binary-marshaled node.
// If the provided bytes are not a forest node or the type cannot be determined,
// an error will be returned and the first return value must be ignored.
func NodeTypeOf(b []byte) (fields.NodeType, error) {
	_, t, err := VersionAndNodeTypeOf(b)
	return t, err
}

func VersionAndNodeTypeOf(b []byte) (fields.Version, fields.NodeType, error) {
	var (
		ver fields.Version
		t   fields.NodeType
		// this array defines the serialization order of the first two fields of
		// any node. If this order ever changes, it must be updated here and in
		// commonNode.presignSerializationOrder
		order = []fields.BidirectionalBinaryMarshaler{
			&ver,
			&t,
		}
	)
	_, err := fields.UnmarshalAll(b, fields.AsUnmarshaler(order)...)
	return ver, t, err
}

// UnmarshalBinaryNode unmarshals a node of any type. If it does not return an
// error, the concrete type of the first return parameter will be one of the
// node structs declared in this package (e.g. Identity, Community, etc...)
func UnmarshalBinaryNode(b []byte) (Node, error) {
	v, t, err := VersionAndNodeTypeOf(b)
	if err != nil {
		return nil, err
	}
	if v > fields.CurrentVersion {
		return nil, fmt.Errorf("Unable to unmarshal node of version %d, only supports <= %d", v, fields.CurrentVersion)
	}
	switch t {
	case fields.NodeTypeIdentity:
		return UnmarshalIdentity(b)
	case fields.NodeTypeCommunity:
		return UnmarshalCommunity(b)
	case fields.NodeTypeReply:
		return UnmarshalReply(b)
	default:
		return nil, fmt.Errorf("Unable to unmarshal node of type %d, unknown type", t)
	}
}

// generic node
type commonNode struct {
	// the ID is deterministically computed from the rest of the values
	id                 fields.Blob
	Type               fields.NodeType
	SchemaVersion      fields.Version
	Parent             fields.QualifiedHash
	IDDesc             fields.HashDescriptor
	Depth              fields.TreeDepth
	Metadata           fields.QualifiedContent
	Author fields.QualifiedHash
	Signature          fields.QualifiedSignature
}

// Compute and return the commonNode's ID as a fields.Qualified Hash
func (n commonNode) ID() *fields.QualifiedHash {
	return &fields.QualifiedHash{
		Descriptor: n.IDDesc,
		Blob:      n.id,
	}
}

func (n commonNode) ParentID() *fields.QualifiedHash {
	return &fields.QualifiedHash{n.Parent.Descriptor, n.Parent.Blob}
}

func (n *commonNode) presignSerializationOrder() []fields.BidirectionalBinaryMarshaler {
	order := []fields.BidirectionalBinaryMarshaler{
		&n.SchemaVersion,
		&n.Type,
	}
	order = append(order, &n.Parent)
	order = append(order, n.IDDesc.SerializationOrder()...)
	order = append(order, &n.Depth)
	order = append(order, &n.Metadata)
	order = append(order, &n.Author)
	return order
}

func (n *commonNode) postsignSerializationOrder() []fields.BidirectionalBinaryMarshaler {
	return []fields.BidirectionalBinaryMarshaler{&n.Signature}
}

// unmarshalBinaryPreamble does the unmarshaling work for all of the common
// node fields before the node-specific fields and returns the unused data.
func (n *commonNode) unmarshalBinaryPreamble(b []byte) ([]byte, error) {
	return fields.UnmarshalAll(b, fields.AsUnmarshaler(n.presignSerializationOrder())...)
}

// unmarshalBinarySignature does the unmarshaling work for the signature field after the
// node-specific fields and returns the unused data.
func (n *commonNode) unmarshalBinarySignature(b []byte) ([]byte, error) {
	return fields.UnmarshalAll(b, fields.AsUnmarshaler(n.postsignSerializationOrder())...)
}

// GetSignature returns the signature for the node, which must correspond to the Signature Authority for
// the node in order to be valid.
func (n *commonNode) GetSignature() *fields.QualifiedSignature {
	return &n.Signature
}

// SignatureIdentityHash returns the node identitifer for the Identity that signed this node.
func (n *commonNode) SignatureIdentityHash() *fields.QualifiedHash {
	return &n.Author
}

func (n commonNode) IsIdentity() bool {
	return n.Type == fields.NodeTypeIdentity
}

func (n commonNode) HashDescriptor() *fields.HashDescriptor {
	return &n.IDDesc
}

func (n *commonNode) Equals(n2 *commonNode) bool {
	return n.Type.Equals(&n2.Type) &&
		n.SchemaVersion.Equals(&n2.SchemaVersion) &&
		n.Parent.Equals(&n2.Parent) &&
		n.IDDesc.Equals(&n2.IDDesc) &&
		n.Depth.Equals(&n2.Depth) &&
		n.Metadata.Equals(&n2.Metadata) &&
		n.Author.Equals(&n2.Author) &&
		n.Signature.Equals(&n2.Signature)
}

// ValidateShallow checks all fields for internal validity. It does not check
// the existence or validity of nodes referenced from this node.
func (n *commonNode) ValidateShallow() error {
	if _, validType := fields.ValidNodeTypes[n.Type]; !validType {
		return fmt.Errorf("%d is not a valid node type", n.Type)
	}
	if n.SchemaVersion > fields.CurrentVersion {
		return fmt.Errorf("%d is higher than than the supported version %d", n.SchemaVersion, fields.CurrentVersion)
	}
	id := n.ID()
	needsValidation := []Validator{id, &n.Parent, &n.Metadata, &n.Author, &n.Signature}
	for _, nv := range needsValidation {
		if err := nv.Validate(); err != nil {
			return err
		}
	}
	if n.Metadata.Descriptor.Type != fields.ContentTypeJSON {
		return fmt.Errorf("Metadata must be JSON, got content type %d", n.Metadata.Descriptor.Type)
	}
	return nil
}

// ValidateDeep checks for the existence of all referenced nodes within the provided store.
func (n *commonNode) ValidateDeep(store Store) error {
	// ensure known parent
	if !n.Parent.Equals(fields.NullHash()) {
		if _, has, err := store.Get(&n.Parent); !has {
			return fmt.Errorf("Unknown parent %v", n.Parent)
		} else if err != nil {
			return err
		}
	}
	// ensure known author
	if !n.Author.Equals(fields.NullHash()) {
		if _, has, err := store.Get(&n.Author); !has {
			return fmt.Errorf("Unknown Author %v", n.Author)
		} else if err != nil {
			return err
		}
	}
	return nil
}

// concrete nodes

// Identity nodes represent a user. They associate a username with a public key that the user
// will sign messages with.
type Identity struct {
	commonNode
	Name      fields.QualifiedContent
	PublicKey fields.QualifiedKey
}

func newIdentity() *Identity {
	i := new(Identity)
	// define how to serialize this node type's fields
	return i
}

func (i *Identity) nodeSpecificSerializationOrder() []fields.BidirectionalBinaryMarshaler {
	return []fields.BidirectionalBinaryMarshaler{&i.Name, &i.PublicKey}
}

func (i *Identity) SerializationOrder() []fields.BidirectionalBinaryMarshaler {
	order := i.commonNode.presignSerializationOrder()
	order = append(order, i.nodeSpecificSerializationOrder()...)
	order = append(order, i.commonNode.postsignSerializationOrder()...)
	return order
}

// MarshalSignedData writes all data that should be signed in the correct order for signing. This
// can be used both to generate and validate message signatures.
func (i Identity) MarshalSignedData() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := fields.MarshalAllInto(buf, fields.AsMarshaler(i.presignSerializationOrder())...); err != nil {
		return nil, err
	}
	if err := fields.MarshalAllInto(buf, fields.AsMarshaler(i.nodeSpecificSerializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (i Identity) MarshalBinary() ([]byte, error) {
	signed, err := i.MarshalSignedData()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(signed)
	if err := fields.MarshalAllInto(buf, fields.AsMarshaler(i.postsignSerializationOrder())...); err != nil {
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
	_, err := fields.UnmarshalAll(b, fields.AsUnmarshaler(i.SerializationOrder())...)
	if err != nil {
		return err
	}
	idBytes, err := computeID(i)
	if err != nil {
		return err
	}
	i.id = fields.Blob(idBytes)
	return nil
}

func (i *Identity) Equals(other interface{}) bool {
	i2, valid := other.(*Identity)
	if !valid {
		return false
	}
	return i.commonNode.Equals(&i2.commonNode) &&
		i.Name.Equals(&i2.Name) &&
		i.PublicKey.Equals(&i2.PublicKey)
}

// ValidateShallow checks all fields for internal validity. It does not check
// the existence or validity of nodes referenced from this node.
func (i *Identity) ValidateShallow() error {
	if err := i.commonNode.ValidateShallow(); err != nil {
		return err
	}
	needsValidation := []Validator{&i.Name, &i.PublicKey}
	for _, nv := range needsValidation {
		if err := nv.Validate(); err != nil {
			return err
		}
	}
	if i.Name.Descriptor.Length > MaxNameLength {
		return fmt.Errorf("Name is longer than maximum of %d", MaxNameLength)
	}
	if i.Depth != fields.TreeDepth(0) {
		return fmt.Errorf("Identity depth must be 0, got %d", i.Depth)
	}
	if !i.Parent.Equals(fields.NullHash()) {
		return fmt.Errorf("Identity parent must be null hash, got %v", i.Parent)
	}
	if !i.Author.Equals(fields.NullHash()) {
		return fmt.Errorf("Identity author must be null hash, got %v", i.Author)
	}
	return nil
}

// ValidateDeep checks all referenced nodes for existence within the store.
func (i *Identity) ValidateDeep(store Store) error {
	return nil
}

type Community struct {
	commonNode
	Name fields.QualifiedContent
}

func newCommunity() *Community {
	c := new(Community)
	// define how to serialize this node type's fields
	return c
}

func (c *Community) nodeSpecificSerializationOrder() []fields.BidirectionalBinaryMarshaler {
	return []fields.BidirectionalBinaryMarshaler{&c.Name}
}

func (c *Community) SerializationOrder() []fields.BidirectionalBinaryMarshaler {
	order := c.commonNode.presignSerializationOrder()
	order = append(order, c.nodeSpecificSerializationOrder()...)
	order = append(order, c.commonNode.postsignSerializationOrder()...)
	return order
}

func (c Community) MarshalSignedData() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := fields.MarshalAllInto(buf, fields.AsMarshaler(c.presignSerializationOrder())...); err != nil {
		return nil, err
	}
	if err := fields.MarshalAllInto(buf, fields.AsMarshaler(c.nodeSpecificSerializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (c Community) MarshalBinary() ([]byte, error) {
	signed, err := c.MarshalSignedData()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(signed)
	if err := fields.MarshalAllInto(buf, fields.AsMarshaler(c.postsignSerializationOrder())...); err != nil {
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
	_, err := fields.UnmarshalAll(b, fields.AsUnmarshaler(c.SerializationOrder())...)
	if err != nil {
		return err
	}
	idBytes, err := computeID(c)
	if err != nil {
		return err
	}
	c.id = fields.Blob(idBytes)
	return nil
}

func (c *Community) Equals(other interface{}) bool {
	c2, valid := other.(*Community)
	if !valid {
		return false
	}
	return c.commonNode.Equals(&c2.commonNode) &&
		c.Name.Equals(&c2.Name)
}

// ValidateShallow checks all fields for internal validity. It does not check
// the existence or validity of nodes referenced from this node.
func (c *Community) ValidateShallow() error {
	if err := c.commonNode.ValidateShallow(); err != nil {
		return err
	}
	needsValidation := []Validator{&c.Name}
	for _, nv := range needsValidation {
		if err := nv.Validate(); err != nil {
			return err
		}
	}
	if c.Name.Descriptor.Length > MaxNameLength {
		return fmt.Errorf("Name is longer than maximum of %d", MaxNameLength)
	}
	if c.Depth != fields.TreeDepth(0) {
		return fmt.Errorf("Community depth must be 0, got %d", c.Depth)
	}
	if !c.Parent.Equals(fields.NullHash()) {
		return fmt.Errorf("Community parent must be null hash, got %v", c.Parent)
	}
	if c.Author.Equals(fields.NullHash()) {
		return fmt.Errorf("Community author must not be null hash")
	}
	return nil
}

// ValidateDeep checks all referenced nodes for existence within the store.
func (c *Community) ValidateDeep(store Store) error {
	if _, has, err := store.Get(&c.Author); !has {
		return fmt.Errorf("Missing author node %v", c.Author)
	} else if err != nil {
		return err
	}
	return nil
}

type Reply struct {
	commonNode
	CommunityID    fields.QualifiedHash
	ConversationID fields.QualifiedHash
	Content        fields.QualifiedContent
}

func newReply() *Reply {
	r := new(Reply)
	// define how to serialize this node type's fields
	return r
}

func (r *Reply) nodeSpecificSerializationOrder() []fields.BidirectionalBinaryMarshaler {
	return []fields.BidirectionalBinaryMarshaler{&r.CommunityID, &r.ConversationID, &r.Content}
}

func (r *Reply) SerializationOrder() []fields.BidirectionalBinaryMarshaler {
	order := r.commonNode.presignSerializationOrder()
	order = append(order, r.nodeSpecificSerializationOrder()...)
	order = append(order, r.commonNode.postsignSerializationOrder()...)
	return order
}

func (r Reply) MarshalSignedData() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := fields.MarshalAllInto(buf, fields.AsMarshaler(r.presignSerializationOrder())...); err != nil {
		return nil, err
	}
	if err := fields.MarshalAllInto(buf, fields.AsMarshaler(r.nodeSpecificSerializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r Reply) MarshalBinary() ([]byte, error) {
	signed, err := r.MarshalSignedData()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(signed)
	if err := fields.MarshalAllInto(buf, fields.AsMarshaler(r.postsignSerializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnmarshalReply(b []byte) (*Reply, error) {
	r := newReply()
	if err := r.UnmarshalBinary(b); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *Reply) UnmarshalBinary(b []byte) error {
	_, err := fields.UnmarshalAll(b, fields.AsUnmarshaler(r.SerializationOrder())...)
	if err != nil {
		return err
	}
	idBytes, err := computeID(r)
	if err != nil {
		return err
	}
	r.id = fields.Blob(idBytes)
	return nil
}

func (r *Reply) Equals(other interface{}) bool {
	r2, valid := other.(*Reply)
	if !valid {
		return false
	}
	return r.commonNode.Equals(&r2.commonNode) &&
		r.Content.Equals(&r2.Content)
}

// ValidateShallow checks all fields for internal validity. It does not check
// the existence or validity of nodes referenced from this node.
func (r *Reply) ValidateShallow() error {
	if err := r.commonNode.ValidateShallow(); err != nil {
		return err
	}
	needsValidation := []Validator{&r.Content, &r.CommunityID, &r.ConversationID}
	for _, nv := range needsValidation {
		if err := nv.Validate(); err != nil {
			return err
		}
	}
	if r.Depth < fields.TreeDepth(1) {
		return fmt.Errorf("Reply depth must be at least 1, got %d", r.Depth)
	} else if r.Depth == fields.TreeDepth(1) && !r.ConversationID.Equals(fields.NullHash()) {
		return fmt.Errorf("Reply conversation id at depth 1 must be null hash")
	} else if r.Depth > fields.TreeDepth(1) && r.ConversationID.Equals(fields.NullHash()) {
		return fmt.Errorf("Reply conversation id at depth > 1 must be null hash, got %v", r.ConversationID)
	}
	if r.Parent.Equals(fields.NullHash()) {
		return fmt.Errorf("Reply parent must not be null hash")
	}
	if r.Author.Equals(fields.NullHash()) {
		return fmt.Errorf("Reply author must not be null hash")
	}
	if r.CommunityID.Equals(fields.NullHash()) {
		return fmt.Errorf("Reply community id must not be null hash")
	}
	return nil
}

// ValidateDeep checks all referenced nodes for existence within the store.
func (r *Reply) ValidateDeep(store Store) error {
	needed := []*fields.QualifiedHash{&r.Author, &r.Parent, &r.CommunityID}
	if r.Depth > fields.TreeDepth(1) {
		needed = append(needed, &r.ConversationID)
	}
	for _, neededNode := range needed {
		if _, has, err := store.Get(neededNode); !has {
			return fmt.Errorf("Missing required node %v", neededNode)
		} else if err != nil {
			return err
		}
	}
	return nil
}
