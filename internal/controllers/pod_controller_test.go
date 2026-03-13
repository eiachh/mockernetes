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
	delay := 100 * time.Millisecond
	controller := NewPodController(store, delay)
	controller.Start()
	defer controller.Stop()

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

	// Call OnPodCreated — sets Pending immediately, schedules Running after delay
	if err := controller.OnPodCreated(pod); err != nil {
		t.Fatalf("Failed to call OnPodCreated: %v", err)
	}

	// Verify the pod is Pending immediately
	storedPod, err := store.GetPod("test-pod")
	if err != nil {
		t.Fatalf("Failed to get pod: %v", err)
	}

	status := storedPod["status"].(map[string]interface{})
	phase := status["phase"].(string)
	if phase != "Pending" {
		t.Errorf("Expected phase Pending, got %s", phase)
	}

	// Wait for the async transition to Running
	time.Sleep(delay + 50*time.Millisecond)

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

func TestPodControllerIndependentPods(t *testing.T) {
	store := storage.NewInMemoryStore()
	delay := 100 * time.Millisecond
	controller := NewPodController(store, delay)
	controller.Start()
	defer controller.Stop()

	// Create two pods
	pod1 := resources.Pod{
		Kind: "Pod", APIVersion: "v1",
		Metadata: resources.ObjectMeta{Name: "pod-a", Namespace: "default"},
		Spec: map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{"name": "nginx", "image": "nginx"},
			},
		},
	}
	pod2 := resources.Pod{
		Kind: "Pod", APIVersion: "v1",
		Metadata: resources.ObjectMeta{Name: "pod-b", Namespace: "default"},
		Spec: map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{"name": "redis", "image": "redis"},
			},
		},
	}

	// Store and create both pods without blocking
	store.CreatePod(pod1)
	store.CreatePod(pod2)
	if err := controller.OnPodCreated(pod1); err != nil {
		t.Fatalf("Failed OnPodCreated for pod-a: %v", err)
	}
	if err := controller.OnPodCreated(pod2); err != nil {
		t.Fatalf("Failed OnPodCreated for pod-b: %v", err)
	}

	// Both should be Pending immediately
	for _, name := range []string{"pod-a", "pod-b"} {
		p, _ := store.GetPod(name)
		status := p["status"].(map[string]interface{})
		if status["phase"] != "Pending" {
			t.Errorf("Expected %s to be Pending, got %v", name, status["phase"])
		}
	}

	// Wait for both to transition
	time.Sleep(delay + 50*time.Millisecond)

	// Both should be Running
	for _, name := range []string{"pod-a", "pod-b"} {
		p, _ := store.GetPod(name)
		status := p["status"].(map[string]interface{})
		if status["phase"] != "Running" {
			t.Errorf("Expected %s to be Running, got %v", name, status["phase"])
		}
	}
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
