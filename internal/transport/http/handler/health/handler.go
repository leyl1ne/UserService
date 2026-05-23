package health

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	appVersing     string
	appName        string
	appEnviroument string
}

func NewHealthHandler(appVersion, appName, appEnvirounment string) *HealthHandler {
	return &HealthHandler{
		appName:        appName,
		appVersing:     appVersion,
		appEnviroument: appEnvirounment,
	}
}

func (h *HealthHandler) Health() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "ok",
			"service":     h.appName,
			"version":     h.appVersing,
			"environment": h.appEnviroument,
		})
	}
}
