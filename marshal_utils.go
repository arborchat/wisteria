package forest

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
	// SizeConstraints tells unmarshal logic how many bytes should be fed into the UnmarshalBinary method.
	// If the second field is true, all known data should be provided to allow the type to unmarshal its
	// variable-length payload. Then BytesConsumed() can be called to determine the total number of bytes
	// that unmarshaling the type consumed from the input.
	SizeConstraints() (staticBytes int, variableLength bool)
	// BytesConsumed can be called after UnmarshalBinary to determine how many bytes of the input to
	// UnmarshalBinary were consumed in the creation of this type.
	BytesConsumed() int
}

func asMarshaler(in []BidirectionalBinaryMarshaler) []encoding.BinaryMarshaler {
	out := make([]encoding.BinaryMarshaler, len(in))
	for i, f := range in {
		out[i] = encoding.BinaryMarshaler(f)
	}
	return out
}

func asUnmarshaler(in []BidirectionalBinaryMarshaler) []ProgressiveBinaryUnmarshaler {
	out := make([]ProgressiveBinaryUnmarshaler, len(in))
	for i, f := range in {
		out[i] = ProgressiveBinaryUnmarshaler(f)
	}
	return out
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
		knownSize, variableSize := unmarshaler.SizeConstraints()
		byteSubrange := b[currentBytesConsumed : currentBytesConsumed+knownSize]
		if variableSize {
			byteSubrange = b[currentBytesConsumed:]
		}
		if err := unmarshaler.UnmarshalBinary(byteSubrange); err != nil {
			return nil, err
		}
		if variableSize {
			knownSize = unmarshaler.BytesConsumed()
		}
		currentBytesConsumed += knownSize
	}
	return b[currentBytesConsumed:], nil
}
