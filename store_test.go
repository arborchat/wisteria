package forest_test

import (
	"testing"

	"git.sr.ht/~whereswaldon/forest-go"
)

func TestMemoryStore(t *testing.T) {
	s := forest.NewMemoryStore()
	testStandardStoreInterface(t, s, "MemoryStore")
}

func testStandardStoreInterface(t *testing.T, s forest.Store, storeImplName string) {
	if size, err := s.Size(); size != 0 {
		t.Errorf("Expected new %s to have size 0, had %d", storeImplName, size)
	} else if err != nil {
		t.Errorf("Expected new %s Size() to succeed, failed with %s", storeImplName, err)
	}
	id, _, com, rep := MakeReplyOrSkip(t)
	nodes := []forest.Node{id, com, rep}
	for _, i := range nodes {
		if node, has, err := s.Get(i.ID()); has {
			t.Errorf("Empty %s should not contain element %v", storeImplName, i.ID())
		} else if err != nil {
			t.Errorf("Empty %s Get() should not err with %s", storeImplName, err)
		} else if node != nil {
			t.Errorf("Empty %s Get() should return none-nil node %v", storeImplName, node)
		}
	}
	for count, i := range nodes {
		if err := s.Add(i); err != nil {
			t.Errorf("%s Add() should not err on Add(): %s", storeImplName, err)
		}
		if size, err := s.Size(); err != nil {
			t.Errorf("%s Size() should never error, got %s", storeImplName, err)
		} else if size != count+1 {
			t.Errorf("%s Size() should be %d after %d Add()s, got %d", storeImplName, count+1, count+1, size)
		}
		if node, has, err := s.Get(i.ID()); !has {
			t.Errorf("%s should contain element %v", storeImplName, i.ID())
		} else if err != nil {
			t.Errorf("%s Has() should not err with %s", storeImplName, err)
		} else if !i.Equals(node) {
			t.Errorf("%s Get() should return a node equal to the one that was Add()ed. Got %v, expected %v", storeImplName, node, i)
		}
	}
}

func TestCacheStore(t *testing.T) {
	s1 := forest.NewMemoryStore()
	s2 := forest.NewMemoryStore()
	c, err := forest.NewCacheStore(s1, s2)
	if err != nil {
		t.Errorf("Unexpected error constructing CacheStore: %v", err)
	}
	testStandardStoreInterface(t, c, "CacheStore")
}

func TestCacheStoreDownPropagation(t *testing.T) {
	s1 := forest.NewMemoryStore()
	id, _, com, rep := MakeReplyOrSkip(t)
	nodes := []forest.Node{id, com, rep}
	subrange := nodes[:len(nodes)-1]
	for _, node := range subrange {
		if err := s1.Add(node); err != nil {
			t.Skipf("Failed adding %v to %v", node, s1)
		}
	}
	s2 := forest.NewMemoryStore()
	if _, err := forest.NewCacheStore(s1, s2); err != nil {
		t.Errorf("Unexpected error when constructing CacheStore: %v", err)
	}

	for _, node := range subrange {
		if n2, has, err := s2.Get(node.ID()); err != nil {
			t.Errorf("Unexpected error getting node from cache base layer: %s", err)
		} else if !has {
			t.Errorf("Expected cache base layer to contain %v", node.ID())
		} else if !n2.Equals(node) {
			t.Errorf("Expected cache base layer to contain the same value for ID %v", node.ID())
		}
	}
}

func TestCacheStoreUpPropagation(t *testing.T) {
	base := forest.NewMemoryStore()
	id, _, com, rep := MakeReplyOrSkip(t)
	nodes := []forest.Node{id, com, rep}
	subrange := nodes[:len(nodes)-1]
	for _, node := range subrange {
		if err := base.Add(node); err != nil {
			t.Skipf("Failed adding %v to %v", node, base)
		}
	}
	cache := forest.NewMemoryStore()
	combined, err := forest.NewCacheStore(cache, base)
	if err != nil {
		t.Errorf("Unexpected error when constructing CacheStore: %v", err)
	}

	for _, node := range subrange {
		if _, has, err := cache.Get(node.ID()); err != nil {
			t.Errorf("Unexpected error getting node from cache layer: %s", err)
		} else if has {
			t.Errorf("Expected cache layer not to contain %v", node.ID())
		}
		if n2, has, err := combined.Get(node.ID()); err != nil {
			t.Errorf("Unexpected error getting node from cache store: %s", err)
		} else if !has {
			t.Errorf("Expected cache store to contain %v", node.ID())
		} else if !n2.Equals(node) {
			t.Errorf("Expected cache store to contain the same value for ID %v", node.ID())
		}
		if n2, has, err := cache.Get(node.ID()); err != nil {
			t.Errorf("Unexpected error getting node from cache layer: %s", err)
		} else if !has {
			t.Errorf("Expected cache layer to contain %v after warming cache", node.ID())
		} else if !n2.Equals(node) {
			t.Errorf("Expected cache layer to contain the same value for ID %v after warming cache", node.ID())
		}
	}
}
