package apis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/util/validation"
	"mockernetes/internal/controllers" // pod lifecycle controller
	"mockernetes/internal/resources"    // custom structs for mock control (no corev1)
	"mockernetes/internal/storage"
)

// TableResponse represents the Kubernetes Table format for kubectl get
type TableResponse struct {
	Kind       string        `json:"kind"`
	APIVersion string        `json:"apiVersion"`
	Metadata   TableMetadata `json:"metadata"`
	ColumnDefinitions []ColumnDefinition `json:"columnDefinitions"`
	Rows       []TableRow    `json:"rows"`
}

// TableMetadata represents metadata for Table response
type TableMetadata struct {
	ResourceVersion string `json:"resourceVersion"`
}

// ColumnDefinition defines a column in the table
type ColumnDefinition struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Format      string `json:"format"`
	Description string `json:"description"`
	Priority    int    `json:"priority"`
}

// TableRow represents a row in the table
type TableRow struct {
	Cells    []interface{} `json:"cells"`
	Object   interface{}   `json:"object,omitempty"`
}

// PodTableColumns defines the standard columns for kubectl get pods
var PodTableColumns = []ColumnDefinition{
	{Name: "Name", Type: "string", Format: "name", Description: "Name must be unique within a namespace.", Priority: 0},
	{Name: "Ready", Type: "string", Format: "", Description: "The ready condition status of the pod.", Priority: 0},
	{Name: "Status", Type: "string", Format: "", Description: "The phase of the pod.", Priority: 0},
	{Name: "Restarts", Type: "integer", Format: "", Description: "The number of times the pod has been restarted.", Priority: 0},
	{Name: "Age", Type: "string", Format: "", Description: "The age of the pod.", Priority: 0},
	{Name: "IP", Type: "string", Format: "", Description: "IP address allocated to the pod.", Priority: 1},
	{Name: "Node", Type: "string", Format: "", Description: "Node name where the pod is running.", Priority: 1},
	{Name: "Nominated Node", Type: "string", Format: "", Description: "Nominated node name for the pod.", Priority: 1},
	{Name: "Readiness Gates", Type: "string", Format: "", Description: "Readiness gates for the pod.", Priority: 1},
}

// isTableRequest checks if the Accept header requests Table format
func isTableRequest(acceptHeader string) bool {
	return strings.Contains(acceptHeader, "as=Table") ||
		strings.Contains(acceptHeader, "application/json;as=Table")
}

// buildPodTable creates a Table response from pod items
func buildPodTable(items []interface{}) TableResponse {
	rows := make([]TableRow, 0, len(items))

	for _, item := range items {
		pod, ok := item.(map[string]interface{})
		if !ok {
			continue
		}

		cells := buildPodCells(pod)
		rows = append(rows, TableRow{
			Cells:  cells,
			Object: pod,
		})
	}

	return TableResponse{
		Kind:             "Table",
		APIVersion:       "meta.k8s.io/v1",
		Metadata:         TableMetadata{ResourceVersion: "1"},
		ColumnDefinitions: PodTableColumns,
		Rows:             rows,
	}
}

// buildPodCells creates the cell values for a pod row
func buildPodCells(pod map[string]interface{}) []interface{} {
	// Extract metadata
	name := ""
	creationTimestamp := ""
	if meta, ok := pod["metadata"].(map[string]interface{}); ok {
		if n, ok := meta["name"].(string); ok {
			name = n
		}
		if ct, ok := meta["creationTimestamp"].(string); ok {
			creationTimestamp = ct
		}
	}

	// Extract status
	phase := "Unknown"
	podIP := ""
	readyCount := 0
	totalContainers := 0
	restartCount := int64(0)

	if status, ok := pod["status"].(map[string]interface{}); ok {
		if p, ok := status["phase"].(string); ok {
			phase = p
		}
		if ip, ok := status["podIP"].(string); ok {
			podIP = ip
		}

		// Count containers and ready status
		if containerStatuses, ok := status["containerStatuses"].([]interface{}); ok {
			totalContainers = len(containerStatuses)
			for _, cs := range containerStatuses {
				if containerStatus, ok := cs.(map[string]interface{}); ok {
					if ready, ok := containerStatus["ready"].(bool); ok && ready {
						readyCount++
					}
					if rc, ok := containerStatus["restartCount"].(int64); ok {
						restartCount += rc
					} else if rcFloat, ok := containerStatus["restartCount"].(float64); ok {
						restartCount += int64(rcFloat)
					}
				}
			}
		}
	}

	// Calculate age
	age := ""
	if creationTimestamp != "" {
		if t, err := time.Parse(time.RFC3339, creationTimestamp); err == nil {
			age = formatAge(time.Since(t))
		}
	}

	// Build ready string
	ready := fmt.Sprintf("%d/%d", readyCount, totalContainers)

	// Return cells matching column definitions
	return []interface{}{
		name,           // Name
		ready,          // Ready
		phase,          // Status
		restartCount,   // Restarts
		age,            // Age
		podIP,          // IP
		"",             // Node
		"",             // Nominated Node
		"",             // Readiness Gates
	}
}

