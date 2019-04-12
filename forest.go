package forest

import (
	"bytes"
	"encoding"
	"encoding/binary"
	"io"
)

// fundamental types
type GenericType uint8

func (g GenericType) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	err := binary.Write(b, binary.BigEndian, g)
	return b.Bytes(), err
}

type ContentLength uint16

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

func (d HashDescriptor) MarshalBinary() ([]byte, error) {
	return Descriptor(d).MarshalBinary()
}

type ContentDescriptor Descriptor

func (d ContentDescriptor) MarshalBinary() ([]byte, error) {
	return Descriptor(d).MarshalBinary()
}

type SignatureDescriptor Descriptor

func (d SignatureDescriptor) MarshalBinary() ([]byte, error) {
	return Descriptor(d).MarshalBinary()
}

type KeyDescriptor Descriptor

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

// concrete qualified data types
type QualifiedHash Qualified

func (q QualifiedHash) MarshalBinary() ([]byte, error) {
	return Qualified(q).MarshalBinary()
}

type QualifiedContent Qualified

func (q QualifiedContent) MarshalBinary() ([]byte, error) {
	return Qualified(q).MarshalBinary()
}

type QualifiedKey Qualified

func (q QualifiedKey) MarshalBinary() ([]byte, error) {
	return Qualified(q).MarshalBinary()
}

type QualifiedSignature Qualified

func (q QualifiedSignature) MarshalBinary() ([]byte, error) {
	return Qualified(q).MarshalBinary()
}

// generic node
type Node struct {
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

func (n Node) WriteCommonFieldsInto(w io.Writer) error {
	// this slice defines the order in which the fields are written
	return MarshalAllInto(w,
		n.Version,
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
	// this is a template method. It always writes the header fields,
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

func (i Identity) MarshalBinary() ([]byte, error) {
	// define how to serialize this node type's fields
	i.Node.WriteNodeTypeFieldsInto = func(w io.Writer) error {
		return MarshalAllInto(w, i.Name, i.PublicKey)
	}
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
