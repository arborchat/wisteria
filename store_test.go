package forest_test

import (
	"testing"

	"git.sr.ht/~whereswaldon/forest-go"
)

func TestMemoryStoreAdd(t *testing.T) {
	s := forest.NewMemoryStore()
	if size, err := s.Size(); size != 0 {
		t.Errorf("Expected new store to have size 0, had %d", size)
	} else if err != nil {
		t.Errorf("Expected new store Size() to succeed, failed with %s", err)
	}
	id, _, com, rep := MakeReplyOrSkip(t)
	nodes := []forest.Node{id, com, rep}
	for _, i := range nodes {
		if has, err := s.Has(i.ID()); has {
			t.Errorf("Empty store should not contain element %v", i.ID())
		} else if err != nil {
			t.Errorf("Empty store Has() should not err with %s", err)
		}
		if _, err := s.Get(i.ID()); err == nil {
			t.Errorf("Empty store Get() should err")
		}
	}
	for _, i := range nodes {
		if err := s.Add(i); err != nil {
			t.Errorf("MemoryStore Add() should not err on Add(): %s", err)
		}
		if has, err := s.Has(i.ID()); !has {
			t.Errorf("MemoryStore should contain element %v", i.ID())
		} else if err != nil {
			t.Errorf("MemoryStore Has() should not err with %s", err)
		}
		if node, err := s.Get(i.ID()); err != nil {
			t.Errorf("MemoryStore Get() should not err with %s", err)
		} else if !i.Equals(node) {
			t.Errorf("MemoryStore Get() should return a node equal to the one that was Add()ed. Got %v, expected %v", node, i)
		}
	}
}
