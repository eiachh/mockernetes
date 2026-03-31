package storage

import (
	"encoding/json"
	"fmt"

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

// GetReplicaSet retrieves a replicaset by name from storage.
// Returns the replicaset as a map or error if not found.
func (s *InMemoryStore) GetReplicaSet(name string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rsJSON, exists := s.rsData[name]
	if !exists {
		return nil, fmt.Errorf("replicaset %s not found", name)
	}

	var rs map[string]interface{}
	if err := json.Unmarshal([]byte(rsJSON), &rs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal replicaset: %w", err)
	}

	return rs, nil
}

// UpdateReplicaSet updates an existing replicaset in storage.
// Returns error if the replicaset doesn't exist.
func (s *InMemoryStore) UpdateReplicaSet(rs resources.KubeObject) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := rs.GetName()
	if name == "" {
		return fmt.Errorf("replicaset name required")
	}

	if _, exists := s.rsData[name]; !exists {
		return fmt.Errorf("replicaset %s not found", name)
	}

	b, err := rs.ToJSON()
	if err != nil {
		return fmt.Errorf("toJSON failed: %w", err)
	}

	s.rsData[name] = string(b)
	return nil
}

// DeleteReplicaSet removes a replicaset from storage.
// Returns error if the replicaset doesn't exist.
func (s *InMemoryStore) DeleteReplicaSet(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rsData[name]; !exists {
		return fmt.Errorf("replicaset %s not found", name)
	}

	delete(s.rsData, name)
	return nil
}
