package main

import (
	"fmt"
	"sort"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

// ReplyList holds a sortable list of replies
type ReplyList []*forest.Reply

func (h ReplyList) Sort() {
	sort.SliceStable(h, func(i, j int) bool {
		return h[i].Created < h[j].Created
	})
}

// IndexForID returns the position of the node with the given `id` inside of the ReplyList,
// or -1 if it is not present.
func (h ReplyList) IndexForID(id *fields.QualifiedHash) int {
	for i, n := range h {
		if n.ID().Equals(id) {
			return i
		}
	}
	return -1
}

// Archive manages a group of known arbor nodes
type Archive struct {
	ReplyList
	forest.Store
}

const defaultArchiveReplyListLen = 1024

func NewArchive(store forest.Store) (*Archive, error) {
	archive := &Archive{
		ReplyList: []*forest.Reply{},
		Store:     store,
	}
	nodes, err := store.Recent(fields.NodeTypeReply, defaultArchiveReplyListLen)
	if err != nil {
		return nil, fmt.Errorf("Failed loading most recent messages: %w", err)
	}
	for _, n := range nodes {
		if r, ok := n.(*forest.Reply); ok {
			archive.ReplyList = append(archive.ReplyList, r)
		}
	}
	return archive, nil
}

// Add accepts an arbor node and stores it in the Archive. If it is
// a Reply node, it will be added to the ReplyList
func (a *Archive) Add(node forest.Node) error {
	if _, has, _ := a.Store.Get(node.ID()); has {
		return fmt.Errorf("Archive already contains %v", node)
	}
	if err := a.Store.Add(node); err != nil {
		return err
	}
	if r, ok := node.(*forest.Reply); ok {
		a.ReplyList = append(a.ReplyList, r)
	}
	return nil

}

// AncestryOf returns the IDs of all known ancestors of the node with the given `id`
func (v *Archive) AncestryOf(id *fields.QualifiedHash) ([]*fields.QualifiedHash, error) {
	node, present, err := v.Store.Get(id)
	if err != nil {
		return nil, err
	} else if !present {
		return []*fields.QualifiedHash{}, nil
	}
	ancestors := make([]*fields.QualifiedHash, 0, node.TreeDepth())
	next := node.ParentID()
	for !next.Equals(fields.NullHash()) {
		parent, present, err := v.Store.Get(next)
		if err != nil {
			return nil, err
		} else if !present {
			return ancestors, nil
		}
		ancestors = append(ancestors, next)
		next = parent.ParentID()
	}
	return ancestors, nil
}

// DescendantsOf returns the IDs of all known descendants of the node with the given `id`
func (v *Archive) DescendantsOf(id *fields.QualifiedHash) ([]*fields.QualifiedHash, error) {
	descendants := make([]*fields.QualifiedHash, 0)
	directChildren := []*fields.QualifiedHash{id}

	for len(directChildren) > 0 {
		target := directChildren[0]
		directChildren = directChildren[1:]
		for _, node := range v.ReplyList {
			if node.ParentID().Equals(target) {
				descendants = append(descendants, node.ID())
				directChildren = append(directChildren, node.ID())
			}
		}
	}
	return descendants, nil
}
