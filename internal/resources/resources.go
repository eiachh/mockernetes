package resources

import "encoding/json"

// KubeObject is a common interface for Kubernetes resources in the mock.
// Owned by resources pkg for clean separation (storage only persists, no struct defs).
// Skeleton methods for type safety + mock control.
type KubeObject interface {
	GetName() string
	GetNamespace() string
	ToJSON() ([]byte, error)
	GetKind() string // e.g., "Pod" (GetKind to avoid field conflict)
}

// ObjectMeta shared for custom resource structs.
type ObjectMeta struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	// TODO: uid, resourceVersion for full mock
}

// Namespace custom struct (impls KubeObject).
type Namespace struct {
	Kind       string      `json:"kind"`
	APIVersion string      `json:"apiVersion"`
	Metadata   ObjectMeta  `json:"metadata"`
	Spec       interface{} `json:"spec,omitempty"`
	Status     interface{} `json:"status,omitempty"`
}

func (n Namespace) GetName() string         { return n.Metadata.Name }
func (n Namespace) GetNamespace() string    { return n.Metadata.Namespace }
func (n Namespace) ToJSON() ([]byte, error) { return json.Marshal(n) }
func (n Namespace) GetKind() string         { return n.Kind }

// Pod custom struct.
type Pod struct {
	Kind       string      `json:"kind"`
	APIVersion string      `json:"apiVersion"`
	Metadata   ObjectMeta  `json:"metadata"`
	Spec       interface{} `json:"spec,omitempty"`
	Status     interface{} `json:"status,omitempty"`
}

func (p Pod) GetName() string         { return p.Metadata.Name }
func (p Pod) GetNamespace() string    { return p.Metadata.Namespace }
func (p Pod) ToJSON() ([]byte, error) { return json.Marshal(p) }
func (p Pod) GetKind() string         { return p.Kind }

// ConfigMap custom struct.
type ConfigMap struct {
	Kind       string            `json:"kind"`
	APIVersion string            `json:"apiVersion"`
	Metadata   ObjectMeta        `json:"metadata"`
	Data       map[string]string `json:"data,omitempty"`
}

func (c ConfigMap) GetName() string         { return c.Metadata.Name }
func (c ConfigMap) GetNamespace() string    { return c.Metadata.Namespace }
func (c ConfigMap) ToJSON() ([]byte, error) { return json.Marshal(c) }
func (c ConfigMap) GetKind() string         { return c.Kind }

// Deployment custom struct.
type Deployment struct {
	Kind       string      `json:"kind"`
	APIVersion string      `json:"apiVersion"`
	Metadata   ObjectMeta  `json:"metadata"`
	Spec       interface{} `json:"spec,omitempty"`
	Status     interface{} `json:"status,omitempty"`
}

func (d Deployment) GetName() string         { return d.Metadata.Name }
func (d Deployment) GetNamespace() string    { return d.Metadata.Namespace }
func (d Deployment) ToJSON() ([]byte, error) { return json.Marshal(d) }
func (d Deployment) GetKind() string         { return d.Kind }

// ReplicaSet custom struct.
type ReplicaSet struct {
	Kind       string      `json:"kind"`
	APIVersion string      `json:"apiVersion"`
	Metadata   ObjectMeta  `json:"metadata"`
	Spec       interface{} `json:"spec,omitempty"`
	Status     interface{} `json:"status,omitempty"`
}

func (r ReplicaSet) GetName() string         { return r.Metadata.Name }
func (r ReplicaSet) GetNamespace() string    { return r.Metadata.Namespace }
func (r ReplicaSet) ToJSON() ([]byte, error) { return json.Marshal(r) }
func (r ReplicaSet) GetKind() string         { return r.Kind }

// ListResponse skeleton for resources.
type ListResponse struct {
	Kind       string            `json:"kind"`
	APIVersion string            `json:"apiVersion"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Items      []KubeObject      `json:"items"`
}

// NewListResponse skeleton.
func NewListResponse(items []KubeObject) ListResponse {
	return ListResponse{Items: items}
}

// TODO: ToKubeObjectFromJSON etc. for resources pkg.
