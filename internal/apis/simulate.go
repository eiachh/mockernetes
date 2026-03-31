package apis

import (
	"io"
	"net/http"

	"encoding/json"

	"github.com/gin-gonic/gin"
	"mockernetes/internal/controllers"
)

// SimulatePodRequest wraps a transition template request for the API
type SimulatePodRequest struct {
	PodName     string                        `json:"podName"`
	Namespace   string                        `json:"namespace,omitempty"`
	Transitions []controllers.TransitionState `json:"transitions"`
}

// SimulatePodResponse is the response for a simulate request
type SimulatePodResponse struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	PodName   string `json:"podName"`
	Namespace string `json:"namespace"`
	Replaced  bool   `json:"replaced,omitempty"`
}

// TransitionTemplateResponse represents a registered template
type TransitionTemplateResponse struct {
	PodName     string                        `json:"podName"`
	Namespace   string                        `json:"namespace"`
	Transitions []controllers.TransitionState `json:"transitions"`
	CreatedAt   string                        `json:"createdAt"`
}

// SimulatePod handles POST /simulate/controller/pod
// Registers a transition template that will be applied when a pod with the given name is created
func SimulatePod(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, SimulatePodResponse{
			Success: false,
			Message: "Failed to read request body: " + err.Error(),
		})
		return
	}

	var req SimulatePodRequest
	if err := json.Unmarshal(body, &req); err != nil {
		c.JSON(http.StatusBadRequest, SimulatePodResponse{
			Success: false,
			Message: "Invalid JSON: " + err.Error(),
		})
		return
	}

	// Validate required fields
	if req.PodName == "" {
		c.JSON(http.StatusBadRequest, SimulatePodResponse{
			Success: false,
			Message: "podName is required",
		})
		return
	}

	if len(req.Transitions) == 0 {
		c.JSON(http.StatusBadRequest, SimulatePodResponse{
			Success: false,
			Message: "at least one transition is required",
		})
		return
	}

	// Use default namespace if not specified
	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	if controllers.DefaultTemplateRegistry == nil {
		c.JSON(http.StatusServiceUnavailable, SimulatePodResponse{
			Success: false,
			Message: "Template registry not initialized",
		})
		return
	}

	// Check if there's an existing template
	_, hadExisting := controllers.DefaultTemplateRegistry.GetTemplate(namespace, req.PodName)

	// Register the template
	template := controllers.TransitionTemplate{
		PodName:     req.PodName,
		Namespace:   namespace,
		Transitions: req.Transitions,
	}

	if err := controllers.DefaultTemplateRegistry.RegisterTemplate(template); err != nil {
		c.JSON(http.StatusInternalServerError, SimulatePodResponse{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	message := "Transition template registered"
	if hadExisting {
		message = "Transition template updated"
	}

	c.JSON(http.StatusOK, SimulatePodResponse{
		Success:   true,
		Message:   message,
		PodName:   req.PodName,
		Namespace: namespace,
		Replaced:  hadExisting,
	})
}

// CancelPodTransition handles DELETE /simulate/controller/pod/:name
// Removes a registered transition template for the specified pod
func CancelPodTransition(c *gin.Context) {
	podName := c.Param("name")
	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	if controllers.DefaultTemplateRegistry == nil {
		c.JSON(http.StatusServiceUnavailable, SimulatePodResponse{
			Success: false,
			Message: "Template registry not initialized",
		})
		return
	}

	removed := controllers.DefaultTemplateRegistry.RemoveTemplate(namespace, podName)

	if removed {
		// Also cancel any active transition for this pod
		if controllers.DefaultTransitionManager != nil {
			controllers.DefaultTransitionManager.CancelTransition(namespace, podName)
		}

		c.JSON(http.StatusOK, SimulatePodResponse{
			Success:   true,
			Message:   "Transition template removed",
			PodName:   podName,
			Namespace: namespace,
		})
	} else {
		c.JSON(http.StatusNotFound, SimulatePodResponse{
			Success:   false,
			Message:   "No transition template found for pod",
			PodName:   podName,
			Namespace: namespace,
		})
	}
}

// ListActiveTransitions handles GET /simulate/controller/pod
// Returns all registered transition templates
func ListActiveTransitions(c *gin.Context) {
	if controllers.DefaultTemplateRegistry == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "Template registry not initialized",
		})
		return
	}

	templates := controllers.DefaultTemplateRegistry.ListTemplates()

	response := make([]TransitionTemplateResponse, 0, len(templates))
	for _, t := range templates {
		response = append(response, TransitionTemplateResponse{
			PodName:     t.PodName,
			Namespace:   t.Namespace,
			Transitions: t.Transitions,
			CreatedAt:   t.CreatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"count":     len(response),
		"templates": response,
	})
}

// GetPodTransition handles GET /simulate/controller/pod/:name
// Returns the registered transition template for a specific pod if any
func GetPodTransition(c *gin.Context) {
	podName := c.Param("name")
	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	if controllers.DefaultTemplateRegistry == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "Template registry not initialized",
		})
		return
	}

	template, exists := controllers.DefaultTemplateRegistry.GetTemplate(namespace, podName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "No transition template found for pod",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"template": TransitionTemplateResponse{
			PodName:     template.PodName,
			Namespace:   template.Namespace,
			Transitions: template.Transitions,
			CreatedAt:   template.CreatedAt.Format("2006-01-02T15:04:05Z"),
		},
	})
}
