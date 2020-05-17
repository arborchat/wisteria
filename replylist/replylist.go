package replylist

import (
	"fmt"
	"sort"
	"sync"

	forest "git.sr.ht/~whereswaldon/forest-go"
	"git.sr.ht/~whereswaldon/forest-go/fields"
	"git.sr.ht/~whereswaldon/forest-go/store"
)

// ReplyList holds a sortable list of replies that can update itself
// automatically by subscribing to a store.ExtendedStore
type ReplyList struct {
	sync.RWMutex
	replies []*forest.Reply
}

// New creates a ReplyList and subscribes it to the provided ExtendedStore.
// It will prepopulate the list with the contents of the store as well.
func New(s store.ExtendedStore) (*ReplyList, error) {
	rl := new(ReplyList)
	err := rl.SubscribeTo(s)
	if err != nil {
		return nil, err
	}
	return rl, nil

}

// SubscribeTo makes this ReplyList watch a particular ExtendedStore. You
// shouldn't need to do this often, as the New() function does this for
// you if you construct the ReplyList that way.
func (r *ReplyList) SubscribeTo(s store.ExtendedStore) error {
	s.SubscribeToNewMessages(func(node forest.Node) {
		// cannot block in subscription
		go func() {
			r.Lock()
			defer r.Unlock()
			if reply, ok := node.(*forest.Reply); ok {
				alreadyInList := false
				for _, element := range r.replies {
					if element.Equals(reply) {
						alreadyInList = true
						break
					}
				}
				if !alreadyInList {
					r.replies = append(r.replies, reply)
				}
			}
		}()
	})
	const defaultArchiveReplyListLen = 1024

	// prepopulate the ReplyList
	nodes, err := s.Recent(fields.NodeTypeReply, defaultArchiveReplyListLen)
	if err != nil {
		return fmt.Errorf("Failed loading most recent messages: %w", err)
	}
	for _, n := range nodes {
		if reply, ok := n.(*forest.Reply); ok {
			r.replies = append(r.replies, reply)
		}
	}
	r.Sort()
	return nil
}

func (r *ReplyList) Sort() {
	r.Lock()
	defer r.Unlock()
	sort.SliceStable(r.replies, func(i, j int) bool {
		return r.replies[i].Created < r.replies[j].Created
	})
}

// IndexForID returns the position of the node with the given `id` inside of the ReplyList,
// or -1 if it is not present.
func (r *ReplyList) IndexForID(id *fields.QualifiedHash) int {
	r.RLock()
	defer r.RUnlock()
	for i, n := range r.replies {
		if n.ID().Equals(id) {
			return i
		}
	}
	return -1
}

// WithReplies executes an arbitrary closure with access to the replies stored
// inside of the ReplyList. The closure must not modify the slice that it is
// given.
func (r *ReplyList) WithReplies(closure func(replies []*forest.Reply)) {
	r.RLock()
	defer r.RUnlock()
	closure(r.replies)
}
