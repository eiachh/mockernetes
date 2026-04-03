package controllers

import (
	"fmt"
	"sync"
	"time"

	"mockernetes/internal/resources"
	"mockernetes/internal/storage"
)

// ReplicaSetController manages the lifecycle of ReplicaSets and their pods
type ReplicaSetController struct {
	store        *storage.InMemoryStore
	mu           sync.RWMutex
	stopCh       chan struct{}
	reconciling  map[string]bool // track which RS are being reconciled
	reconcileMu  sync.Mutex
}

// NewReplicaSetController creates a new ReplicaSetController
func NewReplicaSetController(store *storage.InMemoryStore) *ReplicaSetController {
	return &ReplicaSetController{
		store:       store,
		stopCh:      make(chan struct{}),
		reconciling: make(map[string]bool),
	}
}

// Start starts the controller's reconciliation loop
func (rsc *ReplicaSetController) Start() {
	// Start a background reconciliation loop
	go rsc.reconcileLoop()
}

// Stop stops the controller
func (rsc *ReplicaSetController) Stop() {
	close(rsc.stopCh)
}

// reconcileLoop periodically reconciles ReplicaSets
func (rsc *ReplicaSetController) reconcileLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-rsc.stopCh:
			return
		case <-ticker.C:
			rsc.reconcileAll()
		}
	}
}

// reconcileAll reconciles all ReplicaSets
func (rsc *ReplicaSetController) reconcileAll() {
	rss := rsc.store.ListReplicaSets()
	for _, rsItem := range rss {
		if rs, ok := rsItem.(map[string]interface{}); ok {
			rsc.reconcileReplicaSet(rs)
		}
	}
}

// reconcileReplicaSet ensures the ReplicaSet has the correct number of pods
func (rsc *ReplicaSetController) reconcileReplicaSet(rs map[string]interface{}) {
	// Extract ReplicaSet info
	metadata, _ := rs["metadata"].(map[string]interface{})
	spec, _ := rs["spec"].(map[string]interface{})

	rsName, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	// Prevent concurrent reconciliation of the same ReplicaSet
	rsKey := fmt.Sprintf("%s/%s", namespace, rsName)
	rsc.reconcileMu.Lock()
	if rsc.reconciling[rsKey] {
		rsc.reconcileMu.Unlock()
		fmt.Printf("[RS Controller] Skipping reconciliation of %s - already in progress\n", rsKey)
		return
	}
	rsc.reconciling[rsKey] = true
	rsc.reconcileMu.Unlock()

	defer func() {
		rsc.reconcileMu.Lock()
		delete(rsc.reconciling, rsKey)
		rsc.reconcileMu.Unlock()
	}()

	// Get desired replicas
	desiredReplicas := int32(1) // default
	if replicas, ok := spec["replicas"].(float64); ok {
		desiredReplicas = int32(replicas)
	}

	// Get selector
	selector := make(map[string]string)
	if selectorMap, ok := spec["selector"].(map[string]interface{}); ok {
		if matchLabels, ok := selectorMap["matchLabels"].(map[string]interface{}); ok {
			for k, v := range matchLabels {
				if sv, ok := v.(string); ok {
					selector[k] = sv
				}
			}
		}
	}

	// Count existing pods matching this ReplicaSet
	existingPods := rsc.getPodsForReplicaSet(rsName, namespace)
	currentReplicas := int32(len(existingPods))

	fmt.Printf("[RS Controller] Reconciling ReplicaSet %s: desired=%d, current=%d\n", rsKey, desiredReplicas, currentReplicas)

	// Update ReplicaSet status
	rsc.updateReplicaSetStatus(rsName, namespace, currentReplicas, desiredReplicas)

	// Scale up or down as needed
	if currentReplicas < desiredReplicas {
		// Need to create pods
		diff := desiredReplicas - currentReplicas
		fmt.Printf("[RS Controller] Creating %d new pods for ReplicaSet %s\n", diff, rsName)
		for i := int32(0); i < diff; i++ {
			if err := rsc.createPodForReplicaSet(rsName, namespace, spec, selector, currentReplicas+i); err != nil {
				fmt.Printf("[RS Controller] Error creating pod: %v\n", err)
			}
		}
	} else if currentReplicas > desiredReplicas {
		// Need to delete pods
		diff := currentReplicas - desiredReplicas
		fmt.Printf("[RS Controller] Deleting %d pods for ReplicaSet %s\n", diff, rsName)
		for i := int32(0); i < diff && i < int32(len(existingPods)); i++ {
			if podName, ok := existingPods[i]["podName"].(string); ok {
				if err := rsc.deletePod(podName); err != nil {
					fmt.Printf("[RS Controller] Error deleting pod %s: %v\n", podName, err)
				}
			}
		}
	}
}

