package forest

import (
	"bytes"
	"fmt"
)

// generic descriptor
type descriptor struct {
	Type   genericType
	Length ContentLength
}

const sizeofDescriptor = sizeofgenericType + sizeofContentLength

func newDescriptor(t genericType, length int) (*descriptor, error) {
	if length > MaxContentLength {
		return nil, fmt.Errorf("Cannot represent content of length %d, max is %d", length, MaxContentLength)
	}
	d := descriptor{}
	d.Type = t
	d.Length = ContentLength(length)
	return &d, nil
}

func (d descriptor) MarshalBinary() ([]byte, error) {
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

func (d *descriptor) UnmarshalBinary(b []byte) error {
	if len(b) != sizeofDescriptor {
		return fmt.Errorf("Expected %d bytes, got %d", sizeofDescriptor, len(b))
	}
	if err := (&d.Type).UnmarshalBinary(b[:sizeofgenericType]); err != nil {
		return err
	}
	if err := (&d.Length).UnmarshalBinary(b[sizeofgenericType:]); err != nil {
		return err
	}

	return nil
}

// concrete descriptors
type HashDescriptor descriptor

const sizeofHashDescriptor = sizeofDescriptor

func NewHashDescriptor(t HashType, length int) (*HashDescriptor, error) {
	d, err := newDescriptor(genericType(t), length)
	return (*HashDescriptor)(d), err
}

func (d HashDescriptor) MarshalBinary() ([]byte, error) {
	return descriptor(d).MarshalBinary()
}

func (d *HashDescriptor) UnmarshalBinary(b []byte) error {
	return (*descriptor)(d).UnmarshalBinary(b)
}

type ContentDescriptor descriptor

const sizeofContentDescriptor = sizeofDescriptor

func NewContentDescriptor(t ContentType, length int) (*ContentDescriptor, error) {
	d, err := newDescriptor(genericType(t), length)
	return (*ContentDescriptor)(d), err
}

func (d ContentDescriptor) MarshalBinary() ([]byte, error) {
	return descriptor(d).MarshalBinary()
}

func (d *ContentDescriptor) UnmarshalBinary(b []byte) error {
	return (*descriptor)(d).UnmarshalBinary(b)
}

type SignatureDescriptor descriptor

const sizeofSignatureDescriptor = sizeofDescriptor

func NewSignatureDescriptor(t SignatureType, length int) (*SignatureDescriptor, error) {
	d, err := newDescriptor(genericType(t), length)
	return (*SignatureDescriptor)(d), err
}

func (d SignatureDescriptor) MarshalBinary() ([]byte, error) {
	return descriptor(d).MarshalBinary()
}

func (d *SignatureDescriptor) UnmarshalBinary(b []byte) error {
	return (*descriptor)(d).UnmarshalBinary(b)
}

type KeyDescriptor descriptor

const sizeofKeyDescriptor = sizeofDescriptor

func NewKeyDescriptor(t KeyType, length int) (*KeyDescriptor, error) {
	d, err := newDescriptor(genericType(t), length)
	return (*KeyDescriptor)(d), err
}

func (d KeyDescriptor) MarshalBinary() ([]byte, error) {
	return descriptor(d).MarshalBinary()
}

func (d *KeyDescriptor) UnmarshalBinary(b []byte) error {
	return (*descriptor)(d).UnmarshalBinary(b)
}
