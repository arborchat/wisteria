package fields_test

import (
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
