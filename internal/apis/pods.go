package apis

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/util/validation"
	"mockernetes/internal/resources" // custom structs for mock control (no corev1)
	"mockernetes/internal/storage"
)

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

func ListPods(c *gin.Context) {
	items := storage.DefaultStore.ListPods()
	c.Data(http.StatusOK, "application/json", []byte(buildPodList(items)))
}

// CreatePod parses POST to custom resources.Pod struct (for mock control, no corev1/scheme).
// Validates, stores if not exists.
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
	c.JSON(http.StatusCreated, pod)
}
