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

// buildConfigMapList wraps store items into K8s list (similar to ns/pods; uses custom resources.ConfigMap).
func buildConfigMapList(items []interface{}) string {
	list := map[string]interface{}{
		"kind":       "ConfigMapList",
		"apiVersion": "v1",
		"metadata":   map[string]string{"resourceVersion": "1"},
		"items":      items,
	}
	b, _ := json.Marshal(list)
	return string(b)
}

func ListConfigMaps(c *gin.Context) {
	items := storage.DefaultStore.ListConfigMaps()
	c.Data(http.StatusOK, "application/json", []byte(buildConfigMapList(items)))
}

// CreateConfigMap parses POST to custom resources.ConfigMap struct (for mock control, no corev1/scheme).
// Validates, stores if not exists.
func CreateConfigMap(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	// Unmarshal to custom struct
	var cm resources.ConfigMap
	if err := json.Unmarshal(body, &cm); err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	if cm.Kind == "" {
		WriteError(c, http.StatusBadRequest, "invalid configmap")
		return
	}
	// validator for name
	if errs := validation.IsDNS1123Label(cm.GetName()); len(errs) != 0 {
		WriteError(c, http.StatusBadRequest, errs[0])
		return
	}
	// store; uses KubeObject impl from custom struct
	if err := storage.DefaultStore.CreateConfigMap(cm); err != nil {
		WriteError(c, http.StatusConflict, err.Error())
		return
	}
	c.JSON(http.StatusCreated, cm)
}
