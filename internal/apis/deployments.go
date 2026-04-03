package apis

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/apimachinery/pkg/util/validation"
	"mockernetes/internal/controllers"
	"mockernetes/internal/resources" // custom structs for mock control (no appsv1)
	"mockernetes/internal/storage"
)

// buildDeploymentList wraps store items into K8s list (similar to pods/ns; uses custom resources.Deployment structs).
func buildDeploymentList(items []interface{}) string {
	list := map[string]interface{}{
		"kind":       "DeploymentList",
		"apiVersion": "apps/v1",
		"metadata":   map[string]string{"resourceVersion": "1"},
		"items":      items,
	}
	b, _ := json.Marshal(list)
	return string(b)
}

func ListDeployments(c *gin.Context) {
	// ignore :namespace if present (mock simplification like pods)
	items := storage.DefaultStore.ListDeployments()
	c.Data(http.StatusOK, "application/json", []byte(buildDeploymentList(items)))
}

// CreateDeployment parses POST to custom resources.Deployment struct (for mock control, no appsv1/scheme).
// Validates, stores if not exists.
func CreateDeployment(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	// Unmarshal to custom struct (enforces mock shape)
	var deploy resources.Deployment
	if err := json.Unmarshal(body, &deploy); err != nil {
		WriteError(c, http.StatusBadRequest, err.Error())
		return
	}
	if deploy.Kind == "" {
		WriteError(c, http.StatusBadRequest, "invalid deployment")
		return
	}
	// basic name validation (DNS label like pods)
	if errs := validation.IsDNS1123Label(deploy.GetName()); len(errs) != 0 {
		WriteError(c, http.StatusBadRequest, errs[0])
		return
	}
	// store; uses KubeObject impl from custom struct
	if err := storage.DefaultStore.CreateDeployment(deploy); err != nil {
		WriteError(c, http.StatusConflict, err.Error())
		return
	}

	// Trigger deployment controller for immediate reconciliation
	if controllers.DefaultDeploymentController != nil {
		if err := controllers.DefaultDeploymentController.OnDeploymentCreated(deploy); err != nil {
			// Log error but don't fail the creation - the deployment is already stored
			_ = err
		}
	}

	// Return the deployment with status from storage
	storedDeploy, err := storage.DefaultStore.GetDeployment(deploy.GetName())
	if err != nil {
		// Fallback to returning the original deployment if get fails
		c.JSON(http.StatusCreated, deploy)
		return
	}
	c.JSON(http.StatusCreated, storedDeploy)
}
