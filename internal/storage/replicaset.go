package storage

import (
	"encoding/json"

	"mockernetes/internal/resources" // for KubeObject skeleton
)

// ReplicaSet-specific storage methods (split from original storage.go; apps/v1).
// Skeleton update: Create uses KubeObject.

// ListReplicaSets returns stored replicasets as []interface{}.
func (s *InMemoryStore) ListReplicaSets() []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []interface{}
	for _, rsJSON := range s.rsData {
		var rs map[string]interface{}
		json.Unmarshal([]byte(rsJSON), &rs)
		items = append(items, rs)
	}
	return items
}

// CreateReplicaSet stores a replicaset (error if exists).
// Uses resources.ReplicaSet (custom struct impl of KubeObject) for mock control.
func (s *InMemoryStore) CreateReplicaSet(rs resources.KubeObject) error {
	return s.createHelper(s.rsData, rs, "replicaset")
}
