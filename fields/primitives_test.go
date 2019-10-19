package fields_test

import (
	"crypto/rand"
	"testing"
	"time"

	"git.sr.ht/~whereswaldon/forest-go/fields"
)

func TestTimeBidirectionalConversion(t *testing.T) {
	current := time.Now()
	asField := fields.TimestampFrom(current)
	back := asField.Time()
	current = current.Truncate(time.Millisecond)
	if !current.Equal(back) {
		t.Errorf("Expected %s to equal %s after converting through fields.Timestamp", back, current)
	}
}

func TestTextMarshalQualifiedHash(t *testing.T) {
	hashLen := 32
	hashData := make([]byte, hashLen)
	_, err := rand.Read(hashData)
	if err != nil {
		t.Skipf("unable to read random data: %v", err)
	}
	q := &fields.QualifiedHash{
		Descriptor: fields.HashDescriptor{
			Type:   fields.HashTypeSHA512,
			Length: fields.ContentLength(hashLen),
		},
		Blob: fields.Blob(hashData),
	}
	asText, err := q.MarshalText()
	if err != nil {
		t.Fatalf("Failed to marshal as text: %v", err)
	}
	out := &fields.QualifiedHash{}
	if err := out.UnmarshalText(asText); err != nil {
		t.Fatalf("Failed to unmarshal valid content: %v", err)
	}

	if !q.Equals(out) {
		t.Fatalf("Input and output do not match:\nin: %v\nout: %v", *q, *out)
	}
}

func TestBlobContains(t *testing.T) {
     b := fields.Blob([]byte("something here"))
     if !b.ContainsString("thing") {
        t.Fatal("ContainsString() found nonexistent string in Blob.")
     }
}
