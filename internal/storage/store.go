package storage

import "sync"

// InMemoryStore holds per-resource maps (strict storage; maps for JSON-serialized resources).
// (mock ignores namespace; flat by name for simplicity across resources).
// Struct/maps separated here for strict storage role (helpers in util.go).
// resources pkg (for KubeObject/custom structs) imported in other storage files (util/type .go).
type InMemoryStore struct {
	mu         sync.RWMutex
	nsData     map[string]string
	podData    map[string]string
	cmData     map[string]string
	deployData map[string]string
	rsData     map[string]string
}

// DefaultStore singleton (storage only).
var (
	DefaultStore = NewInMemoryStore()
)

// NewInMemoryStore inits store (ns default; uses custom resources shapes).
func NewInMemoryStore() *InMemoryStore {
	s := &InMemoryStore{
		nsData:     make(map[string]string),
		podData:    make(map[string]string),
		cmData:     make(map[string]string),
		deployData: make(map[string]string),
		rsData:     make(map[string]string),
	}
	// default NS JSON matching resources.Namespace struct
	defaultNS := `{"kind":"Namespace","apiVersion":"v1","metadata":{"name":"default"},"spec":{"finalizers":["kubernetes"]},"status":{"phase":"Active"}}`
	s.nsData["default"] = defaultNS
	return s
}