// getPodsForReplicaSet returns pods owned by this ReplicaSet
func (rsc *ReplicaSetController) getPodsForReplicaSet(rsName, namespace string) []map[string]interface{} {
	var pods []map[string]interface{}
	allPods := rsc.store.ListPods()

	fmt.Printf("[RS Controller] getPodsForReplicaSet: checking %d total pods for RS %s/%s\n", len(allPods), namespace, rsName)

	for _, podItem := range allPods {
		if pod, ok := podItem.(map[string]interface{}); ok {
			// Check if this pod is owned by our ReplicaSet
			if metadata, ok := pod["metadata"].(map[string]interface{}); ok {
				podName, _ := metadata["name"].(string)
				podNS, _ := metadata["namespace"].(string)

				// Check owner references - handle both []interface{} and []map[string]interface{}
				if ownerRefsRaw, ok := metadata["ownerReferences"]; ok {
					fmt.Printf("[RS Controller] getPodsForReplicaSet: pod %s/%s has ownerReferences\n", podNS, podName)
					var ownerRefs []interface{}
					switch v := ownerRefsRaw.(type) {
					case []interface{}:
						ownerRefs = v
					case []map[string]interface{}:
						for _, ref := range v {
							ownerRefs = append(ownerRefs, ref)
						}
					}

					for _, ownerRef := range ownerRefs {
						if ref, ok := ownerRef.(map[string]interface{}); ok {
							refName, _ := ref["name"].(string)
							refKind, _ := ref["kind"].(string)
							fmt.Printf("[RS Controller] getPodsForReplicaSet: checking ownerRef name=%s kind=%s against rsName=%s\n", refName, refKind, rsName)
							if refName == rsName && refKind == "ReplicaSet" {
								fmt.Printf("[RS Controller] getPodsForReplicaSet: found matching pod %s\n", podName)
								pods = append(pods, map[string]interface{}{
									"podName":   podName,
									"namespace": podNS,
								})
							}
						}
					}
				} else {
					fmt.Printf("[RS Controller] getPodsForReplicaSet: pod %s/%s has NO ownerReferences\n", podNS, podName)
				}
			}
		}
	}

	fmt.Printf("[RS Controller] getPodsForReplicaSet: found %d pods for RS %s/%s\n", len(pods), namespace, rsName)
	return pods
}

