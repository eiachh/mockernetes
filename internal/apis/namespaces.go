package apis

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"mockernetes/internal/resources" // custom structs for mock control (no corev1)
	"mockernetes/internal/storage"
)

// buildNamespaceList wraps store items (actual ns only) into K8s list response (uses custom resources.Namespace).
func buildNamespaceList(items []interface{}) string {
	list := map[string]interface{}{
		"kind":       "NamespaceList",
		"apiVersion": "v1",
		"metadata":   map[string]string{"resourceVersion": "1"},
		"items":      items,
	}
	b, _ := json.Marshal(list)
	return string(b)
}

func ListNamespaces(c *gin.Context) {
	items := storage.DefaultStore.ListNamespaces()
	c.Data(http.StatusOK, "application/json", []byte(buildNamespaceList(items)))
}

// CreateNamespace parses POST to custom resources.Namespace struct (for mock control, no corev1/scheme decode).
// Validates, stores; returns Status error for kubectl compat.
func CreateNamespace(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	// Unmarshal to custom struct (enforces mock shape; kind/apiVersion from body)
	var ns resources.Namespace
	if err := json.Unmarshal(body, &ns); err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	if ns.Kind == "" {
		WriteError(c, http.StatusBadRequest, "invalid namespace")
		return
	}
	// validator for name (catches invalid like "Invalid-NS")
	if errs := validation.IsDNS1123Label(ns.GetName()); len(errs) != 0 {
		WriteError(c, http.StatusBadRequest, errs[0])
		return
	}
	// store; error if exists (uses KubeObject impl)
	if err := storage.DefaultStore.CreateNamespace(ns); err != nil {
		WriteError(c, http.StatusConflict, err.Error()) // 409 for exists
		return
	}
	c.JSON(http.StatusCreated, ns)
}

// WriteError returns K8s Status for kubectl to parse/display error (e.g. on invalid ns).
func WriteError(c *gin.Context, code int, msg string) {
	status := metav1.Status{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Status",
			APIVersion: "v1",
		},
		Status:  metav1.StatusFailure,
		Message: msg,
		Reason:  metav1.StatusReasonInvalid,
		Code:    int32(code),
	}
	c.JSON(code, status)
}
