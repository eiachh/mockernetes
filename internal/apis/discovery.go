package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Discovery responses (hardcoded valid K8s shapes)
const (
	apiJSON  = `{"kind":"APIVersions","versions":["v1"]}`
	apisJSON = `{"kind":"APIGroupList","groups":[{"name":"apps","versions":[{"groupVersion":"apps/v1","version":"v1"}],"preferredVersion":{"groupVersion":"apps/v1","version":"v1"}}]}`
	// namespaces with canonical form + shortNames["ns"] for kubectl get ns; plus common resources
	apiV1JSON = `{"kind":"APIResourceList","groupVersion":"v1","resources":[{"name":"namespaces","singularName":"namespace","namespaced":false,"kind":"Namespace","verbs":["create","delete","get","list","patch","update","watch"],"shortNames":["ns"],"categories":["all"]},{"name":"pods","singularName":"pod","namespaced":true,"kind":"Pod","verbs":["get","list"],"shortNames":["po"]},{"name":"configmaps","singularName":"configmap","namespaced":true,"kind":"ConfigMap","verbs":["get","list"],"shortNames":["cm"]}]}`

	// apps/v1 resources (deployments + replicasets; expanded for full kubectl discovery compat.
	// shortNames, verbs mirror pods/cm; enables `kubectl get deploy,rs` without errors.
	// Uses custom struct JSON shapes from k8s pkg for mock control.
	// Note: lists create verbs too for POST support, though mock focuses list/create like pods).
	appsV1JSON = `{"kind":"APIResourceList","groupVersion":"apps/v1","resources":[{"name":"deployments","singularName":"deployment","namespaced":true,"kind":"Deployment","verbs":["create","get","list"],"shortNames":["deploy"]},{"name":"replicasets","singularName":"replicaset","namespaced":true,"kind":"ReplicaSet","verbs":["create","get","list"],"shortNames":["rs"]}]}`
)

func APIHandler(c *gin.Context) {
	c.Data(http.StatusOK, "application/json", []byte(apiJSON))
}

func APIsHandler(c *gin.Context) {
	c.Data(http.StatusOK, "application/json", []byte(apisJSON))
}

func APIV1Handler(c *gin.Context) {
	c.Data(http.StatusOK, "application/json", []byte(apiV1JSON))
}

func AppsV1Handler(c *gin.Context) {
	c.Data(http.StatusOK, "application/json", []byte(appsV1JSON))
}
