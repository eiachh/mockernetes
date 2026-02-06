package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	// Health and ready endpoints
	r.GET("/healthz", healthzHandler)
	r.GET("/readyz", readyzHandler)

	// API routes for namespaces
	r.GET("/api/v1/namespaces", getNamespacesHandler)
	r.POST("/api/v1/namespaces", createNamespaceHandler)

	// Start the server
	r.Run(":8080") // TODO: make configurable
}

func healthzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func readyzHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func getNamespacesHandler(c *gin.Context) {
	// TODO: implement kubectl get ns logic
	c.JSON(http.StatusOK, gin.H{"items": []string{}}) // stub
}

func createNamespaceHandler(c *gin.Context) {
	// TODO: implement kubectl create ns logic
	c.JSON(http.StatusCreated, gin.H{"message": "created"}) // stub
}