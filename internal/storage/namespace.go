package storage

import (
	"encoding/json"

	"mockernetes/internal/resources" // for KubeObject skeleton in Create
)

// Namespace-specific storage methods (split from monolithic storage.go for modularity).
// List/Create mirror original; use shared InMemoryStore + createHelper from util.go (same package).
// (json import here as List unmarshals; resources for KubeObject; skeleton only - callers must cast e.g. resources.NamespaceImpl).

// ListNamespaces returns all stored namespaces as []interface{} (unmarshals JSON; includes default).
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

// CreateNamespace stores a namespace (error if exists; for kubectl compat).
// Uses resources.Namespace (custom struct impl of KubeObject) for mock control.
func (s *InMemoryStore) CreateNamespace(ns resources.KubeObject) error {
	return s.createHelper(s.nsData, ns, "namespace")
}
