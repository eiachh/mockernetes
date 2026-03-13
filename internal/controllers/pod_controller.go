package controllers

import (
	"fmt"
	"sync"
	"time"

	"mockernetes/internal/resources"
	"mockernetes/internal/storage"
)

// PodPhase represents the lifecycle phase of a Pod
type PodPhase string

const (
	PodPending   PodPhase = "Pending"
	PodRunning   PodPhase = "Running"
	PodSucceeded PodPhase = "Succeeded"
	PodFailed    PodPhase = "Failed"
	PodUnknown   PodPhase = "Unknown"
)

// PodStatus represents the status of a Pod
// JSON field names match Kubernetes API conventions
type PodStatus struct {
	Phase      PodPhase        `json:"phase"`
	Conditions []PodCondition  `json:"conditions,omitempty"`
	PodIP      string          `json:"podIP"`
	HostIP     string          `json:"hostIP"`
	StartTime  *time.Time      `json:"startTime,omitempty"`
	ContainerStatuses []ContainerStatus `json:"containerStatuses,omitempty"`
}

// PodCondition represents a condition of a Pod
type PodCondition struct {
	Type               string     `json:"type"`
	Status             string     `json:"status"`
	LastTransitionTime *time.Time `json:"lastTransitionTime,omitempty"`
	Reason             string     `json:"reason,omitempty"`
	Message            string     `json:"message,omitempty"`
}

// ContainerStatus represents the status of a container in a Pod
type ContainerStatus struct {
	Name         string                 `json:"name"`
	Image        string                 `json:"image"`
	ImageID      string                 `json:"imageID"`
	ContainerID  string                 `json:"containerID"`
	Ready        bool                   `json:"ready"`
	RestartCount int32                  `json:"restartCount"`
	State        map[string]interface{} `json:"state,omitempty"`
	LastState    map[string]interface{} `json:"lastState,omitempty"`
}

// PodController manages the lifecycle of Pods
type PodController struct {
	store  *storage.InMemoryStore
	mu     sync.RWMutex
	stopCh chan struct{}
}

// NewPodController creates a new PodController
func NewPodController(store *storage.InMemoryStore) *PodController {
	return &PodController{
		store:  store,
		stopCh: make(chan struct{}),
	}
}

// Start starts the controller (for now just initializes)
func (pc *PodController) Start() {
	// In the future, this could start a watch loop for pod reconciliations
}

// Stop stops the controller
func (pc *PodController) Stop() {
	close(pc.stopCh)
}

// OnPodCreated is called when a new pod is created
// It initializes the pod lifecycle by setting the Pending phase
func (pc *PodController) OnPodCreated(pod resources.Pod) error {
	// Set initial status to Pending
	now := time.Now()
	status := PodStatus{
		Phase: PodPending,
		Conditions: []PodCondition{
			{
				Type:               "Initialized",
				Status:             "True",
				LastTransitionTime: &now,
			},
			{
				Type:               "Ready",
				Status:             "False",
				LastTransitionTime: &now,
				Reason:             "ContainersNotReady",
				Message:            "containers with unready status",
			},
		},
	}

	// Update the pod with the new status
	return pc.updatePodStatusWithMetadata(pod, status, now)
}

// OnPodStarted is called when a pod's containers are started
// It transitions the pod from Pending to Running
func (pc *PodController) OnPodStarted(pod resources.Pod) error {
	now := time.Now()

	// Build container statuses from pod spec
	containerStatuses := pc.buildContainerStatuses(pod.Spec, true)

	status := PodStatus{
		Phase:             PodRunning,
		PodIP:             "10.244.0.1", // Mock pod IP
		HostIP:            "127.0.0.1",  // Mock host IP
		StartTime:         &now,
		ContainerStatuses: containerStatuses,
		Conditions: []PodCondition{
			{
				Type:               "Initialized",
				Status:             "True",
				LastTransitionTime: &now,
			},
			{
				Type:               "Ready",
				Status:             "True",
				LastTransitionTime: &now,
			},
			{
				Type:               "ContainersReady",
				Status:             "True",
				LastTransitionTime: &now,
			},
			{
				Type:               "PodScheduled",
				Status:             "True",
				LastTransitionTime: &now,
			},
		},
	}

	return pc.updatePodStatus(pod, status)
}

