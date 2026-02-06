package storage

// TODO: Define interfaces for storage abstraction

// TODO: Implement in-memory storage

// TODO: Interface for NamespaceStore
type NamespaceStore interface {
	List() []interface{} // TODO: proper types
	Create(ns interface{}) error
	Get(name string) (interface{}, error)
}

// TODO: In-memory implementation
type InMemoryNamespaceStore struct {
	// TODO: fields
}

// TODO: Implement List
func (s *InMemoryNamespaceStore) List() []interface{} {
	// TODO
	return nil
}

// TODO: Implement Create
func (s *InMemoryNamespaceStore) Create(ns interface{}) error {
	// TODO
	return nil
}

// TODO: Implement Get
func (s *InMemoryNamespaceStore) Get(name string) (interface{}, error) {
	// TODO
	return nil, nil
}