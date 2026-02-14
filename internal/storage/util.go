package storage

import (
	"encoding/json"
	"fmt"

	"mockernetes/internal/resources" // KubeObject + custom structs (resources pkg owns impls)
)

// Helpers for storage (strict: createHelper/Get; InMemoryStore/maps now in store.go).
// createHelper internal for resources.KubeObject (uses custom structs for mock control).
// Note: legacy param name; calls obj.ToJSON() from impl (Namespace/Pod/etc.).
// Metadata extract kept for name check (extend to use GetName()).
func (s *InMemoryStore) createHelper(dataMap map[string]string, obj resources.KubeObject, typ string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := obj.ToJSON() // uses custom struct impl
	if err != nil {
		return fmt.Errorf("toJSON failed: %w", err)
	}
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

// Get helpers omitted for minimal (extend if needed; placeholder for ns).
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
