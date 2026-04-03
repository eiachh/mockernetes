package controllers

import (
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"mockernetes/internal/resources"
	"mockernetes/internal/storage"
)

// DeploymentController manages the lifecycle of Deployments and their ReplicaSets
type DeploymentController struct {
	store       *storage.InMemoryStore
	mu          sync.RWMutex
	stopCh      chan struct{}
	reconciling map[string]bool // track which deployments are being reconciled
	reconcileMu sync.Mutex
}

// NewDeploymentController creates a new DeploymentController
func NewDeploymentController(store *storage.InMemoryStore) *DeploymentController {
	return &DeploymentController{
		store:       store,
		stopCh:      make(chan struct{}),
		reconciling: make(map[string]bool),
	}
}

// Start starts the controller's reconciliation loop
func (dc *DeploymentController) Start() {
	go dc.reconcileLoop()
}

// Stop stops the controller
func (dc *DeploymentController) Stop() {
	close(dc.stopCh)
}

// reconcileLoop periodically reconciles Deployments
func (dc *DeploymentController) reconcileLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-dc.stopCh:
			return
		case <-ticker.C:
			dc.reconcileAll()
		}
	}
}

// reconcileAll reconciles all Deployments
func (dc *DeploymentController) reconcileAll() {
	deployments := dc.store.ListDeployments()
	for _, deployItem := range deployments {
		if deploy, ok := deployItem.(map[string]interface{}); ok {
			dc.reconcileDeployment(deploy)
		}
	}
}

// reconcileDeployment ensures the Deployment has the correct ReplicaSet
func (dc *DeploymentController) reconcileDeployment(deploy map[string]interface{}) {
	// Extract Deployment info
	metadata, _ := deploy["metadata"].(map[string]interface{})
	spec, _ := deploy["spec"].(map[string]interface{})

	deployName, _ := metadata["name"].(string)
	namespace, _ := metadata["namespace"].(string)
	if namespace == "" {
		namespace = "default"
	}

	// Prevent concurrent reconciliation of the same Deployment
	deployKey := fmt.Sprintf("%s/%s", namespace, deployName)
	dc.reconcileMu.Lock()
	if dc.reconciling[deployKey] {
		dc.reconcileMu.Unlock()
		fmt.Printf("[Deployment Controller] Skipping reconciliation of %s - already in progress\n", deployKey)
		return
	}
	dc.reconciling[deployKey] = true
	dc.reconcileMu.Unlock()

	defer func() {
		dc.reconcileMu.Lock()
		delete(dc.reconciling, deployKey)
		dc.reconcileMu.Unlock()
	}()

	// Get desired replicas (default: 1)
	desiredReplicas := int32(1)
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

	// Get pod template
	template, _ := spec["template"].(map[string]interface{})

	// Generate pod-template-hash from template
	templateHash := computeTemplateHash(template)

	// Calculate expected ReplicaSet name
	rsName := fmt.Sprintf("%s-%s", deployName, templateHash)

	// Check if ReplicaSet exists
	existingRS, err := dc.store.GetReplicaSet(rsName)

	if err != nil {
		// ReplicaSet doesn't exist, create it
		fmt.Printf("[Deployment Controller] Creating ReplicaSet %s for Deployment %s\n", rsName, deployName)
		if err := dc.createReplicaSetForDeployment(deployName, namespace, rsName, desiredReplicas, selector, template, templateHash); err != nil {
			fmt.Printf("[Deployment Controller] Error creating ReplicaSet: %v\n", err)
			return
		}
	} else {
		// ReplicaSet exists, check if replicas need updating
		rsSpec, _ := existingRS["spec"].(map[string]interface{})
		currentReplicas := int32(1)
		if replicas, ok := rsSpec["replicas"].(float64); ok {
			currentReplicas = int32(replicas)
		}

		if currentReplicas != desiredReplicas {
			fmt.Printf("[Deployment Controller] Scaling ReplicaSet %s: %d -> %d\n", rsName, currentReplicas, desiredReplicas)
			if err := dc.updateReplicaSetReplicas(rsName, desiredReplicas); err != nil {
				fmt.Printf("[Deployment Controller] Error updating ReplicaSet: %v\n", err)
				return
			}
		}
	}

	// Update Deployment status
	dc.updateDeploymentStatus(deployName, namespace, desiredReplicas)
}

// computeTemplateHash generates a simple hash from the pod template
func computeTemplateHash(template map[string]interface{}) string {
	if template == nil {
		return "00000000"
	}
	// Simple hash based on template content
	data := fmt.Sprintf("%v", template)
	h := fnv.New32a()
	h.Write([]byte(data))
	return fmt.Sprintf("%08x", h.Sum32())[:8]
}

