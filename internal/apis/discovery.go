package apis

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Discovery responses (hardcoded valid K8s shapes)
const (
	apiJSON = `{"kind":"APIVersions","versions":["v1"]}`
	apisJSON = `{"kind":"APIGroupList","groups":[{"name":"apps","versions":[{"groupVersion":"apps/v1","version":"v1"}],"preferredVersion":{"groupVersion":"apps/v1","version":"v1"}}]}`
	// namespaces with canonical form + shortNames["ns"] for kubectl get ns; plus common resources
	apiV1JSON = `{"kind":"APIResourceList","groupVersion":"v1","resources":[{"name":"namespaces","singularName":"namespace","namespaced":false,"kind":"Namespace","verbs":["create","delete","get","list","patch","update","watch"],"shortNames":["ns"],"categories":["all"]},{"name":"pods","singularName":"pod","namespaced":true,"kind":"Pod","verbs":["get","list"],"shortNames":["po"]},{"name":"configmaps","singularName":"configmap","namespaced":true,"kind":"ConfigMap","verbs":["get","list"],"shortNames":["cm"]}]}`

	// minimal for /apis/apps/v1 (to avoid memcache errors on discovery)
	appsV1JSON = `{"kind":"APIResourceList","groupVersion":"apps/v1","resources":[{"name":"deployments","namespaced":true,"kind":"Deployment","verbs":["get","list"]}]}` 
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