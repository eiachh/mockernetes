package controllers

import (
	"context"
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
	Phase             PodPhase          `json:"phase"`
	Conditions        []PodCondition    `json:"conditions,omitempty"`
	PodIP             string            `json:"podIP"`
	HostIP            string            `json:"hostIP"`
	StartTime         *time.Time        `json:"startTime,omitempty"`
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
	// StartupDelay is the duration a pod stays in Pending before transitioning to Running.
	StartupDelay time.Duration
}

// NewPodController creates a new PodController with the given startup delay.
func NewPodController(store *storage.InMemoryStore, startupDelay time.Duration) *PodController {
	return &PodController{
		store:        store,
		stopCh:       make(chan struct{}),
		StartupDelay: startupDelay,
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

// OnPodCreated is called when a new pod is created.
// It sets the pod to Pending immediately and schedules an async transition
// to Running after StartupDelay.
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
	if err := pc.updatePodStatusWithMetadata(pod, status, now); err != nil {
		return err
	}

	// Schedule async transition to Running after StartupDelay
	go func() {
		select {
		case <-time.After(pc.StartupDelay):
			pc.OnPodStarted(pod)
		case <-pc.stopCh:
			return
		}
	}()

	return nil
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
// Preserves existing metadata including creationTimestamp and ownerReferences
func (pc *PodController) updatePodStatus(pod resources.Pod, status PodStatus) error {
	// Get existing pod from storage to preserve metadata
	existingPod, err := pc.store.GetPod(pod.GetName())
	if err != nil {
		// Pod not found, use original pod metadata
		existingPod = nil
	}

	metadata := pod.Metadata
	if existingPod != nil {
		if meta, ok := existingPod["metadata"].(map[string]interface{}); ok {
			// Preserve creationTimestamp from existing pod
			if ct, ok := meta["creationTimestamp"].(string); ok && ct != "" {
				metadata.CreationTimestamp = ct
			}
			// Preserve ownerReferences from existing pod if not set in incoming pod
			if len(metadata.OwnerReferences) == 0 {
				if ownerRefsRaw, ok := meta["ownerReferences"].([]interface{}); ok {
					metadata.OwnerReferences = convertToOwnerReferences(ownerRefsRaw)
				}
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

// convertToOwnerReferences converts []interface{} to []resources.OwnerReference
func convertToOwnerReferences(ownerRefsRaw []interface{}) []resources.OwnerReference {
	var ownerRefs []resources.OwnerReference
	for _, refRaw := range ownerRefsRaw {
		if refMap, ok := refRaw.(map[string]interface{}); ok {
			ownerRef := resources.OwnerReference{}
			if apiVersion, ok := refMap["apiVersion"].(string); ok {
				ownerRef.APIVersion = apiVersion
			}
			if kind, ok := refMap["kind"].(string); ok {
				ownerRef.Kind = kind
			}
			if name, ok := refMap["name"].(string); ok {
				ownerRef.Name = name
			}
			if uid, ok := refMap["uid"].(string); ok {
				ownerRef.UID = uid
			}
			if controller, ok := refMap["controller"].(bool); ok {
				ownerRef.Controller = controller
			}
			if blockOwnerDeletion, ok := refMap["blockOwnerDeletion"].(bool); ok {
				ownerRef.BlockOwnerDeletion = blockOwnerDeletion
			}
			ownerRefs = append(ownerRefs, ownerRef)
		}
	}
	return ownerRefs
}

// updatePodStatusWithMetadata updates the pod status and sets creation timestamp
// Preserves existing metadata including ownerReferences
func (pc *PodController) updatePodStatusWithMetadata(pod resources.Pod, status PodStatus, creationTime time.Time) error {
	// Get existing pod from storage to preserve metadata
	existingPod, err := pc.store.GetPod(pod.GetName())
	if err != nil {
		existingPod = nil
	}

	// Create a new pod with updated status and creation timestamp
	metadata := pod.Metadata
	if metadata.CreationTimestamp == "" {
		metadata.CreationTimestamp = creationTime.Format(time.RFC3339)
	}

	// Preserve ownerReferences from existing pod if not set in incoming pod
	if existingPod != nil && len(metadata.OwnerReferences) == 0 {
		if meta, ok := existingPod["metadata"].(map[string]interface{}); ok {
			if ownerRefsRaw, ok := meta["ownerReferences"].([]interface{}); ok {
				metadata.OwnerReferences = convertToOwnerReferences(ownerRefsRaw)
			}
		}
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

// TransitionState defines a single state in a pod's lifecycle transition
// with the duration to wait before applying it.
type TransitionState struct {
	Phase           string                 `json:"phase"`
	Delay           string                 `json:"delay"`
	PodIP           string                 `json:"podIP,omitempty"`
	HostIP          string                 `json:"hostIP,omitempty"`
	Conditions      []PodCondition         `json:"conditions,omitempty"`
	ContainerStates map[string]interface{} `json:"containerStates,omitempty"`
}

// TransitionRequest defines a complete state transition sequence for a pod
type TransitionRequest struct {
	PodName        string            `json:"podName"`
	Namespace      string            `json:"namespace,omitempty"`
	CancelExisting bool              `json:"cancelExisting,omitempty"`
	Transitions    []TransitionState `json:"transitions"`
}

// ActiveTransition tracks a running transition sequence for a pod
type ActiveTransition struct {
	PodName     string
	Namespace   string
	CancelFunc  context.CancelFunc
	Transitions []TransitionState
}

// TransitionManager manages active state transitions across pods
type TransitionManager struct {
	mu        sync.RWMutex
	active    map[string]*ActiveTransition // key: "namespace/podName"
	store     *storage.InMemoryStore
}

// NewTransitionManager creates a new transition manager
func NewTransitionManager(store *storage.InMemoryStore) *TransitionManager {
	return &TransitionManager{
		active: make(map[string]*ActiveTransition),
		store:  store,
	}
}

// key generates a unique key for a pod
func (tm *TransitionManager) key(namespace, podName string) string {
	if namespace == "" {
		namespace = "default"
	}
	return fmt.Sprintf("%s/%s", namespace, podName)
}

// CancelTransition cancels any active transition for the given pod
func (tm *TransitionManager) CancelTransition(namespace, podName string) bool {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	key := tm.key(namespace, podName)
	if active, exists := tm.active[key]; exists {
		active.CancelFunc()
		delete(tm.active, key)
		return true
	}
	return false
}

// StartTransition begins a new state transition sequence for a pod
func (tm *TransitionManager) StartTransition(req TransitionRequest) (*ActiveTransition, error) {
	if req.PodName == "" {
		return nil, fmt.Errorf("podName is required")
	}
	if len(req.Transitions) == 0 {
		return nil, fmt.Errorf("at least one transition is required")
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	// Cancel existing transition if requested
	if req.CancelExisting {
		tm.CancelTransition(namespace, req.PodName)
	}

	// Verify pod exists
	_, err := tm.store.GetPod(req.PodName)
	if err != nil {
		return nil, fmt.Errorf("pod %s not found: %w", req.PodName, err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	active := &ActiveTransition{
		PodName:     req.PodName,
		Namespace:   namespace,
		CancelFunc:  cancel,
		Transitions: req.Transitions,
	}

	// Store the active transition
	tm.mu.Lock()
	tm.active[tm.key(namespace, req.PodName)] = active
	tm.mu.Unlock()

	// Start the transition sequence in background
	go tm.runTransitionSequence(ctx, active)

	return active, nil
}

// runTransitionSequence executes the state transition sequence
func (tm *TransitionManager) runTransitionSequence(ctx context.Context, active *ActiveTransition) {
	defer func() {
		// Clean up on completion
		tm.mu.Lock()
		delete(tm.active, tm.key(active.Namespace, active.PodName))
		tm.mu.Unlock()
	}()

	for i, state := range active.Transitions {
		select {
		case <-ctx.Done():
			return // Transition was cancelled
		default:
		}

		// Parse delay duration
		delay, err := time.ParseDuration(state.Delay)
		if err != nil {
			delay = 0
		}

		// Wait for the delay (skip for first state if delay is 0)
		if i > 0 || delay > 0 {
			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return
			}
		}

		// Apply the state
		tm.applyState(active.Namespace, active.PodName, state)
	}
}

// applyState applies a single transition state to a pod
func (tm *TransitionManager) applyState(namespace, podName string, state TransitionState) error {
	// Get current pod
	existingPod, err := tm.store.GetPod(podName)
	if err != nil {
		return err
	}

	// Build conditions with timestamps
	now := time.Now()
	conditions := make([]PodCondition, 0, len(state.Conditions))
	for _, cond := range state.Conditions {
		cond.LastTransitionTime = &now
		conditions = append(conditions, cond)
	}

	// Build status
	status := PodStatus{
		Phase:      PodPhase(state.Phase),
		PodIP:      state.PodIP,
		HostIP:     state.HostIP,
		Conditions: conditions,
	}

	// Build container statuses if provided
	if len(state.ContainerStates) > 0 {
		status.ContainerStatuses = tm.buildContainerStatusesFromStates(state.ContainerStates)
	}

	// Extract existing metadata
	metadata := resources.ObjectMeta{
		Name:      podName,
		Namespace: namespace,
	}
	if meta, ok := existingPod["metadata"].(map[string]interface{}); ok {
		if ns, ok := meta["namespace"].(string); ok {
			metadata.Namespace = ns
		}
		if ct, ok := meta["creationTimestamp"].(string); ok {
			metadata.CreationTimestamp = ct
		}
		if labels, ok := meta["labels"].(map[string]interface{}); ok {
			metadata.Labels = make(map[string]string)
			for k, v := range labels {
				if sv, ok := v.(string); ok {
					metadata.Labels[k] = sv
				}
			}
		}
	}

	// Extract spec
	var spec interface{}
	if s, ok := existingPod["spec"]; ok {
		spec = s
	}

	// Create updated pod
	updatedPod := resources.Pod{
		Kind:       "Pod",
		APIVersion: "v1",
		Metadata:   metadata,
		Spec:       spec,
		Status:     status,
	}

	return tm.store.UpdatePod(updatedPod)
}

// buildContainerStatusesFromStates creates ContainerStatus from the state definition
func (tm *TransitionManager) buildContainerStatusesFromStates(states map[string]interface{}) []ContainerStatus {
	var statuses []ContainerStatus
	now := time.Now()

	for name, stateDef := range states {
		if stateMap, ok := stateDef.(map[string]interface{}); ok {
			status := ContainerStatus{
				Name:         name,
				Image:        "custom-image",
				ImageID:      "docker-pullable://custom@sha256:mock",
				ContainerID:  fmt.Sprintf("docker://%s", name),
				Ready:        true,
				RestartCount: 0,
				State:        stateMap,
				LastState:    map[string]interface{}{},
			}
			// Set startedAt if running state
			if running, ok := stateMap["running"].(map[string]interface{}); ok {
				if _, exists := running["startedAt"]; !exists {
					running["startedAt"] = now.Format(time.RFC3339)
				}
			}
			statuses = append(statuses, status)
		}
	}

	return statuses
}

// GetActiveTransition returns the active transition for a pod if any
func (tm *TransitionManager) GetActiveTransition(namespace, podName string) (*ActiveTransition, bool) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	active, exists := tm.active[tm.key(namespace, podName)]
	return active, exists
}

// ListActiveTransitions returns all active transitions
func (tm *TransitionManager) ListActiveTransitions() []*ActiveTransition {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make([]*ActiveTransition, 0, len(tm.active))
	for _, active := range tm.active {
		result = append(result, active)
	}
	return result
}

// DefaultPodController is the singleton instance
var DefaultPodController *PodController

// DefaultTransitionManager is the singleton transition manager
var DefaultTransitionManager *TransitionManager

// DefaultStartupDelay is the default time a pod stays in Pending before transitioning to Running.
const DefaultStartupDelay = 20 * time.Second

// InitPodController initializes the default pod controller with the default startup delay.
func InitPodController(store *storage.InMemoryStore) {
	DefaultPodController = NewPodController(store, DefaultStartupDelay)
	DefaultPodController.Start()
	DefaultTransitionManager = NewTransitionManager(store)
}

// TransitionTemplate stores a pre-defined transition sequence for a pod name
type TransitionTemplate struct {
	PodName     string            `json:"podName"`
	Namespace   string            `json:"namespace,omitempty"`
	Transitions []TransitionState `json:"transitions"`
	CreatedAt   time.Time         `json:"createdAt"`
}

// TemplateRegistry manages pre-defined transition templates
type TemplateRegistry struct {
	mu        sync.RWMutex
	templates map[string]*TransitionTemplate // key: "namespace/podName"
}

// NewTemplateRegistry creates a new template registry
func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{
		templates: make(map[string]*TransitionTemplate),
	}
}

// key generates a unique key for a pod template
func (tr *TemplateRegistry) key(namespace, podName string) string {
	if namespace == "" {
		namespace = "default"
	}
	return fmt.Sprintf("%s/%s", namespace, podName)
}

// RegisterTemplate stores a transition template for a pod name
func (tr *TemplateRegistry) RegisterTemplate(template TransitionTemplate) error {
	if template.PodName == "" {
		return fmt.Errorf("podName is required")
	}
	if len(template.Transitions) == 0 {
		return fmt.Errorf("at least one transition is required")
	}

	namespace := template.Namespace
	if namespace == "" {
		namespace = "default"
	}

	template.Namespace = namespace
	template.CreatedAt = time.Now()

	tr.mu.Lock()
	defer tr.mu.Unlock()

	tr.templates[tr.key(namespace, template.PodName)] = &template
	return nil
}

// GetTemplate retrieves a template for a pod if one exists
func (tr *TemplateRegistry) GetTemplate(namespace, podName string) (*TransitionTemplate, bool) {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	template, exists := tr.templates[tr.key(namespace, podName)]
	return template, exists
}

// RemoveTemplate removes a template for a pod
func (tr *TemplateRegistry) RemoveTemplate(namespace, podName string) bool {
	tr.mu.Lock()
	defer tr.mu.Unlock()

	key := tr.key(namespace, podName)
	if _, exists := tr.templates[key]; exists {
		delete(tr.templates, key)
		return true
	}
	return false
}

// ListTemplates returns all registered templates
func (tr *TemplateRegistry) ListTemplates() []*TransitionTemplate {
	tr.mu.RLock()
	defer tr.mu.RUnlock()

	result := make([]*TransitionTemplate, 0, len(tr.templates))
	for _, template := range tr.templates {
		result = append(result, template)
	}
	return result
}

// DefaultTemplateRegistry is the singleton template registry
var DefaultTemplateRegistry *TemplateRegistry

// InitTemplateRegistry initializes the default template registry
func InitTemplateRegistry() {
	DefaultTemplateRegistry = NewTemplateRegistry()
}
