package storage

import (
	"encoding/json"
	"fmt"

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

// GetDeployment retrieves a deployment by name from storage.
// Returns the deployment as a map or error if not found.
func (s *InMemoryStore) GetDeployment(name string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	deployJSON, exists := s.deployData[name]
	if !exists {
		return nil, fmt.Errorf("deployment %s not found", name)
	}

	var deploy map[string]interface{}
	if err := json.Unmarshal([]byte(deployJSON), &deploy); err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	return deploy, nil
}

// UpdateDeployment updates an existing deployment in storage.
// Returns error if the deployment doesn't exist.
func (s *InMemoryStore) UpdateDeployment(deploy resources.KubeObject) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := deploy.GetName()
	if name == "" {
		return fmt.Errorf("deployment name required")
	}

	if _, exists := s.deployData[name]; !exists {
		return fmt.Errorf("deployment %s not found", name)
	}

	b, err := deploy.ToJSON()
	if err != nil {
		return fmt.Errorf("toJSON failed: %w", err)
	}

	s.deployData[name] = string(b)
	return nil
}

// DeleteDeployment removes a deployment from storage.
// Returns error if the deployment doesn't exist.
func (s *InMemoryStore) DeleteDeployment(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.deployData[name]; !exists {
		return fmt.Errorf("deployment %s not found", name)
	}

	delete(s.deployData, name)
	return nil
}
