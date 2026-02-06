package apis

import (
	"encoding/json"
	"io"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes/scheme"
	"mockernetes/internal/storage"
	"github.com/gin-gonic/gin"
)

// buildNamespaceList wraps store items (actual ns only) into K8s list response.
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

// CreateNamespace parses POST, validates, stores in in-mem map (error if exists); returns Status error for kubectl compat.
func CreateNamespace(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	// client-go validator: decode via scheme (enforces kind, apiVersion, basic struct)
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()
	obj, _, err := decoder.Decode(body, nil, nil)
	if err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	ns, ok := obj.(*corev1.Namespace)
	if !ok {
		WriteError(c, http.StatusBadRequest, "invalid namespace")
		return
	}
	// client-go/apimachinary validator for name (catches invalid like "Invalid-NS")
	if errs := validation.IsDNS1123Label(ns.Name); len(errs) != 0 {
		WriteError(c, http.StatusBadRequest, errs[0])
		return
	}
	// store; error if exists
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