package forest_test

import (
	"bytes"
	"encoding"
	"encoding/json"
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

func TestNewArborSerializerSymmetry(t *testing.T) {
	identity, _, community, reply := MakeReplyOrSkip(t)
	identity2, _, community2, reply2 := MakeReplyOrSkip(t)
	nodes := []forest.Node{identity, community, reply}
	outNodes := []forest.Node{identity2, community2, reply2}
	for i, node := range nodes {
		node2 := outNodes[i]
		data, err := forest.ArborSerialize(reflect.ValueOf(node))
		if err != nil {
			t.Errorf("Failed to serialize tagged node: %s", err)
		}
		excess, err := forest.ArborDeserialize(reflect.ValueOf(node2), data)
		if err != nil {
			t.Errorf("Failed to deserialize tagged node: %s", err)
		}
		if len(excess) != 0 {
			t.Errorf("Expected 0 bytes of excess data, got %d", len(excess))
		}
		if !node.Equals(node2) {
			json1, _ := json.Marshal(node)
			json2, _ := json.Marshal(node2)
			t.Errorf("Expected %s and %s to be equal", json1, json2)
		}
	}
}
