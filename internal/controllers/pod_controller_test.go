package controllers

import (
	"encoding/json"
	"testing"
	"time"

	"mockernetes/internal/resources"
	"mockernetes/internal/storage"
)

func TestPodControllerLifecycle(t *testing.T) {
	store := storage.NewInMemoryStore()
	controller := NewPodController(store)
	controller.Start()

	// Create a test pod
	pod := resources.Pod{
		Kind:       "Pod",
		APIVersion: "v1",
		Metadata: resources.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
		},
		Spec: map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"name":  "nginx",
					"image": "nginx:latest",
				},
			},
		},
	}

	// Store the pod
	if err := store.CreatePod(pod); err != nil {
		t.Fatalf("Failed to create pod: %v", err)
	}

	// Call OnPodCreated
	if err := controller.OnPodCreated(pod); err != nil {
		t.Fatalf("Failed to call OnPodCreated: %v", err)
	}

	// Verify the pod status was updated
	storedPod, err := store.GetPod("test-pod")
	if err != nil {
		t.Fatalf("Failed to get pod: %v", err)
	}

	status := storedPod["status"].(map[string]interface{})
	phase := status["phase"].(string)
	if phase != "Pending" {
		t.Errorf("Expected phase Pending, got %s", phase)
	}

	// Call OnPodStarted
	if err := controller.OnPodStarted(pod); err != nil {
		t.Fatalf("Failed to call OnPodStarted: %v", err)
	}

	// Verify the pod status was updated to Running
	storedPod, err = store.GetPod("test-pod")
	if err != nil {
		t.Fatalf("Failed to get pod: %v", err)
	}

	status = storedPod["status"].(map[string]interface{})
	phase = status["phase"].(string)
	if phase != "Running" {
		t.Errorf("Expected phase Running, got %s", phase)
	}

	// Verify the JSON output
	listItems := store.ListPods()
	if len(listItems) != 1 {
		t.Fatalf("Expected 1 pod, got %d", len(listItems))
	}

	jsonBytes, _ := json.MarshalIndent(listItems[0], "", "  ")
	t.Logf("Pod JSON:\n%s", string(jsonBytes))
}

func TestPodStatusJSON(t *testing.T) {
	now := time.Now()
	status := PodStatus{
		Phase:     PodRunning,
		PodIP:     "10.244.0.1",
		HostIP:    "127.0.0.1",
		StartTime: &now,
		Conditions: []PodCondition{
			{
				Type:               "Ready",
				Status:             "True",
				LastTransitionTime: &now,
			},
		},
	}

	b, err := json.Marshal(status)
	if err != nil {
		t.Fatalf("Failed to marshal status: %v", err)
	}

	var unmarshaled map[string]interface{}
	if err := json.Unmarshal(b, &unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal status: %v", err)
	}

	if unmarshaled["phase"] != "Running" {
		t.Errorf("Expected phase Running, got %v", unmarshaled["phase"])
	}

	if unmarshaled["podIP"] != "10.244.0.1" {
		t.Errorf("Expected podIP 10.244.0.1, got %v", unmarshaled["podIP"])
	}
}
