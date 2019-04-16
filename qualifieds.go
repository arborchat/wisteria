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

func (q *QualifiedHash) serializationOrder() []BidirectionalBinaryMarshaler {
	return append(q.Descriptor.serializationOrder(), &q.Value)
}

func (q *QualifiedHash) Equals(other *QualifiedHash) bool {
	return q.Descriptor.Equals(&other.Descriptor) && bytes.Equal([]byte(q.Value), []byte(other.Value))
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