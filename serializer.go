package forest

import (
	"bytes"

	"git.sr.ht/~whereswaldon/forest-go/fields"
)

// serializer is a type that can describe how to serialize and deserialize itself
type serializer interface {
	SerializationOrder() []fields.BidirectionalBinaryMarshaler
}

func MarshalBinary(s serializer) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := fields.MarshalAllInto(buf, fields.AsMarshaler(s.SerializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnmarshalBinary(s serializer, b []byte) error {
	if _, err := fields.UnmarshalAll(b, fields.AsUnmarshaler(s.SerializationOrder())...); err != nil {
		return err
	}
	return nil
}

func BytesConsumed(s serializer) int {
	return fields.TotalBytesConsumed(s.SerializationOrder()...)
}
