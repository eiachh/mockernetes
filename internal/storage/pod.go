package storage

import (
	"encoding/json"

	"mockernetes/internal/resources" // for KubeObject skeleton
)

// Pod-specific storage methods (split from original storage.go).
// Skeleton update: Create uses KubeObject.

// ListPods returns stored pods as []interface{} (JSON unmarshal for K8s compat; no defaults).
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

// CreatePod stores a pod (error if exists).
// Uses resources.Pod (custom struct impl of KubeObject) for mock control.
func (s *InMemoryStore) CreatePod(pod resources.KubeObject) error {
	return s.createHelper(s.podData, pod, "pod")
}
