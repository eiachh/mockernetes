package storage

import (
	"encoding/json"

	"mockernetes/internal/resources" // for KubeObject skeleton
)

// ConfigMap-specific storage methods (split from original storage.go).
// Skeleton update: Create uses KubeObject.

// ListConfigMaps returns stored configmaps as []interface{}.
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

// CreateConfigMap stores a configmap (error if exists).
// Uses resources.ConfigMap (custom struct impl of KubeObject) for mock control.
func (s *InMemoryStore) CreateConfigMap(cm resources.KubeObject) error {
	return s.createHelper(s.cmData, cm, "configmap")
}