// createReplicaSetForDeployment creates a new ReplicaSet for the Deployment
func (dc *DeploymentController) createReplicaSetForDeployment(deployName, namespace, rsName string, replicas int32, selector map[string]string, template map[string]interface{}, templateHash string) error {
	// Build owner reference
	ownerRef := resources.OwnerReference{
		APIVersion:         "apps/v1",
		Kind:               "Deployment",
		Name:               deployName,
		UID:                fmt.Sprintf("deploy-uid-%s", deployName),
		Controller:         true,
		BlockOwnerDeletion: true,
	}

	// Build labels with pod-template-hash
	labels := make(map[string]string)
	if templateMetadata, ok := template["metadata"].(map[string]interface{}); ok {
		if podLabels, ok := templateMetadata["labels"].(map[string]interface{}); ok {
			for k, v := range podLabels {
				if sv, ok := v.(string); ok {
					labels[k] = sv
				}
			}
		}
	}
	// Add pod-template-hash label
	labels["pod-template-hash"] = templateHash

	// Build ReplicaSet spec
	rsSpec := map[string]interface{}{
		"replicas": replicas,
		"selector": map[string]interface{}{
			"matchLabels": labels,
		},
		"template": template,
	}

	// Create the ReplicaSet
	rs := resources.ReplicaSet{
		Kind:       "ReplicaSet",
		APIVersion: "apps/v1",
		Metadata: resources.ObjectMeta{
			Name:              rsName,
			Namespace:         namespace,
			Labels:            labels,
			CreationTimestamp: time.Now().Format(time.RFC3339),
			OwnerReferences:   []resources.OwnerReference{ownerRef},
		},
		Spec: rsSpec,
	}

	if err := dc.store.CreateReplicaSet(rs); err != nil {
		return fmt.Errorf("failed to create ReplicaSet: %w", err)
	}

	fmt.Printf("[Deployment Controller] Created ReplicaSet %s for Deployment %s\n", rsName, deployName)

	// Trigger ReplicaSet controller for immediate reconciliation
	if DefaultReplicaSetController != nil {
		DefaultReplicaSetController.OnReplicaSetCreated(rs)
	}

	return nil
}

// updateReplicaSetReplicas updates the replicas count of an existing ReplicaSet
func (dc *DeploymentController) updateReplicaSetReplicas(rsName string, replicas int32) error {
	rs, err := dc.store.GetReplicaSet(rsName)
	if err != nil {
		return err
	}

	// Update spec.replicas
	if spec, ok := rs["spec"].(map[string]interface{}); ok {
		spec["replicas"] = replicas
	}

	// Convert back to ReplicaSet struct
	updatedRS := resources.ReplicaSet{
		Kind:       "ReplicaSet",
		APIVersion: "apps/v1",
		Metadata: resources.ObjectMeta{
			Name:      rsName,
			Namespace: "default",
		},
		Spec:   rs["spec"],
		Status: rs["status"],
	}

	if err := dc.store.UpdateReplicaSet(updatedRS); err != nil {
		return err
	}

	// Trigger reconciliation
	if DefaultReplicaSetController != nil {
		DefaultReplicaSetController.OnReplicaSetCreated(updatedRS)
	}

	return nil
}

// updateDeploymentStatus updates the Deployment status
func (dc *DeploymentController) updateDeploymentStatus(deployName, namespace string, replicas int32) error {
	deploy, err := dc.store.GetDeployment(deployName)
	if err != nil {
		return err
	}

	status := map[string]interface{}{
		"replicas":             replicas,
		"availableReplicas":    replicas,
		"readyReplicas":        replicas,
		"updatedReplicas":      replicas,
		"observedGeneration":   1,
	}

	deploy["status"] = status

	updatedDeploy := resources.Deployment{
		Kind:       "Deployment",
		APIVersion: "apps/v1",
		Metadata: resources.ObjectMeta{
			Name:      deployName,
			Namespace: namespace,
		},
		Spec:   deploy["spec"],
		Status: status,
	}

	return dc.store.UpdateDeployment(updatedDeploy)
}

// OnDeploymentCreated is called when a new Deployment is created
func (dc *DeploymentController) OnDeploymentCreated(deploy resources.Deployment) error {
	// Trigger immediate reconciliation
	deployMap := map[string]interface{}{
		"metadata": map[string]interface{}{
			"name":      deploy.GetName(),
			"namespace": deploy.GetNamespace(),
		},
		"spec": deploy.Spec,
	}
	dc.reconcileDeployment(deployMap)
	return nil
}

// OnDeploymentDeleted handles cleanup when a Deployment is deleted
func (dc *DeploymentController) OnDeploymentDeleted(deployName, namespace string) error {
	// Find and delete all ReplicaSets owned by this Deployment
	allRS := dc.store.ListReplicaSets()
	for _, rsItem := range allRS {
		if rs, ok := rsItem.(map[string]interface{}); ok {
			if metadata, ok := rs["metadata"].(map[string]interface{}); ok {
				// Check owner references
				if ownerRefsRaw, ok := metadata["ownerReferences"]; ok {
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
							if refName == deployName && refKind == "Deployment" {
								rsName, _ := metadata["name"].(string)
								fmt.Printf("[Deployment Controller] Deleting ReplicaSet %s (owned by Deployment %s)\n", rsName, deployName)
								dc.store.DeleteReplicaSet(rsName)
							}
						}
					}
				}
			}
		}
	}
	return nil
}

// DefaultDeploymentController is the singleton instance
var DefaultDeploymentController *DeploymentController

// InitDeploymentController initializes the default Deployment controller
func InitDeploymentController(store *storage.InMemoryStore) {
	DefaultDeploymentController = NewDeploymentController(store)
	DefaultDeploymentController.Start()
}