// buildContainerStatuses extracts container info from pod spec and creates container statuses
func (pc *PodController) buildContainerStatuses(spec interface{}, ready bool) []ContainerStatus {
	var statuses []ContainerStatus

	// Try to extract containers from spec
	if specMap, ok := spec.(map[string]interface{}); ok {
		if containers, ok := specMap["containers"].([]interface{}); ok {
			for i, c := range containers {
				if container, ok := c.(map[string]interface{}); ok {
					name := ""
					image := ""
					if n, ok := container["name"].(string); ok {
						name = n
					}
					if img, ok := container["image"].(string); ok {
						image = img
					}

					now := time.Now()
					status := ContainerStatus{
						Name:         name,
						Image:        image,
						ImageID:      "docker-pullable://nginx@sha256:mock",
						ContainerID:  fmt.Sprintf("docker://container-%d", i),
						Ready:        ready,
						RestartCount: 0,
						State: map[string]interface{}{
							"running": map[string]interface{}{
								"startedAt": now.Format(time.RFC3339),
							},
						},
						LastState: map[string]interface{}{},
					}
					statuses = append(statuses, status)
				}
			}
		}
	}

	return statuses
}

// OnPodCompleted is called when a pod's containers have completed
// It transitions the pod to Succeeded phase
func (pc *PodController) OnPodCompleted(pod resources.Pod) error {
	now := time.Now()
	status := PodStatus{
		Phase: PodSucceeded,
		Conditions: []PodCondition{
			{
				Type:               "Initialized",
				Status:             "True",
				LastTransitionTime: &now,
			},
			{
				Type:               "Ready",
				Status:             "False",
				LastTransitionTime: &now,
				Reason:             "PodCompleted",
			},
		},
	}

	return pc.updatePodStatus(pod, status)
}

// OnPodFailed is called when a pod fails
// It transitions the pod to Failed phase
func (pc *PodController) OnPodFailed(pod resources.Pod, reason string) error {
	now := time.Now()
	status := PodStatus{
		Phase: PodFailed,
		Conditions: []PodCondition{
			{
				Type:               "Initialized",
				Status:             "True",
				LastTransitionTime: &now,
			},
			{
				Type:               "Ready",
				Status:             "False",
				LastTransitionTime: &now,
				Reason:             reason,
			},
		},
	}

	return pc.updatePodStatus(pod, status)
}

// updatePodStatus updates the pod status in the store
// Preserves existing metadata including creationTimestamp
func (pc *PodController) updatePodStatus(pod resources.Pod, status PodStatus) error {
	// Get existing pod from storage to preserve metadata
	existingPod, err := pc.store.GetPod(pod.GetName())
	if err != nil {
		// Pod not found, use original pod metadata
		existingPod = nil
	}

	metadata := pod.Metadata
	if existingPod != nil {
		// Preserve creationTimestamp from existing pod
		if meta, ok := existingPod["metadata"].(map[string]interface{}); ok {
			if ct, ok := meta["creationTimestamp"].(string); ok && ct != "" {
				metadata.CreationTimestamp = ct
			}
		}
	}

	// Create a new pod with updated status
	updatedPod := resources.Pod{
		Kind:       pod.Kind,
		APIVersion: pod.APIVersion,
		Metadata:   metadata,
		Spec:       pod.Spec,
		Status:     status,
	}

	// Update in storage
	return pc.store.UpdatePod(updatedPod)
}

// updatePodStatusWithMetadata updates the pod status and sets creation timestamp
func (pc *PodController) updatePodStatusWithMetadata(pod resources.Pod, status PodStatus, creationTime time.Time) error {
	// Create a new pod with updated status and creation timestamp
	metadata := pod.Metadata
	if metadata.CreationTimestamp == "" {
		metadata.CreationTimestamp = creationTime.Format(time.RFC3339)
	}

	updatedPod := resources.Pod{
		Kind:       pod.Kind,
		APIVersion: pod.APIVersion,
		Metadata:   metadata,
		Spec:       pod.Spec,
		Status:     status,
	}

	// Update in storage
	return pc.store.UpdatePod(updatedPod)
}

// DefaultPodController is the singleton instance
var DefaultPodController *PodController

// InitPodController initializes the default pod controller
func InitPodController(store *storage.InMemoryStore) {
	DefaultPodController = NewPodController(store)
	DefaultPodController.Start()
}
