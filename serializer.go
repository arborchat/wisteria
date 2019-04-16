package forest

import "bytes"

// serializer is a type that can describe how to serialize and deserialize itself
type serializer interface {
	serializationOrder() []BidirectionalBinaryMarshaler
}

func MarshalBinary(s serializer) ([]byte, error) {
	buf := new(bytes.Buffer)
	if err := MarshalAllInto(buf, asMarshaler(s.serializationOrder())...); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func UnmarshalBinary(s serializer, b []byte) error {
	if _, err := UnmarshalAll(b, asUnmarshaler(s.serializationOrder())...); err != nil {
		return err
	}
	return nil
}

func BytesConsumed(s serializer) int {
	return totalBytesConsumed(s.serializationOrder()...)
}
