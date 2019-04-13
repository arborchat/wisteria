package forest

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

const (
	// CurrentVersion is the Forest version that this library writes
	CurrentVersion Version = 1

	// HashDigestLengthSHA512_256 is the length of the digest produced by the SHA512/256 hash algorithm
	HashDigestLengthSHA512_256 ContentLength = 32
)

// multiByteSerializationOrder defines the order in which multi-byte
// integers are serialized into binary
var multiByteSerializationOrder binary.ByteOrder = binary.BigEndian

// fundamental types
type genericType uint8

const sizeofgenericType = 1

func (g genericType) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	err := binary.Write(b, multiByteSerializationOrder, g)
	return b.Bytes(), err
}

func (g *genericType) UnmarshalBinary(b []byte) error {
	buf := bytes.NewBuffer(b)
	return binary.Read(buf, multiByteSerializationOrder, g)
}

// ContentLength represents the length of a piece of data in the Forest
type ContentLength uint16

const sizeofContentLength = 2

const (
	// MaxContentLength is the maximum representable content length in this
	// version of the Forest
	MaxContentLength = math.MaxUint16
)

// MarshalBinary converts the ContentLength into its binary representation
func (c ContentLength) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	err := binary.Write(b, multiByteSerializationOrder, c)
	return b.Bytes(), err
}

// UnmarshalBinary converts from the binary representation of a ContentLength
// back to its structured form
func (c *ContentLength) UnmarshalBinary(b []byte) error {
	buf := bytes.NewBuffer(b)
	return binary.Read(buf, multiByteSerializationOrder, c)
}

// TreeDepth represents the depth of a node within a tree
type TreeDepth uint32

// MarshalBinary converts the TreeDepth into its binary representation
func (t TreeDepth) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	err := binary.Write(b, multiByteSerializationOrder, t)
	return b.Bytes(), err
}

// UnmarshalBinary converts from the binary representation of a TreeDepth
// back to its structured form
func (t *TreeDepth) UnmarshalBinary(b []byte) error {
	buf := bytes.NewBuffer(b)
	return binary.Read(buf, multiByteSerializationOrder, t)
}

// Value represents a quantity of arbitrary binary data in the Forest
type Value []byte

// MarshalBinary converts the Value into its binary representation
func (v Value) MarshalBinary() ([]byte, error) {
	return v, nil
}

// UnmarshalBinary converts from the binary representation of a Value
// back to its structured form
func (v *Value) UnmarshalBinary(b []byte) error {
	*v = b
	return nil
}

// Version represents the version of the Arbor Forest Schema used to construct
// a particular node
type Version uint64

// MarshalBinary converts the Version into its binary representation
func (v Version) MarshalBinary() ([]byte, error) {
	b := new(bytes.Buffer)
	err := binary.Write(b, multiByteSerializationOrder, v)
	return b.Bytes(), err
}

// UnmarshalBinary converts from the binary representation of a Version
// back to its structured form
func (v *Version) UnmarshalBinary(b []byte) error {
	buf := bytes.NewBuffer(b)
	return binary.Read(buf, multiByteSerializationOrder, v)
}

// specialized types
type NodeType genericType

const (
	NodeTypeIdentity     NodeType = 1
	NodeTypeCommunity    NodeType = 2
	NodeTypeConversation NodeType = 3
	NodeTypeReply        NodeType = 4
)

var validNodeTypes = map[NodeType]struct{}{
	NodeTypeIdentity:     struct{}{},
	NodeTypeCommunity:    struct{}{},
	NodeTypeConversation: struct{}{},
	NodeTypeReply:        struct{}{},
}

func (t NodeType) MarshalBinary() ([]byte, error) {
	return genericType(t).MarshalBinary()
}

func (t *NodeType) UnmarshalBinary(b []byte) error {
	if err := (*genericType)(t).UnmarshalBinary(b); err != nil {
		return err
	}
	if _, valid := validNodeTypes[*t]; !valid {
		return fmt.Errorf("%d is not a valid node type", *t)
	}
	return nil
}

type HashType genericType

const (
	HashTypeNullHash   HashType = 0
	HashTypeSHA512_256 HashType = 1
)

var validHashTypes = map[HashType]struct{}{
	HashTypeNullHash:   struct{}{},
	HashTypeSHA512_256: struct{}{},
}

func (t HashType) MarshalBinary() ([]byte, error) {
	return genericType(t).MarshalBinary()
}

func (t *HashType) UnmarshalBinary(b []byte) error {
	if err := (*genericType)(t).UnmarshalBinary(b); err != nil {
		return err
	}
	if _, valid := validHashTypes[*t]; !valid {
		return fmt.Errorf("%d is not a valid hash type", *t)
	}
	return nil
}

type ContentType genericType

const (
	ContentTypeUTF8String ContentType = 1
	ContentTypeJSON       ContentType = 2
)

var validContentTypes = map[ContentType]struct{}{
	ContentTypeUTF8String: struct{}{},
	ContentTypeJSON:       struct{}{},
}

func (t ContentType) MarshalBinary() ([]byte, error) {
	return genericType(t).MarshalBinary()
}

func (t *ContentType) UnmarshalBinary(b []byte) error {
	if err := (*genericType)(t).UnmarshalBinary(b); err != nil {
		return err
	}
	if _, valid := validContentTypes[*t]; !valid {
		return fmt.Errorf("%d is not a valid content type", *t)
	}
	return nil
}

type KeyType genericType

const (
	KeyTypeNoKey   KeyType = 0
	KeyTypeOpenPGP KeyType = 1
)

var validKeyTypes = map[KeyType]struct{}{
	KeyTypeNoKey:   struct{}{},
	KeyTypeOpenPGP: struct{}{},
}

func (t KeyType) MarshalBinary() ([]byte, error) {
	return genericType(t).MarshalBinary()
}

func (t *KeyType) UnmarshalBinary(b []byte) error {
	if err := (*genericType)(t).UnmarshalBinary(b); err != nil {
		return err
	}
	if _, valid := validKeyTypes[*t]; !valid {
		return fmt.Errorf("%d is not a valid key type", *t)
	}
	return nil
}

type SignatureType genericType

const (
	SignatureTypeOpenPGP SignatureType = 1
)

var validSignatureTypes = map[SignatureType]struct{}{
	SignatureTypeOpenPGP: struct{}{},
}

func (t SignatureType) MarshalBinary() ([]byte, error) {
	return genericType(t).MarshalBinary()
}

func (t *SignatureType) UnmarshalBinary(b []byte) error {
	if err := (*genericType)(t).UnmarshalBinary(b); err != nil {
		return err
	}
	if _, valid := validSignatureTypes[*t]; !valid {
		return fmt.Errorf("%d is not a valid signature type", *t)
	}
	return nil
}
