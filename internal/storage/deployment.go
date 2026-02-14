package storage

import (
	"encoding/json"

	"mockernetes/internal/resources" // for KubeObject skeleton
)

// Deployment-specific storage methods (split from original storage.go; apps/v1 like RS).
// Skeleton update: Create uses KubeObject.

// ListDeployments returns stored deployments as []interface{} (mirrors pod/cm pattern).
func (s *InMemoryStore) ListDeployments() []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var items []interface{}
	for _, dJSON := range s.deployData {
		var d map[string]interface{}
		json.Unmarshal([]byte(dJSON), &d)
		items = append(items, d)
	}
	return items
}

// CreateDeployment stores a deployment (error if exists).
// Uses resources.Deployment (custom struct impl of KubeObject) for mock control.
func (s *InMemoryStore) CreateDeployment(deploy resources.KubeObject) error {
	return s.createHelper(s.deployData, deploy, "deployment")
}