// createPodForReplicaSet creates a new pod for the ReplicaSet
func (rsc *ReplicaSetController) createPodForReplicaSet(rsName, namespace string, rsSpec map[string]interface{}, selector map[string]string, index int32) error {
	fmt.Printf("[RS Controller] createPodForReplicaSet called for %s/%s, index %d\n", namespace, rsName, index)

	// Extract pod template from ReplicaSet spec
	template, _ := rsSpec["template"].(map[string]interface{})
	templateMetadata, _ := template["metadata"].(map[string]interface{})
	templateSpec, _ := template["spec"].(map[string]interface{})

	// Build pod labels (merge template labels with selector labels)
	labels := make(map[string]string)
	if podLabels, ok := templateMetadata["labels"].(map[string]interface{}); ok {
		for k, v := range podLabels {
			if sv, ok := v.(string); ok {
				labels[k] = sv
			}
		}
	}
	// Add selector labels
	for k, v := range selector {
		labels[k] = v
	}

	// Generate unique pod name
	podName := fmt.Sprintf("%s-%d", rsName, time.Now().UnixNano())

	// Build owner reference
	ownerRef := resources.OwnerReference{
		APIVersion:         "apps/v1",
		Kind:               "ReplicaSet",
		Name:               rsName,
		UID:                fmt.Sprintf("rs-uid-%s", rsName),
		Controller:         true,
		BlockOwnerDeletion: true,
	}

	// Create the pod struct with ownerReferences
	pod := resources.Pod{
		Kind:       "Pod",
		APIVersion: "v1",
		Metadata: resources.ObjectMeta{
			Name:              podName,
			Namespace:         namespace,
			Labels:            labels,
			CreationTimestamp: time.Now().Format(time.RFC3339),
			OwnerReferences:   []resources.OwnerReference{ownerRef},
		},
		Spec: templateSpec,
	}

	// Store the pod
	if err := rsc.store.CreatePod(pod); err != nil {
		return fmt.Errorf("failed to create pod for ReplicaSet: %w", err)
	}

	fmt.Printf("[RS Controller] Created pod %s with ownerReferences for ReplicaSet %s\n", podName, rsName)

	// Trigger pod controller for lifecycle management
	if DefaultPodController != nil {
		DefaultPodController.OnPodCreated(pod)
	}

	return nil
}

// deletePod deletes a pod by name
func (rsc *ReplicaSetController) deletePod(podName string) error {
	// Cancel any active transitions
	if DefaultTransitionManager != nil {
		DefaultTransitionManager.CancelTransition("default", podName)
	}
	// Remove any templates
	if DefaultTemplateRegistry != nil {
		DefaultTemplateRegistry.RemoveTemplate("default", podName)
	}
	return rsc.store.DeletePod(podName)
}

// updateReplicaSetStatus updates the ReplicaSet status with current replica counts
func (rsc *ReplicaSetController) updateReplicaSetStatus(rsName, namespace string, current, desired int32) error {
	rs, err := rsc.store.GetReplicaSet(rsName)
	if err != nil {
		return err
	}

	// Update status
	status := map[string]interface{}{
		"replicas":             current,
		"availableReplicas":    current,
		"readyReplicas":        current,
		"fullyLabeledReplicas": current,
		"observedGeneration":   1,
	}

	rs["status"] = status

	// Convert back to ReplicaSet struct and update
	updatedRS := resources.ReplicaSet{
		Kind:       "ReplicaSet",
		APIVersion: "apps/v1",
		Metadata: resources.ObjectMeta{
			Name:      rsName,
			Namespace: namespace,
		},
		Spec:   rs["spec"],
		Status: status,
	}

	return rsc.store.UpdateReplicaSet(updatedRS)
}

// OnReplicaSetCreated is called when a new ReplicaSet is created
func (rsc *ReplicaSetController) OnReplicaSetCreated(rs resources.ReplicaSet) error {
	// Trigger immediate reconciliation
	rsMap := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      rs.GetName(),
			"namespace": rs.GetNamespace(),
		},
		"spec": rs.Spec,
	}
	rsc.reconcileReplicaSet(rsMap)
	return nil
}

// OnReplicaSetDeleted handles cleanup when a ReplicaSet is deleted
func (rsc *ReplicaSetController) OnReplicaSetDeleted(rsName, namespace string) error {
	// Find and delete all pods owned by this ReplicaSet
	pods := rsc.getPodsForReplicaSet(rsName, namespace)
	for _, pod := range pods {
		if podName, ok := pod["podName"].(string); ok {
			rsc.deletePod(podName)
		}
	}
	return nil
}

// DefaultReplicaSetController is the singleton instance
var DefaultReplicaSetController *ReplicaSetController

// InitReplicaSetController initializes the default ReplicaSet controller
func InitReplicaSetController(store *storage.InMemoryStore) {
	DefaultReplicaSetController = NewReplicaSetController(store)
	DefaultReplicaSetController.Start()
}
