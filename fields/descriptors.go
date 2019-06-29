package fields

import (
	"bytes"
	"encoding"
	"fmt"
)

const sizeofDescriptor = sizeofgenericType + sizeofContentLength

func marshalTextDescriptor(descriptorType encoding.TextMarshaler, length encoding.TextMarshaler) ([]byte, error) {
	buf := new(bytes.Buffer)
	b, err := descriptorType.MarshalText()
	if err != nil {
		return nil, err
	}
	_, _ = buf.Write(b)
	_, _ = buf.Write([]byte("_"))
	b, err = length.MarshalText()
	if err != nil {
		return nil, err
	}
	_, _ = buf.Write(b)
	return buf.Bytes(), nil
}

// concrete descriptors
type HashDescriptor struct {
	Type   HashType      `arbor:"order=0"`
	Length ContentLength `arbor:"order=1"`
}

const sizeofHashDescriptor = sizeofDescriptor

func NewHashDescriptor(t HashType, length int) (*HashDescriptor, error) {
	cLength, err := NewContentLength(length)
	if err != nil {
		return nil, err
	}
	return &HashDescriptor{t, *cLength}, nil
}

func (d *HashDescriptor) Equals(other *HashDescriptor) bool {
	return d.Type.Equals(&other.Type) && d.Length.Equals(&other.Length)
}

func (d *HashDescriptor) MarshalText() ([]byte, error) {
	return marshalTextDescriptor(d.Type, d.Length)
}

func (d *HashDescriptor) Validate() error {
	validLengths, validType := ValidHashTypes[d.Type]
	if !validType {
		return fmt.Errorf("%d is not a valid hash type", d.Type)
	}
	found := false
	for _, length := range validLengths {
		if length == d.Length {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("%d is not a valid hash length for hash type %d", d.Length, d.Type)
	}
	return nil
}

type ContentDescriptor struct {
	Type   ContentType   `arbor:"order=0"`
	Length ContentLength `arbor:"order=1"`
}

const sizeofContentDescriptor = sizeofDescriptor

func NewContentDescriptor(t ContentType, length int) (*ContentDescriptor, error) {
	cLength, err := NewContentLength(length)
	if err != nil {
		return nil, err
	}
	return &ContentDescriptor{t, *cLength}, nil
}

func (d *ContentDescriptor) Equals(other *ContentDescriptor) bool {
	return d.Type.Equals(&other.Type) && d.Length.Equals(&other.Length)
}

func (d *ContentDescriptor) MarshalText() ([]byte, error) {
	return marshalTextDescriptor(d.Type, d.Length)
}

func (d *ContentDescriptor) Validate() error {
	_, validType := ValidContentTypes[d.Type]
	if !validType {
		return fmt.Errorf("%d is not a valid content type", d.Type)
	}
	return nil
}

type SignatureDescriptor struct {
	Type   SignatureType `arbor:"order=0"`
	Length ContentLength `arbor:"order=1"`
}

const sizeofSignatureDescriptor = sizeofDescriptor

func NewSignatureDescriptor(t SignatureType, length int) (*SignatureDescriptor, error) {
	cLength, err := NewContentLength(length)
	if err != nil {
		return nil, err
	}
	return &SignatureDescriptor{t, *cLength}, nil
}

func (d *SignatureDescriptor) Equals(other *SignatureDescriptor) bool {
	return d.Type.Equals(&other.Type) && d.Length.Equals(&other.Length)
}

func (d *SignatureDescriptor) MarshalText() ([]byte, error) {
	return marshalTextDescriptor(d.Type, d.Length)
}

func (d *SignatureDescriptor) Validate() error {
	_, validType := ValidSignatureTypes[d.Type]
	if !validType {
		return fmt.Errorf("%d is not a valid signature type", d.Type)
	}
	return nil
}

type KeyDescriptor struct {
	Type   KeyType       `arbor:"order=0"`
	Length ContentLength `arbor:"order=1"`
}

const sizeofKeyDescriptor = sizeofDescriptor

func NewKeyDescriptor(t KeyType, length int) (*KeyDescriptor, error) {
	cLength, err := NewContentLength(length)
	if err != nil {
		return nil, err
	}
	return &KeyDescriptor{t, *cLength}, nil
}

func (d *KeyDescriptor) Equals(other *KeyDescriptor) bool {
	return d.Type.Equals(&other.Type) && d.Length.Equals(&other.Length)
}

func (d *KeyDescriptor) MarshalText() ([]byte, error) {
	return marshalTextDescriptor(d.Type, d.Length)
}

func (d *KeyDescriptor) Validate() error {
	_, validType := ValidKeyTypes[d.Type]
	if !validType {
		return fmt.Errorf("%d is not a valid key type", d.Type)
	}
	return nil
}
