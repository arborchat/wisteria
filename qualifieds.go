package forest

import (
	"bytes"
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
