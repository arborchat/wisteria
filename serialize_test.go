package forest_test

import (
	"bytes"
	"encoding"
	"reflect"
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
)

func TestNewArborSerializer(t *testing.T) {
	identity, _, community, reply := MakeReplyOrSkip(t)
	nodes := []encoding.BinaryMarshaler{identity, community, reply}
	for _, node := range nodes {
		data, err := forest.ArborSerialize(reflect.ValueOf(node))
		if err != nil {
			t.Errorf("Failed to serialize node with tags: %s", err)
		}
		data2, err := node.MarshalBinary()
		if err != nil {
			t.Errorf("Failed to serialize node the old way: %s", err)
		}
		if !bytes.Equal(data, data2) {
			t.Errorf("Expected\n%v\nand\n%v\nto be the same", data, data2)
		}
	}
}
