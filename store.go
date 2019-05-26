package forest

import "git.sr.ht/~whereswaldon/forest-go/fields"

type Store interface {
	Size() (int, error)
	Has(*fields.QualifiedHash) (bool, error)
	Get(*fields.QualifiedHash) (Node, error)
	Add(Node) error
}

type MemoryStore struct {
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

func (m *MemoryStore) Size() (int, error) {
	return 0, nil
}

func (m *MemoryStore) Has(id *fields.QualifiedHash) (bool, error) {
	return false, nil
}

func (m *MemoryStore) Get(id *fields.QualifiedHash) (Node, error) {
	return nil, nil
}

func (m *MemoryStore) Add(node Node) error {
	return nil
}
