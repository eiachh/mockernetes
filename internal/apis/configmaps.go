package apis

import (
	"encoding/json"
	"io"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes/scheme"
	"mockernetes/internal/storage"
	"github.com/gin-gonic/gin"
)

// buildConfigMapList wraps store items into K8s list (similar to ns/pods).
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

// CreateConfigMap parses POST (like ns/pods), validates with client-go, stores if not exists.
func CreateConfigMap(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	decoder := serializer.NewCodecFactory(scheme.Scheme).UniversalDeserializer()
	obj, _, err := decoder.Decode(body, nil, nil)
	if err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	cm, ok := obj.(*corev1.ConfigMap)
	if !ok {
		WriteError(c, http.StatusBadRequest, "invalid configmap")
		return
	}
	if errs := validation.IsDNS1123Label(cm.Name); len(errs) != 0 {
		WriteError(c, http.StatusBadRequest, errs[0])
		return
	}
	if err := storage.DefaultStore.CreateConfigMap(cm); err != nil {
		WriteError(c, http.StatusConflict, err.Error())
		return
	}
	c.JSON(http.StatusCreated, cm)
}