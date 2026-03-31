package apis

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/util/validation"
	"mockernetes/internal/controllers"
	"mockernetes/internal/resources" // custom structs for mock control (no appsv1)
	"mockernetes/internal/storage"
)

// buildReplicaSetList wraps store items into K8s list (similar to pods/deployments; uses custom resources.ReplicaSet structs).
func buildReplicaSetList(items []interface{}) string {
	list := map[string]interface{}{
		"kind":       "ReplicaSetList",
		"apiVersion": "apps/v1",
		"metadata":   map[string]string{"resourceVersion": "1"},
		"items":      items,
	}
	b, _ := json.Marshal(list)
	return string(b)
}

func ListReplicaSets(c *gin.Context) {
	// ignore :namespace if present (mock simplification like pods)
	items := storage.DefaultStore.ListReplicaSets()
	c.Data(http.StatusOK, "application/json", []byte(buildReplicaSetList(items)))
}

// GetReplicaSet handles GET /apis/apps/v1/namespaces/:namespace/replicasets/:name
func GetReplicaSet(c *gin.Context) {
	rsName := c.Param("name")

	rs, err := storage.DefaultStore.GetReplicaSet(rsName)
	if err != nil {
		WriteError(c, http.StatusNotFound, fmt.Sprintf("replicasets.apps \"%s\" not found", rsName))
		return
	}

	c.JSON(http.StatusOK, rs)
}

// CreateReplicaSet parses POST to custom resources.ReplicaSet struct (for mock control, no appsv1/scheme).
// Note: in mock, RS allowed direct (unlike real K8s managed by Deployments).
// Validates, stores if not exists, and triggers the controller to manage pods.
func CreateReplicaSet(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	// Unmarshal to custom struct
	var rs resources.ReplicaSet
	if err := json.Unmarshal(body, &rs); err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	if rs.Kind == "" {
		WriteError(c, http.StatusBadRequest, "invalid replicaset")
		return
	}
	// basic name validation (DNS label)
	if errs := validation.IsDNS1123Label(rs.GetName()); len(errs) != 0 {
		WriteError(c, http.StatusBadRequest, errs[0])
		return
	}
	// store; uses KubeObject impl from custom struct
	if err := storage.DefaultStore.CreateReplicaSet(rs); err != nil {
		WriteError(c, http.StatusConflict, err.Error())
		return
	}

	// Notify the ReplicaSet controller to manage pods
	if controllers.DefaultReplicaSetController != nil {
		controllers.DefaultReplicaSetController.OnReplicaSetCreated(rs)
	}

	// Return the stored ReplicaSet with any status updates
	storedRS, err := storage.DefaultStore.GetReplicaSet(rs.GetName())
	if err != nil {
		c.JSON(http.StatusCreated, rs)
		return
	}
	c.JSON(http.StatusCreated, storedRS)
}

// DeleteReplicaSet handles DELETE /apis/apps/v1/namespaces/:namespace/replicasets/:name
func DeleteReplicaSet(c *gin.Context) {
	rsName := c.Param("name")
	namespace := c.Param("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Check if ReplicaSet exists
	rs, err := storage.DefaultStore.GetReplicaSet(rsName)
	if err != nil {
		WriteError(c, http.StatusNotFound, fmt.Sprintf("replicasets.apps \"%s\" not found", rsName))
		return
	}

	// Notify the controller to clean up pods
	if controllers.DefaultReplicaSetController != nil {
		controllers.DefaultReplicaSetController.OnReplicaSetDeleted(rsName, namespace)
	}

	// Delete the ReplicaSet
	if err := storage.DefaultStore.DeleteReplicaSet(rsName); err != nil {
		WriteError(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, rs)
}
