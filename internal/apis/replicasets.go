package apis

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/util/validation"
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

// CreateReplicaSet parses POST to custom resources.ReplicaSet struct (for mock control, no appsv1/scheme).
// Note: in mock, RS allowed direct (unlike real K8s managed by Deployments).
// Validates, stores if not exists.
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
	c.JSON(http.StatusCreated, rs)
}
