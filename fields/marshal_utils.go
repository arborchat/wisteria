package fields

import (
	"encoding"
	"io"
)

type BidirectionalBinaryMarshaler interface {
	encoding.BinaryMarshaler
	ProgressiveBinaryUnmarshaler
}

// ProgressiveBinaryUnmarshaler is a type that fully describes how to unmarshal itself
// from a stream of bytes.
type ProgressiveBinaryUnmarshaler interface {
	encoding.BinaryUnmarshaler
	// BytesConsumed can be called after UnmarshalBinary to determine how many bytes of the input to
	// UnmarshalBinary were consumed in the creation of this type.
	BytesConsumed() int
}

func AsMarshaler(in []BidirectionalBinaryMarshaler) []encoding.BinaryMarshaler {
	out := make([]encoding.BinaryMarshaler, len(in))
	for i, f := range in {
		out[i] = encoding.BinaryMarshaler(f)
	}
	return out
}

func AsUnmarshaler(in []BidirectionalBinaryMarshaler) []ProgressiveBinaryUnmarshaler {
	out := make([]ProgressiveBinaryUnmarshaler, len(in))
	for i, f := range in {
		out[i] = ProgressiveBinaryUnmarshaler(f)
	}
	return out
}

func TotalBytesConsumed(in ...BidirectionalBinaryMarshaler) int {
	total := 0
	for _, unmarshaler := range in {
		total += unmarshaler.BytesConsumed()
	}
	return total
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

func UnmarshalAll(b []byte, unmarshalers ...ProgressiveBinaryUnmarshaler) ([]byte, error) {
	currentBytesConsumed := 0
	for _, unmarshaler := range unmarshalers {
		byteSubrange := b[currentBytesConsumed:]
		if err := unmarshaler.UnmarshalBinary(byteSubrange); err != nil {
			return nil, err
		}
		currentBytesConsumed += unmarshaler.BytesConsumed()
	}
	return b[currentBytesConsumed:], nil
}
