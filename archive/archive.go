package archive

import (
	"fmt"
	"sort"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/forest-go/store"
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
	*store.Archive

	ReplyList
}

const defaultArchiveReplyListLen = 1024

func NewArchive(s forest.Store) (*Archive, error) {
	archive := &Archive{
		ReplyList: []*forest.Reply{},
		Archive:   store.NewArchive(s),
	}

	// insert new messages into the ReplyList as they're added
	archive.Archive.SubscribeToNewMessages(archive.tryListInsert)

	// prepopulate the ReplyList
	nodes, err := s.Recent(fields.NodeTypeReply, defaultArchiveReplyListLen)
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

func (a *Archive) tryListInsert(node forest.Node) {
	if r, ok := node.(*forest.Reply); ok {
		alreadyInList := false
		for _, element := range a.ReplyList {
			if element.Equals(r) {
				alreadyInList = true
				break
			}
		}
		if !alreadyInList {
			a.ReplyList = append(a.ReplyList, r)
		}
	}
}
