package forest

import (
	"bytes"
	"crypto/sha512"
	"encoding"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"math"

	"golang.org/x/crypto/openpgp"
	"golang.org/x/crypto/openpgp/packet"
)

const (
	Version Varint = 1

	NodeTypeIdentity     NodeType = 1
	NodeTypeCommunity    NodeType = 2
	NodeTypeConversation NodeType = 3
	NodeTypeReply        NodeType = 4

	ContentTypeUTF8String ContentType = 1
	ContentTypeJSON       ContentType = 2

	KeyTypeNoKey   KeyType = 0
	KeyTypeOpenPGP KeyType = 1

	SignatureTypeOpenPGP SignatureType = 1

	HashTypeNullHash   HashType = 0
	HashTypeSHA512_256 HashType = 1

	HashDigestLengthSHA512_256 ContentLength = 32
)

// fundamental types
type GenericType uint8

func (g GenericType) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	err := binary.Write(b, binary.BigEndian, g)
	return b.Bytes(), err
}

type ContentLength uint16

const (
	MaxContentLength = math.MaxUint16
)

func (c ContentLength) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	err := binary.Write(b, binary.BigEndian, c)
	return b.Bytes(), err
}

type TreeDepth uint32

func (t TreeDepth) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	err := binary.Write(b, binary.BigEndian, t)
	return b.Bytes(), err
}

type Value []byte

func (v Value) MarshalBinary() ([]byte, error) {
	return v, nil
}

type Varint uint64

func (v Varint) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	err := binary.Write(b, binary.BigEndian, v)
	return b.Bytes(), err
}

// specialized types
type NodeType GenericType

func (t NodeType) MarshalBinary() ([]byte, error) {
	return GenericType(t).MarshalBinary()
}

type HashType GenericType

func (t HashType) MarshalBinary() ([]byte, error) {
	return GenericType(t).MarshalBinary()
}

type ContentType GenericType

func (t ContentType) MarshalBinary() ([]byte, error) {
	return GenericType(t).MarshalBinary()
}

type KeyType GenericType

func (t KeyType) MarshalBinary() ([]byte, error) {
	return GenericType(t).MarshalBinary()
}

type SignatureType GenericType

func (t SignatureType) MarshalBinary() ([]byte, error) {
	return GenericType(t).MarshalBinary()
}

// generic descriptor
type Descriptor struct {
	Type   GenericType
	Length ContentLength
}

func NewDescriptor(t GenericType, length int) (*Descriptor, error) {
	if length > MaxContentLength {
		return nil, fmt.Errorf("Cannot represent content of length %d, max is %d", length, MaxContentLength)
	}
	d := Descriptor{}
	d.Type = t
	d.Length = ContentLength(length)
	return &d, nil
}

func (d Descriptor) MarshalBinary() ([]byte, error) {
	b, err := d.Type.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(b)
	b, err = d.Length.MarshalBinary()
	if err != nil {
		return nil, err
	}
	_, err = buf.Write(b)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil

}

// concrete descriptors
type HashDescriptor Descriptor

func NewHashDescriptor(t HashType, length int) (*HashDescriptor, error) {
	d, err := NewDescriptor(GenericType(t), length)
	return (*HashDescriptor)(d), err
}

func (d HashDescriptor) MarshalBinary() ([]byte, error) {
	return Descriptor(d).MarshalBinary()
}

type ContentDescriptor Descriptor

func NewContentDescriptor(t ContentType, length int) (*ContentDescriptor, error) {
	d, err := NewDescriptor(GenericType(t), length)
	return (*ContentDescriptor)(d), err
}

func (d ContentDescriptor) MarshalBinary() ([]byte, error) {
	return Descriptor(d).MarshalBinary()
}

type SignatureDescriptor Descriptor

func NewSignatureDescriptor(t SignatureType, length int) (*SignatureDescriptor, error) {
	d, err := NewDescriptor(GenericType(t), length)
	return (*SignatureDescriptor)(d), err
}

func (d SignatureDescriptor) MarshalBinary() ([]byte, error) {
	return Descriptor(d).MarshalBinary()
}

type KeyDescriptor Descriptor

func NewKeyDescriptor(t KeyType, length int) (*KeyDescriptor, error) {
	d, err := NewDescriptor(GenericType(t), length)
	return (*KeyDescriptor)(d), err
}

func (d KeyDescriptor) MarshalBinary() ([]byte, error) {
	return Descriptor(d).MarshalBinary()
}

// generic qualified data
type Qualified struct {
	Descriptor
	Value
}

