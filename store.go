package forest

import (
	"fmt"

	"git.sr.ht/~whereswaldon/forest-go/fields"
)

type Store interface {
	Size() (int, error)
	Has(*fields.QualifiedHash) (bool, error)
	Get(*fields.QualifiedHash) (Node, error)
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

func (m *MemoryStore) Has(id *fields.QualifiedHash) (bool, error) {
	idString, err := id.MarshalString()
	if err != nil {
		return false, err
	}
	return m.HasID(idString)
}

func (m *MemoryStore) HasID(id string) (bool, error) {
	_, has := m.Items[id]
	return has, nil
}

func (m *MemoryStore) Get(id *fields.QualifiedHash) (Node, error) {
	idString, err := id.MarshalString()
	if err != nil {
		return nil, err
	}
	return m.GetID(idString)
}

func (m *MemoryStore) GetID(id string) (Node, error) {
	item, has := m.Items[id]
	if !has {
		return nil, fmt.Errorf("MemoryStore does not contain id %s", id)
	}
	return item, nil
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
	if has, _ := m.HasID(id); has {
		return nil
	}
	m.Items[id] = node
	return nil
}
