package storage

import (
	"encoding/json"
	"fmt"

	"mockernetes/internal/resources" // for KubeObject skeleton
)

// Pod-specific storage methods (split from original storage.go).
// Skeleton update: Create uses KubeObject.

// ListPods returns stored pods as []interface{} (JSON unmarshal for K8s compat).
// Status/phase is expected to be set by the pod controller, not patched here.
func (s *InMemoryStore) ListPods() []interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]interface{}, 0, len(s.podData))
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

// UpdatePod updates an existing pod in storage.
// Returns error if the pod doesn't exist.
func (s *InMemoryStore) UpdatePod(pod resources.KubeObject) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	name := pod.GetName()
	if name == "" {
		return fmt.Errorf("pod name required")
	}

	if _, exists := s.podData[name]; !exists {
		return fmt.Errorf("pod %s not found", name)
	}

	b, err := pod.ToJSON()
	if err != nil {
		return fmt.Errorf("toJSON failed: %w", err)
	}

	s.podData[name] = string(b)
	return nil
}

// GetPod retrieves a pod by name from storage.
// Returns the pod as a map or error if not found.
func (s *InMemoryStore) GetPod(name string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	podJSON, exists := s.podData[name]
	if !exists {
		return nil, fmt.Errorf("pod %s not found", name)
	}

	var pod map[string]interface{}
	if err := json.Unmarshal([]byte(podJSON), &pod); err != nil {
		return nil, fmt.Errorf("failed to unmarshal pod: %w", err)
	}

	return pod, nil
}

// DeletePod removes a pod from storage.
// Returns error if the pod doesn't exist.
func (s *InMemoryStore) DeletePod(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.podData[name]; !exists {
		return fmt.Errorf("pod %s not found", name)
	}

	delete(s.podData, name)
	return nil
}

// UpdatePodFromJSON updates a pod directly with JSON bytes.
// This is used internally by controllers that need to preserve complex metadata.
func (s *InMemoryStore) UpdatePodFromJSON(name string, jsonData []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.podData[name]; !exists {
		return fmt.Errorf("pod %s not found", name)
	}

	s.podData[name] = string(jsonData)
	return nil
}
