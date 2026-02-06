package server

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"mockernetes/internal/apis"
)

func NewServer() {
	r := gin.Default()

	// wire routes including discovery
	wireRoutes(r)

	// run HTTPS on localhost:8443 with certs
	if err := r.RunTLS("127.0.0.1:8443", "certs/server.crt", "certs/server.key"); err != nil {
		panic(err)
	}
}

func wireRoutes(r *gin.Engine) {
	// discovery endpoints (hardcoded; includes /apis/apps/v1 + ns canonical/shortNames)
	r.GET("/api", apis.APIHandler)
	r.GET("/apis", apis.APIsHandler)
	r.GET("/api/v1", apis.APIV1Handler)
	r.GET("/apis/apps/v1", apis.AppsV1Handler)

	// health/ready + core resources (ns/pods/cms with in-mem storage)
	// namespaced routes for pods/cms (mock ignores :namespace param; kubectl uses e.g. /namespaces/default/...)
	r.GET("/healthz", healthzHandler)
	r.GET("/readyz", readyzHandler)
	r.GET("/api/v1/namespaces", apis.ListNamespaces)
	r.POST("/api/v1/namespaces", apis.CreateNamespace)
	// cluster + namespaced for pods/cms
	r.GET("/api/v1/pods", apis.ListPods)
	r.POST("/api/v1/pods", apis.CreatePod)
	r.GET("/api/v1/namespaces/:namespace/pods", apis.ListPods)
	r.POST("/api/v1/namespaces/:namespace/pods", apis.CreatePod)
	r.GET("/api/v1/configmaps", apis.ListConfigMaps)
	r.POST("/api/v1/configmaps", apis.CreateConfigMap)
	r.GET("/api/v1/namespaces/:namespace/configmaps", apis.ListConfigMaps)
	r.POST("/api/v1/namespaces/:namespace/configmaps", apis.CreateConfigMap)
}

func healthzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func readyzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}