// formatAge formats duration as a human-readable age string
func formatAge(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// buildPodList wraps store items into K8s list (similar to ns; uses custom resources.Pod).
func buildPodList(items []interface{}) string {
	list := map[string]interface{}{
		"kind":       "PodList",
		"apiVersion": "v1",
		"metadata":   map[string]string{"resourceVersion": "1"},
		"items":      items,
	}
	b, _ := json.Marshal(list)
	return string(b)
}

// WatchEvent represents a single watch event
type WatchEvent struct {
	Type   string      `json:"type"`
	Object interface{} `json:"object"`
}

func ListPods(c *gin.Context) {
	// Check if this is a watch request
	if c.Query("watch") == "true" {
		WatchPods(c)
		return
	}

	items := storage.DefaultStore.ListPods()

	// Check if client requests Table format (kubectl get)
	acceptHeader := c.GetHeader("Accept")
	if isTableRequest(acceptHeader) {
		table := buildPodTable(items)
		c.JSON(http.StatusOK, table)
		return
	}

	c.Data(http.StatusOK, "application/json", []byte(buildPodList(items)))
}

// WatchPods handles watch requests for pods
// Returns a stream of watch events (ADDED, MODIFIED, DELETED)
func WatchPods(c *gin.Context) {
	// Set headers for streaming response
	c.Header("Content-Type", "application/json")
	c.Header("Transfer-Encoding", "chunked")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")

	// Get the response writer for streaming
	w := c.Writer

	// Flush helper
	flusher, ok := w.(http.Flusher)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "streaming not supported"})
		return
	}

	// Get namespace filter if provided
	namespace := c.Param("namespace")

	// Send initial ADDED events for all existing pods
	items := storage.DefaultStore.ListPods()
	for _, item := range items {
		// Filter by namespace if specified
		if namespace != "" {
			if obj, ok := item.(map[string]interface{}); ok {
				if meta, ok := obj["metadata"].(map[string]interface{}); ok {
					if ns, ok := meta["namespace"].(string); ok && ns != namespace {
						continue
					}
				}
			}
		}

		event := WatchEvent{
			Type:   "ADDED",
			Object: item,
		}
		data, _ := json.Marshal(event)
		fmt.Fprintf(w, "%s\n", data)
		flusher.Flush()
	}

	// Keep the connection alive and send periodic updates
	// In a real implementation, this would watch for actual changes
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Send periodic MODIFIED events to keep the connection alive
	// and simulate status updates
	for {
		select {
		case <-c.Request.Context().Done():
			// Client disconnected
			return
		case <-ticker.C:
			// Send MODIFIED events for all pods with updated status
			items := storage.DefaultStore.ListPods()
			for _, item := range items {
				// Filter by namespace if specified
				if namespace != "" {
					if obj, ok := item.(map[string]interface{}); ok {
						if meta, ok := obj["metadata"].(map[string]interface{}); ok {
							if ns, ok := meta["namespace"].(string); ok && ns != namespace {
								continue
							}
						}
					}
				}

				event := WatchEvent{
					Type:   "MODIFIED",
					Object: item,
				}
				data, _ := json.Marshal(event)
				fmt.Fprintf(w, "%s\n", data)
				flusher.Flush()
			}
		}
	}
}

// CreatePod parses POST to custom resources.Pod struct (for mock control, no corev1/scheme).
// Validates, stores if not exists, and triggers the controller for lifecycle management.
func CreatePod(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	// Unmarshal to custom struct (enforces mock shape; kind/apiVersion from body)
	var pod resources.Pod
	if err := json.Unmarshal(body, &pod); err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	if pod.Kind == "" {
		WriteError(c, http.StatusBadRequest, "invalid pod")
		return
	}
	// validator for name
	if errs := validation.IsDNS1123Label(pod.GetName()); len(errs) != 0 {
		WriteError(c, http.StatusBadRequest, errs[0])
		return
	}
	// store; uses KubeObject impl from custom struct
	if err := storage.DefaultStore.CreatePod(pod); err != nil {
		WriteError(c, http.StatusConflict, err.Error())
		return
	}

	// Notify the controller to manage the pod lifecycle
	// The controller will update the pod status to Pending, then to Running
	if controllers.DefaultPodController != nil {
		// Set initial Pending status synchronously
		if err := controllers.DefaultPodController.OnPodCreated(pod); err != nil {
			// Log error but don't fail the creation - the pod is already stored
			_ = err
		}

		// Transition to Running immediately for mock (no real containers to start)
		// In a real controller, this would be async watching actual container states
		if err := controllers.DefaultPodController.OnPodStarted(pod); err != nil {
			_ = err
		}
	}

	// Return the pod with status from storage
	storedPod, err := storage.DefaultStore.GetPod(pod.GetName())
	if err != nil {
		// Fallback to returning the original pod if get fails
		c.JSON(http.StatusCreated, pod)
		return
	}
	c.JSON(http.StatusCreated, storedPod)
}
