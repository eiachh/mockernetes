package apis

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"mockernetes/internal/controllers"
	"mockernetes/internal/resources"
	"mockernetes/internal/storage"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestListPods(t *testing.T) {
	// Create a test pod first
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
					"image": "nginx",
				},
			},
		},
	}
	storage.DefaultStore.CreatePod(pod)

	// Run pod through the controller lifecycle so status is set
	// Use a short delay so the test doesn't wait long
	controllers.DefaultPodController = controllers.NewPodController(storage.DefaultStore, 20*time.Second)
	controllers.DefaultPodController.Start()
	controllers.DefaultPodController.OnPodCreated(pod)

	// Wait for async transition to Running
	time.Sleep(100 * time.Millisecond)

	// Create request
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/pods", nil)

	ListPods(c)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response["kind"] != "PodList" {
		t.Errorf("Expected kind PodList, got %v", response["kind"])
	}

	items, ok := response["items"].([]interface{})
	if !ok {
		t.Fatal("Expected items array")
	}
	if len(items) == 0 {
		t.Error("Expected at least one pod")
	}

	// Check that the pod has status set by the controller
	firstPod := items[0].(map[string]interface{})
	status := firstPod["status"].(map[string]interface{})
	if status["phase"] != "Running" {
		t.Errorf("Expected phase Running, got %v", status["phase"])
	}
}

func TestWatchPods(t *testing.T) {
	// Create a test pod first
	pod := resources.Pod{
		Kind:       "Pod",
		APIVersion: "v1",
		Metadata: resources.ObjectMeta{
			Name:      "watch-test-pod",
			Namespace: "default",
		},
		Spec: map[string]interface{}{
			"containers": []interface{}{
				map[string]interface{}{
					"name":  "nginx",
					"image": "nginx",
				},
			},
		},
	}
	storage.DefaultStore.CreatePod(pod)

	// Create request with watch=true
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/api/v1/pods?watch=true", nil)
	c.Request = c.Request.WithContext(c.Request.Context())

	// Run watch in goroutine with timeout
	done := make(chan bool)
	go func() {
		WatchPods(c)
		done <- true
	}()

	// Wait a bit for initial events
	time.Sleep(100 * time.Millisecond)

	// Check we got some output
	if w.Body.Len() == 0 {
		t.Error("Expected some watch output")
	}

	// Parse the first line as a watch event
	scanner := bufio.NewScanner(strings.NewReader(w.Body.String()))
	if scanner.Scan() {
		var event WatchEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			t.Fatalf("Failed to parse watch event: %v", err)
		}
		if event.Type != "ADDED" {
			t.Errorf("Expected ADDED event, got %s", event.Type)
		}
		if event.Object == nil {
			t.Error("Expected object in watch event")
		}
	}
}
