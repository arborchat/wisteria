package forest

import (
	"git.sr.ht/~whereswaldon/forest-go/fields"
)

type Store interface {
	Size() (int, error)
	Get(*fields.QualifiedHash) (Node, bool, error)
	Add(Node) error
}

type MemoryStore struct {
	Items map[string]Node
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{make(map[string]Node)}
}

func (m *MemoryStore) Size() (int, error) {
	return len(m.Items), nil
}

func (m *MemoryStore) Get(id *fields.QualifiedHash) (Node, bool, error) {
	idString, err := id.MarshalString()
	if err != nil {
		return nil, false, err
	}
	return m.GetID(idString)
}

func (m *MemoryStore) GetID(id string) (Node, bool, error) {
	item, has := m.Items[id]
	return item, has, nil
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
	return nil
}

type CacheStore struct {
	Cache, Back Store
}

func NewCacheStore(cache, back Store) *CacheStore {
	return &CacheStore{cache, back}
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
