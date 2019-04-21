package forest

import "bytes"

const minSizeofQualified = sizeofDescriptor

// concrete qualified data types
type QualifiedHash struct {
	Descriptor HashDescriptor
	Value      Value
}

const minSizeofQualifiedHash = sizeofHashDescriptor

// NewQualifiedHash returns a valid QualifiedHash from the given data
func NewQualifiedHash(t HashType, content []byte) (*QualifiedHash, error) {
	hd, err := NewHashDescriptor(t, len(content))
	if err != nil {
		return nil, err
	}
	return &QualifiedHash{*hd, Value(content)}, nil
}

func NullHash() *QualifiedHash {
	return &QualifiedHash{
		Descriptor: HashDescriptor{
			Type:   HashTypeNullHash,
			Length: 0,
		},
		Value: []byte{},
	}
}

func (q *QualifiedHash) UnmarshalBinary(b []byte) error {
	unused, err := UnmarshalAll(b, asUnmarshaler(q.Descriptor.serializationOrder())...)
	if err != nil {
		return err
	}
	if err := q.Value.UnmarshalBinary(unused[:q.Descriptor.Length]); err != nil {
		return err
	}
	return nil
}

func (q *QualifiedHash) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := MarshalAllInto(buf, asMarshaler(q.serializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (q *QualifiedHash) BytesConsumed() int {
	return totalBytesConsumed(q.serializationOrder()...)
}

func (q *QualifiedHash) serializationOrder() []BidirectionalBinaryMarshaler {
	return append(q.Descriptor.serializationOrder(), &q.Value)
}

func (q *QualifiedHash) Equals(other *QualifiedHash) bool {
	return q.Descriptor.Equals(&other.Descriptor) && q.Value.Equals(&other.Value)
}

type QualifiedContent struct {
	Descriptor ContentDescriptor
	Value      Value
}

const minSizeofQualifiedContent = sizeofContentDescriptor

// NewQualifiedContent returns a valid QualifiedContent from the given data
func NewQualifiedContent(t ContentType, content []byte) (*QualifiedContent, error) {
	hd, err := NewContentDescriptor(t, len(content))
	if err != nil {
		return nil, err
	}
	return &QualifiedContent{*hd, Value(content)}, nil
}

func (q *QualifiedContent) serializationOrder() []BidirectionalBinaryMarshaler {
	return append(q.Descriptor.serializationOrder(), &q.Value)
}

func (q *QualifiedContent) Equals(other *QualifiedContent) bool {
	return q.Descriptor.Equals(&other.Descriptor) && q.Value.Equals(&other.Value)
}

func (q *QualifiedContent) UnmarshalBinary(b []byte) error {
	unused, err := UnmarshalAll(b, asUnmarshaler(q.Descriptor.serializationOrder())...)
	if err != nil {
		return err
	}
	if err := q.Value.UnmarshalBinary(unused[:q.Descriptor.Length]); err != nil {
		return err
	}
	return nil
}

func (q *QualifiedContent) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := MarshalAllInto(buf, asMarshaler(q.serializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (q *QualifiedContent) BytesConsumed() int {
	return totalBytesConsumed(q.serializationOrder()...)
}

type QualifiedKey struct {
	Descriptor KeyDescriptor
	Value      Value
}

const minSizeofQualifiedKey = sizeofKeyDescriptor

// NewQualifiedKey returns a valid QualifiedKey from the given data
func NewQualifiedKey(t KeyType, content []byte) (*QualifiedKey, error) {
	hd, err := NewKeyDescriptor(t, len(content))
	if err != nil {
		return nil, err
	}
	return &QualifiedKey{*hd, Value(content)}, nil
}

func (q *QualifiedKey) serializationOrder() []BidirectionalBinaryMarshaler {
	return append(q.Descriptor.serializationOrder(), &q.Value)
}

func (q *QualifiedKey) Equals(other *QualifiedKey) bool {
	return q.Descriptor.Equals(&other.Descriptor) && q.Value.Equals(&other.Value)
}

func (q *QualifiedKey) UnmarshalBinary(b []byte) error {
	unused, err := UnmarshalAll(b, asUnmarshaler(q.Descriptor.serializationOrder())...)
	if err != nil {
		return err
	}
	if err := q.Value.UnmarshalBinary(unused[:q.Descriptor.Length]); err != nil {
		return err
	}
	return nil
}

func (q *QualifiedKey) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := MarshalAllInto(buf, asMarshaler(q.serializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (q *QualifiedKey) BytesConsumed() int {
	return totalBytesConsumed(q.serializationOrder()...)
}

type QualifiedSignature struct {
	Descriptor SignatureDescriptor
	Value      Value
}

const minSizeofQualifiedSignature = sizeofSignatureDescriptor

// NewQualifiedSignature returns a valid QualifiedSignature from the given data
func NewQualifiedSignature(t SignatureType, content []byte) (*QualifiedSignature, error) {
	hd, err := NewSignatureDescriptor(t, len(content))
	if err != nil {
		return nil, err
	}
	return &QualifiedSignature{*hd, Value(content)}, nil
}

func (q *QualifiedSignature) serializationOrder() []BidirectionalBinaryMarshaler {
	return append(q.Descriptor.serializationOrder(), &q.Value)
}

func (q *QualifiedSignature) Equals(other *QualifiedSignature) bool {
	return q.Descriptor.Equals(&other.Descriptor) && q.Value.Equals(&other.Value)
}

func (q *QualifiedSignature) UnmarshalBinary(b []byte) error {
	unused, err := UnmarshalAll(b, asUnmarshaler(q.Descriptor.serializationOrder())...)
	if err != nil {
		return err
	}
	if err := q.Value.UnmarshalBinary(unused[:q.Descriptor.Length]); err != nil {
		return err
	}
	return nil
}

func (q *QualifiedSignature) MarshalBinary() ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := MarshalAllInto(buf, asMarshaler(q.serializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (q *QualifiedSignature) BytesConsumed() int {
	return totalBytesConsumed(q.serializationOrder()...)
}
