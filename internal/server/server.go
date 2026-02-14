package server

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"mockernetes/internal/apis"
	"mockernetes/internal/auth"
)

func NewServer() {
	r := gin.Default()

	// wire routes including discovery
	wireRoutes(r)

	// Configure full mTLS using auth package (Gin RunTLS only serves certs but does not verify client certs).
	// We use http.Server + tls.Config from auth.NewTLSConfig which:
	// - Sets ClientAuth: tls.RequireAndVerifyClientCert
	// - Uses CA pool to verify client cert chain
	// - Delegates further cert checks (via ExtractUser) to auth.go
	// This satisfies kubectl mTLS via the provided kubeconfig/client.crt
	tlsConfig, err := auth.NewTLSConfig("certs/server.crt", "certs/server.key", "certs/ca.crt")
	if err != nil {
		panic(fmt.Errorf("failed to setup mTLS config: %w", err))
	}

	// Custom http.Server required because Gin's RunTLS does not expose ClientCAs/ClientAuth options.
	// ListenAndServeTLS with empty cert/key paths uses the pre-loaded Certificates from TLSConfig.
	srv := &http.Server{
		Addr:      "127.0.0.1:8443",
		Handler:   r,
		TLSConfig: tlsConfig,
	}

	if err := srv.ListenAndServeTLS("", ""); err != nil {
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
	// cluster + namespaced for pods/cms (similarly for apps/v1 deploy/rs below)
	r.GET("/api/v1/pods", apis.ListPods)
	r.POST("/api/v1/pods", apis.CreatePod)
	r.GET("/api/v1/namespaces/:namespace/pods", apis.ListPods)
	r.POST("/api/v1/namespaces/:namespace/pods", apis.CreatePod)
	r.GET("/api/v1/configmaps", apis.ListConfigMaps)
	r.POST("/api/v1/configmaps", apis.CreateConfigMap)
	r.GET("/api/v1/namespaces/:namespace/configmaps", apis.ListConfigMaps)
	r.POST("/api/v1/namespaces/:namespace/configmaps", apis.CreateConfigMap)

	// apps/v1 resources (deployments + replicasets; cluster-scoped paths + namespaced like pods.
	// Note: /apis/apps/v1/... for group-version; mirrors pod handling for minimal mock.
	r.GET("/apis/apps/v1/deployments", apis.ListDeployments)
	r.POST("/apis/apps/v1/deployments", apis.CreateDeployment)
	r.GET("/apis/apps/v1/namespaces/:namespace/deployments", apis.ListDeployments)
	r.POST("/apis/apps/v1/namespaces/:namespace/deployments", apis.CreateDeployment)
	r.GET("/apis/apps/v1/replicasets", apis.ListReplicaSets)
	r.POST("/apis/apps/v1/replicasets", apis.CreateReplicaSet)
	r.GET("/apis/apps/v1/namespaces/:namespace/replicasets", apis.ListReplicaSets)
	r.POST("/apis/apps/v1/namespaces/:namespace/replicasets", apis.CreateReplicaSet)
}

func healthzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func readyzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
