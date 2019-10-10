package forest_test

import (
	"testing"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
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
	// create three test nodes, one of each type
	identity, _, community, reply := MakeReplyOrSkip(t)
	nodes := []forest.Node{identity, community, reply}

	// create a set of functions that perform different "Get" operations on nodes
	getFuncs := map[string]func(*fields.QualifiedHash) (forest.Node, bool, error){
		"get":       s.Get,
		"identity":  s.GetIdentity,
		"community": s.GetCommunity,
		"conversation": func(id *fields.QualifiedHash) (forest.Node, bool, error) {
			return s.GetConversation(community.ID(), id)
		},
		"reply": func(id *fields.QualifiedHash) (forest.Node, bool, error) {
			return s.GetReply(community.ID(), reply.ID(), id)
		},
	}

	// ensure no getter functions succeed on an empty store
	for _, i := range nodes {
		for _, get := range getFuncs {
			if node, has, err := get(i.ID()); has {
				t.Errorf("Empty %s should not contain element %v", storeImplName, i.ID())
			} else if err != nil {
				t.Errorf("Empty %s Get() should not err with %s", storeImplName, err)
			} else if node != nil {
				t.Errorf("Empty %s Get() should return none-nil node %v", storeImplName, node)
			}
		}
	}

	// add each node
	for count, i := range nodes {
		if err := s.Add(i); err != nil {
			t.Errorf("%s Add() should not err on Add(): %s", storeImplName, err)
		}
		if size, err := s.Size(); err != nil {
			t.Errorf("%s Size() should never error, got %s", storeImplName, err)
		} else if size != count+1 {
			t.Errorf("%s Size() should be %d after %d Add()s, got %d", storeImplName, count+1, count+1, size)
		}
	}

	// map each node to the getters that should be successful in fetching it
	nodesToGetters := []struct {
		forest.Node
		funcs []string
	}{
		{identity, []string{"get", "identity"}},
		{community, []string{"get", "community"}},
		{reply, []string{"get", "conversation", "reply"}},
	}

	// ensure all getters work for each node
	for _, getterConfig := range nodesToGetters {
		currentNode := getterConfig.Node
		for _, getterName := range getterConfig.funcs {
			if node, has, err := getFuncs[getterName](currentNode.ID()); !has {
				t.Errorf("%s should contain element %v", storeImplName, currentNode.ID())
			} else if err != nil {
				t.Errorf("%s Has() should not err with %s", storeImplName, err)
			} else if !currentNode.Equals(node) {
				t.Errorf("%s Get() should return a node equal to the one that was Add()ed. Got %v, expected %v", storeImplName, node, currentNode)
			}
		}
	}

	// map nodes to the children that they ought to have within the store
	nodesToChildren := []struct {
		forest.Node
		children []*fields.QualifiedHash
	}{
		{identity, []*fields.QualifiedHash{}},
		{community, []*fields.QualifiedHash{reply.ID()}},
		{reply, []*fields.QualifiedHash{}},
	}

	// check each node has its proper children
	for _, childConfig := range nodesToChildren {
		if children, err := s.Children(childConfig.ID()); err != nil {
			t.Errorf("%s should not error fetching children of %v", storeImplName, childConfig.ID())
		} else {
			for _, child := range childConfig.children {
				if !containsID(children, child) {
					t.Errorf("%s should have %v as a child of %v", storeImplName, child, childConfig.ID())
				}
			}
		}
	}

	// add some more nodes so that we can test the Recent method
	identity2, _, community2, reply2 := MakeReplyOrSkip(t)
	for _, i := range []forest.Node{identity2, community2, reply2} {
		if err := s.Add(i); err != nil {
			t.Errorf("%s Add() should not err on Add(): %s", storeImplName, err)
		}
	}
	// try recent on each node type and ensure that it returns the right
	// number and order of results
	type recentRun struct {
		fields.NodeType
		atZero forest.Node
		atOne  forest.Node
	}
	for _, run := range []recentRun{
		{fields.NodeTypeIdentity, identity2, identity},
		{fields.NodeTypeCommunity, community2, community},
		{fields.NodeTypeReply, reply2, reply},
	} {
		recentNodes, err := s.Recent(run.NodeType, 1)
		if err != nil {
			t.Errorf("Recent failed on valid input: %v", err)
		} else if len(recentNodes) < 1 {
			t.Errorf("Recent on store with data returned too few results")
		} else if !recentNodes[0].Equals(run.atZero) {
			t.Errorf("Expected most recent node to be the newly created one")
		}
		recentNodes, err = s.Recent(run.NodeType, 2)
		if err != nil {
			t.Errorf("Recent failed on valid input: %v", err)
		} else if len(recentNodes) < 2 {
			t.Errorf("Recent on store with data returned too few results")
		} else if !recentNodes[0].Equals(run.atZero) {
			t.Errorf("Expected most recent node to be the newly created one")
		} else if !recentNodes[1].Equals(run.atOne) {
			t.Errorf("Expected first node to be the older one")
		}
		recentNodes, err = s.Recent(run.NodeType, 3)
		if err != nil {
			t.Errorf("Recent failed on valid input: %v", err)
		} else if len(recentNodes) > 2 {
			t.Errorf("Recent on store with only two matching nodes returned more than 2 results")
		}
	}
}

func containsID(ids []*fields.QualifiedHash, id *fields.QualifiedHash) bool {
	for _, current := range ids {
		if current.Equals(id) {
			return true
		}
	}
	return false
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
