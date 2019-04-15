package forest

import (
	"bytes"
	"encoding/base64"
	"fmt"
)

// generic qualified data
type qualified struct {
	Descriptor descriptor
	Value
}

const minSizeofQualified = sizeofDescriptor

// newQualified creates a valid qualified from the given data
func newQualified(t genericType, content []byte) (*qualified, error) {
	q := qualified{}
	d, err := newDescriptor(t, len(content))
	if err != nil {
		return nil, err
	}
	q.Descriptor = *d
	q.Value = Value(content)
	return &q, nil
}

func (q qualified) MarshalBinary() ([]byte, error) {
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

func (q *qualified) UnmarshalBinary(b []byte) error {
	if len(b) < sizeofDescriptor {
		return fmt.Errorf("Not enough data for qualified type, need at least %d bytes, have %d", sizeofDescriptor, len(b))
	}
	if err := (&q.Descriptor).UnmarshalBinary(b[:sizeofDescriptor]); err != nil {
		return err
	}
	var length = sizeofDescriptor + q.Descriptor.Length
	if err := (&q.Value).UnmarshalBinary(b[sizeofDescriptor:length]); err != nil {
		return err
	}
	return nil
}

func (q qualified) Equals(o qualified) bool {
	return q.Descriptor == o.Descriptor && bytes.Equal([]byte(q.Value), []byte(o.Value))
}

func (q *qualified) SizeConstraints() (int, bool) {
	return minSizeofQualified, true
}

func (q *qualified) BytesConsumed() int {
	return minSizeofQualified + len([]byte(q.Value))
}

// concrete qualified data types
type QualifiedHash qualified

const minSizeofQualifiedHash = sizeofHashDescriptor

// NewQualifiedHash returns a valid QualifiedHash from the given data
func NewQualifiedHash(t HashType, content []byte) (*QualifiedHash, error) {
	q, e := newQualified(genericType(t), content)
	return (*QualifiedHash)(q), e
}

func NullHash() QualifiedHash {
	return QualifiedHash{
		Descriptor: descriptor{
			Type:   genericType(HashTypeNullHash),
			Length: 0,
		},
		Value: []byte{},
	}
}

func (q QualifiedHash) MarshalBinary() ([]byte, error) {
	return qualified(q).MarshalBinary()
}

func (q *QualifiedHash) UnmarshalBinary(b []byte) error {
	return (*qualified)(q).UnmarshalBinary(b)
}

func (q *QualifiedHash) SizeConstraints() (int, bool) {
	return minSizeofQualifiedHash, true
}

func (q *QualifiedHash) BytesConsumed() int {
	return minSizeofQualifiedHash + len([]byte(q.Value))
}

func (q QualifiedHash) MarshalText() ([]byte, error) {
	hashData := base64.StdEncoding.EncodeToString([]byte(q.Value))
	text := fmt.Sprintf("Hash(%s,len:%d):%s", hashNames[HashType(q.Descriptor.Type)], q.Descriptor.Length, hashData)
	return []byte(text), nil
}

type QualifiedContent qualified

const minSizeofQualifiedContent = sizeofContentDescriptor

// NewQualifiedContent returns a valid QualifiedContent from the given data
func NewQualifiedContent(t ContentType, content []byte) (*QualifiedContent, error) {
	q, e := newQualified(genericType(t), content)
	return (*QualifiedContent)(q), e
}

func (q QualifiedContent) MarshalBinary() ([]byte, error) {
	return qualified(q).MarshalBinary()
}

func (q *QualifiedContent) UnmarshalBinary(b []byte) error {
	return (*qualified)(q).UnmarshalBinary(b)
}

func (q *QualifiedContent) SizeConstraints() (int, bool) {
	return minSizeofQualifiedContent, true
}

func (q *QualifiedContent) BytesConsumed() int {
	return minSizeofQualifiedContent + len([]byte(q.Value))
}

func (q QualifiedContent) MarshalText() ([]byte, error) {
	var contentData []byte
	if ContentType(q.Descriptor.Type) == ContentTypeUTF8String || ContentType(q.Descriptor.Type) == ContentTypeJSON {
		contentData = []byte(q.Value)
	} else {
		contentData = []byte(base64.StdEncoding.EncodeToString([]byte(q.Value)))
	}
	text := fmt.Sprintf("Content(%s,len:%d):%s", contentNames[ContentType(q.Descriptor.Type)], q.Descriptor.Length, contentData)
	return []byte(text), nil
}

type QualifiedKey qualified

const minSizeofQualifiedKey = sizeofKeyDescriptor

// NewQualifiedKey returns a valid QualifiedKey from the given data
func NewQualifiedKey(t KeyType, content []byte) (*QualifiedKey, error) {
	q, e := newQualified(genericType(t), content)
	return (*QualifiedKey)(q), e
}

func (q QualifiedKey) MarshalBinary() ([]byte, error) {
	return qualified(q).MarshalBinary()
}

func (q *QualifiedKey) UnmarshalBinary(b []byte) error {
	return (*qualified)(q).UnmarshalBinary(b)
}

func (q *QualifiedKey) SizeConstraints() (int, bool) {
	return minSizeofQualifiedKey, true
}

func (q *QualifiedKey) BytesConsumed() int {
	return minSizeofQualifiedKey + len([]byte(q.Value))
}

func (q QualifiedKey) MarshalText() ([]byte, error) {
	keyData := base64.StdEncoding.EncodeToString([]byte(q.Value))
	text := fmt.Sprintf("Key(%s,len:%d):%s", keyNames[KeyType(q.Descriptor.Type)], q.Descriptor.Length, keyData)
	return []byte(text), nil
}

type QualifiedSignature qualified

const minSizeofQualifiedSignature = sizeofSignatureDescriptor

// NewQualifiedSignature returns a valid QualifiedSignature from the given data
func NewQualifiedSignature(t SignatureType, content []byte) (*QualifiedSignature, error) {
	q, e := newQualified(genericType(t), content)
	return (*QualifiedSignature)(q), e
}

func (q QualifiedSignature) MarshalBinary() ([]byte, error) {
	return qualified(q).MarshalBinary()
}

func (q *QualifiedSignature) UnmarshalBinary(b []byte) error {
	return (*qualified)(q).UnmarshalBinary(b)
}

func (q *QualifiedSignature) SizeConstraints() (int, bool) {
	return minSizeofQualifiedSignature, true
}

func (q *QualifiedSignature) BytesConsumed() int {
	return minSizeofQualifiedSignature + len([]byte(q.Value))
}

func (q QualifiedSignature) MarshalText() ([]byte, error) {
	signatureData := base64.StdEncoding.EncodeToString([]byte(q.Value))
	text := fmt.Sprintf("Signature(%s,len:%d):%s", signatureNames[SignatureType(q.Descriptor.Type)], q.Descriptor.Length, signatureData)
	return []byte(text), nil
}
