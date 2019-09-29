package forest

import (
	"fmt"

	"git.sr.ht/~whereswaldon/forest-go/fields"
)

type Store interface {
	Size() (int, error)
	CopyInto(Store) error
	Get(*fields.QualifiedHash) (Node, bool, error)
	GetIdentity(*fields.QualifiedHash) (Node, bool, error)
	GetCommunity(*fields.QualifiedHash) (Node, bool, error)
	GetConversation(communityID, conversationID *fields.QualifiedHash) (Node, bool, error)
	GetReply(communityID, conversationID, replyID *fields.QualifiedHash) (Node, bool, error)
	Children(*fields.QualifiedHash) ([]*fields.QualifiedHash, error)
	Add(Node) error
}

type MemoryStore struct {
	Items    map[string]Node
	ChildMap map[string][]string
}

var _ Store = &MemoryStore{}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		Items:    make(map[string]Node),
		ChildMap: make(map[string][]string),
	}
}

func (m *MemoryStore) Size() (int, error) {
	return len(m.Items), nil
}

func (m *MemoryStore) CopyInto(other Store) error {
	for _, node := range m.Items {
		if err := other.Add(node); err != nil {
			return err
		}
	}
	return nil
}

func (m *MemoryStore) Get(id *fields.QualifiedHash) (Node, bool, error) {
	idString, err := id.MarshalString()
	if err != nil {
		return nil, false, err
	}
	return m.GetID(idString)
}

func (m *MemoryStore) GetIdentity(id *fields.QualifiedHash) (Node, bool, error) {
	return m.Get(id)
}

func (m *MemoryStore) GetCommunity(id *fields.QualifiedHash) (Node, bool, error) {
	return m.Get(id)
}

func (m *MemoryStore) GetConversation(communityID, conversationID *fields.QualifiedHash) (Node, bool, error) {
	return m.Get(conversationID)
}

func (m *MemoryStore) GetReply(communityID, conversationID, replyID *fields.QualifiedHash) (Node, bool, error) {
	return m.Get(replyID)
}

func (m *MemoryStore) GetID(id string) (Node, bool, error) {
	item, has := m.Items[id]
	return item, has, nil
}

func (m *MemoryStore) Children(id *fields.QualifiedHash) ([]*fields.QualifiedHash, error) {
	idString, err := id.MarshalString()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal node id into key: %w", err)
	}
	children, any := m.ChildMap[idString]
	if !any {
		return []*fields.QualifiedHash{}, nil
	}
	childIDs := make([]*fields.QualifiedHash, len(children))
	for i, childStr := range children {
		childIDs[i] = &fields.QualifiedHash{}
		if err := childIDs[i].UnmarshalText([]byte(childStr)); err != nil {
			return nil, fmt.Errorf("failed to transform key back into node id: %w", err)
		}
	}
	return childIDs, nil
}

func (m *MemoryStore) Add(node Node) error {
	id, err := node.ID().MarshalString()
	if err != nil {
		return err
	}
	return m.AddID(id, node)
}

func (m *MemoryStore) AddID(id string, node Node) error {
	// safe to ignore error because we know it can't happen
	if _, has, _ := m.GetID(id); has {
		return nil
	}
	m.Items[id] = node
	parentID, err := node.ParentID().MarshalString()
	if err != nil {
		return fmt.Errorf("failed to marshal string of parent node: %w", err)
	}
	m.ChildMap[parentID] = append(m.ChildMap[parentID], id)
	return nil
}

// CacheStore combines two other stores into one logical store. It is
// useful when store implementations have different performance
// characteristics and one is dramatically faster than the other. Once
// a CacheStore is created, the individual stores within it should not
// be directly modified.
type CacheStore struct {
	Cache, Back Store
}

var _ Store = &CacheStore{}

// NewCacheStore creates a single logical store from the given two stores.
// All items from `cache` are automatically copied into `base` during
// the construction of the CacheStore, and from then on (assuming
// neither store is modified directly outside of CacheStore) all elements
// added are guaranteed to be added to `base`. It is recommended to use
// fast in-memory implementations as the `cache` layer and disk or
// network-based implementations as the `base` layer.
func NewCacheStore(cache, back Store) (*CacheStore, error) {
	if err := cache.CopyInto(back); err != nil {
		return nil, err
	}
	return &CacheStore{cache, back}, nil
}

// Size returns the effective size of this CacheStore, which is the size of the
// Back Store.
func (m *CacheStore) Size() (int, error) {
	return m.Back.Size()
}

// Get returns the requested node if it is present in either the Cache or the Back Store.
// If the cache is missed by the backing store is hit, the node will automatically be
// added to the cache.
func (m *CacheStore) Get(id *fields.QualifiedHash) (Node, bool, error) {
	if node, has, err := m.Cache.Get(id); err != nil {
		return nil, false, err
	} else if has {
		return node, has, nil
	}
	if node, has, err := m.Back.Get(id); err != nil {
		return nil, false, err
	} else if has {
		if err := m.Cache.Add(node); err != nil {
			return nil, false, err
		}
		return node, has, nil
	}
	return nil, false, nil
}

func (m *CacheStore) CopyInto(other Store) error {
	return m.Back.CopyInto(other)
}

// Add inserts the given node into both stores of the CacheStore
func (m *CacheStore) Add(node Node) error {
	if err := m.Back.Add(node); err != nil {
		return err
	}
	if err := m.Cache.Add(node); err != nil {
		return err
	}
	return nil
}
