package storage

import (
	"encoding/json"
	"fmt"
	"sync"
)

// In-memory store for ns/pods/cms: map[name]=JSON string per type. Ns inits with default; others empty.
type InMemoryStore struct {
	mu       sync.RWMutex
	nsData   map[string]string
	podData  map[string]string
	cmData   map[string]string
}

var (
	DefaultStore = NewInMemoryStore()
)

func NewInMemoryStore() *InMemoryStore {
	s := &InMemoryStore{
		nsData:  make(map[string]string),
		podData: make(map[string]string),
		cmData:  make(map[string]string),
	}
	// init ns with default (others empty)
	defaultNS := `{"kind":"Namespace","apiVersion":"v1","metadata":{"name":"default","uid":"abc123","resourceVersion":"1","creationTimestamp":"2024-01-01T00:00:00Z"},"spec":{"finalizers":["kubernetes"]},"status":{"phase":"Active"}}`
	s.nsData["default"] = defaultNS
	return s
}

func (s *InMemoryStore) ListNamespaces() []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []interface{}
	for _, nsJSON := range s.nsData {
		var ns map[string]interface{}
		json.Unmarshal([]byte(nsJSON), &ns)
		items = append(items, ns)
	}
	return items
}

// ListPods, ListConfigMaps similar (no defaults).
func (s *InMemoryStore) ListPods() []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []interface{}
	for _, pJSON := range s.podData {
		var p map[string]interface{}
		json.Unmarshal([]byte(pJSON), &p)
		items = append(items, p)
	}
	return items
}

func (s *InMemoryStore) ListConfigMaps() []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []interface{}
	for _, cmJSON := range s.cmData {
		var cm map[string]interface{}
		json.Unmarshal([]byte(cmJSON), &cm)
		items = append(items, cm)
	}
	return items
}

// createHelper internal for any type (pods/cms/ns use it).
func (s *InMemoryStore) createHelper(dataMap map[string]string, ns interface{}, typ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, _ := json.Marshal(ns)
	var m map[string]interface{}
	json.Unmarshal(b, &m)
	meta, _ := m["metadata"].(map[string]interface{})
	name, _ := meta["name"].(string)
	if name == "" {
		return fmt.Errorf("%s name required", typ)
	}
	if _, exists := dataMap[name]; exists {
		return fmt.Errorf("%s %s already exists", typ, name)
	}
	dataMap[name] = string(b)
	return nil
}

func (s *InMemoryStore) CreateNamespace(ns interface{}) error {
	return s.createHelper(s.nsData, ns, "namespace")
}

func (s *InMemoryStore) CreatePod(pod interface{}) error {
	return s.createHelper(s.podData, pod, "pod")
}

func (s *InMemoryStore) CreateConfigMap(cm interface{}) error {
	return s.createHelper(s.cmData, cm, "configmap")
}

// Get helpers omitted for minimal (extend if needed).
func (s *InMemoryStore) Get(name string) (interface{}, error) {
	// placeholder, ns only for now
	s.mu.RLock()
	defer s.mu.RUnlock()
	if nsJSON, ok := s.nsData[name]; ok {
		var ns map[string]interface{}
		json.Unmarshal([]byte(nsJSON), &ns)
		return ns, nil
	}
	return nil, fmt.Errorf("not found")
}