func (q Qualified) MarshalBinary() ([]byte, error) {
	b, err := q.Descriptor.MarshalBinary()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(b)
	_, err = buf.Write([]byte(q.Value))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// NewQualified creates a valid Qualified from the given data
func NewQualified(t GenericType, content []byte) (*Qualified, error) {
	q := Qualified{}
	d, err := NewDescriptor(t, len(content))
	if err != nil {
		return nil, err
	}
	q.Descriptor = *d
	q.Value = Value(content)
	return &q, nil
}

func (q Qualified) Equals(o Qualified) bool {
	return q.Descriptor == o.Descriptor && bytes.Equal([]byte(q.Value), []byte(o.Value))
}

// concrete qualified data types
type QualifiedHash Qualified

// NewQualifiedHash returns a valid QualifiedHash from the given data
func NewQualifiedHash(t HashType, content []byte) (*QualifiedHash, error) {
	q, e := NewQualified(GenericType(t), content)
	return (*QualifiedHash)(q), e
}

func NullHash() QualifiedHash {
	return QualifiedHash{
		Descriptor: Descriptor{
			Type:   GenericType(HashTypeNullHash),
			Length: 0,
		},
		Value: []byte{},
	}
}

func (q QualifiedHash) MarshalBinary() ([]byte, error) {
	return Qualified(q).MarshalBinary()
}

type QualifiedContent Qualified

// NewQualifiedContent returns a valid QualifiedContent from the given data
func NewQualifiedContent(t ContentType, content []byte) (*QualifiedContent, error) {
	q, e := NewQualified(GenericType(t), content)
	return (*QualifiedContent)(q), e
}

func (q QualifiedContent) MarshalBinary() ([]byte, error) {
	return Qualified(q).MarshalBinary()
}

type QualifiedKey Qualified

// NewQualifiedKey returns a valid QualifiedKey from the given data
func NewQualifiedKey(t KeyType, content []byte) (*QualifiedKey, error) {
	q, e := NewQualified(GenericType(t), content)
	return (*QualifiedKey)(q), e
}

func (q QualifiedKey) MarshalBinary() ([]byte, error) {
	return Qualified(q).MarshalBinary()
}

type QualifiedSignature Qualified

// NewQualifiedSignature returns a valid QualifiedSignature from the given data
func NewQualifiedSignature(t SignatureType, content []byte) (*QualifiedSignature, error) {
	q, e := NewQualified(GenericType(t), content)
	return (*QualifiedSignature)(q), e
}

func (q QualifiedSignature) MarshalBinary() ([]byte, error) {
	return Qualified(q).MarshalBinary()
}

// generic node
type Node struct {
	// the ID is deterministically computed from the rest of the values
	id                 Value
	Type               NodeType
	Version            Varint
	Parent             QualifiedHash
	IDDesc             HashDescriptor
	Depth              TreeDepth
	Metadata           QualifiedContent
	SignatureAuthority QualifiedHash
	Signature          QualifiedSignature
	// WriteNodeTypeFieldsInto allows higher-level logic to define
	// how to serialize extra fields. See the concrete Node type
	// implementations for details
	WriteNodeTypeFieldsInto func(w io.Writer) error
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

// computeID determines the correct value of this node's ID without modifying
// the node.
func (n Node) computeID() ([]byte, error) {
	// map from HashType to the function that creates an instance of that hash
	// algorithm
	hashType2Func := map[HashType]func() hash.Hash{
		HashTypeSHA512_256: sha512.New512_256,
	}
	if HashType(n.IDDesc.Type) == HashTypeNullHash {
		return []byte{}, nil
	}
	binaryContent, err := n.MarshalBinary()
	if err != nil {
		return nil, err
	}
	hashFunc, found := hashType2Func[HashType(n.IDDesc.Type)]
	if !found {
		return nil, fmt.Errorf("Unknown HashType %d", n.IDDesc.Type)
	}
	hasher := hashFunc()
	_, _ = hasher.Write(binaryContent) // never errors
	return hasher.Sum(nil), nil
}

// ValidateID returns whether the ID of this Node matches the data. The first
// return value indicates the result of the comparison. If there is an error,
// the first return value will always be false and the second will indicate
// what went wrong when computing the hash.
func (n Node) ValidateID() (bool, error) {
	currentID := n.ID()
	id, err := n.computeID()
	if err != nil {
		return false, err
	}
	computedID := QualifiedHash{
		Descriptor: Descriptor(n.IDDesc),
		Value:      Value(id),
	}
	return Qualified(currentID).Equals(Qualified(computedID)), nil
}

// ValidateSignature returns whether the signature contained in this Node is a valid
// signature for the given Identity. When validating an Identity node, you should
// pass the Identity to this method.
func (n Node) ValidateSignatureFor(identity *Identity) (bool, error) {
	if Qualified(n.SignatureAuthority).Equals(Qualified(NullHash())) {
		if n.Type != NodeTypeIdentity {
			return false, fmt.Errorf("Only Identity nodes can have the null hash as their Signature Authority")
		}
	} else if !Qualified(n.SignatureAuthority).Equals(Qualified(identity.ID())) {
		return false, fmt.Errorf("This node was signed by a different identity")
	}
	// get the key used to sign this node
	pubkeyBuf := bytes.NewBuffer([]byte(identity.PublicKey.Value))
	pubkeyEntity, err := openpgp.ReadEntity(packet.NewReader(pubkeyBuf))
	if err != nil {
		return false, err
	}

	signedContentBuf := new(bytes.Buffer)
	if err = n.WriteDataForSigningInto(signedContentBuf); err != nil {
		return false, err
	}
	signatureBuf := bytes.NewBuffer([]byte(n.Signature.Value))
	keyring := openpgp.EntityList([]*openpgp.Entity{pubkeyEntity})
	_, err = openpgp.CheckDetachedSignature(keyring, signedContentBuf, signatureBuf)
	if err != nil {
		return false, err
	}
	return true, nil
}

// Compute and return the Node's ID as a Qualified Hash
func (n Node) ID() QualifiedHash {
	return QualifiedHash{
		Descriptor: Descriptor(n.IDDesc),
		Value:      n.id,
	}
}

func (n Node) WriteCommonFieldsInto(w io.Writer) error {
	// this slice defines the order in which the fields are written
	return MarshalAllInto(w,
		n.Version,
		n.Type,
		n.Parent,
		n.IDDesc,
		n.Depth,
		n.Metadata,
		n.SignatureAuthority)
}

func (n Node) WriteSignatureInto(w io.Writer) error {
	return MarshalAllInto(w, n.Signature)
}

func (n Node) WriteDataForSigningInto(w io.Writer) error {
	if err := n.WriteCommonFieldsInto(w); err != nil {
		return err
	}
	if err := n.WriteNodeTypeFieldsInto(w); err != nil {
		return err
	}
	return nil
}

func (n Node) MarshalBinary() ([]byte, error) {
	// this is a template method. It always writes the common fields,
	// then invokes a method responsible for writing data that varies
	// between Node Types, then writes the final data
	b := new(bytes.Buffer)
	writeFuncs := []func(io.Writer) error{
		n.WriteDataForSigningInto,
		n.WriteSignatureInto,
	}

	// invoke the methods in the order defined by the slice above
	for _, f := range writeFuncs {
		err := f(b)
		if err != nil {
			return nil, err
		}
	}
	// invoke the methods in the order defined by the slice above	}
	return b.Bytes(), nil
}

// concrete nodes
type Identity struct {
	Node
	Name      QualifiedContent
	PublicKey QualifiedKey
}

func newIdentity() *Identity {
	i := new(Identity)
	// define how to serialize this node type's fields
	i.Node.WriteNodeTypeFieldsInto = func(w io.Writer) error {
		return MarshalAllInto(w, i.Name, i.PublicKey)
	}
	return i
}

func (i Identity) MarshalBinary() ([]byte, error) {
	return i.Node.MarshalBinary()
}

type Community struct {
	Node
	Name QualifiedContent
}

func (c Community) MarshalBinary() ([]byte, error) {
	// define how to serialize this node type's fields
	c.Node.WriteNodeTypeFieldsInto = func(w io.Writer) error {
		return MarshalAllInto(w, c.Name)
	}
	return c.Node.MarshalBinary()
}

type Conversation struct {
	Node
	Content QualifiedContent
}

func (c Conversation) MarshalBinary() ([]byte, error) {
	// define how to serialize this node type's fields
	c.Node.WriteNodeTypeFieldsInto = func(w io.Writer) error {
		return MarshalAllInto(w, c.Content)
	}
	return c.Node.MarshalBinary()
}

type Reply struct {
	Node
	ConversationID QualifiedHash
	Content        QualifiedContent
}

func (r Reply) MarshalBinary() ([]byte, error) {
	// define how to serialize this node type's fields
	r.Node.WriteNodeTypeFieldsInto = func(w io.Writer) error {
		return MarshalAllInto(w, r.Content)
	}
	return r.Node.MarshalBinary()
}
