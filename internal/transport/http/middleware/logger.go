package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/leyl1ne/UserService/internal/logger"
)

func LoggerMiddleware(log logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		mLog := log.With(
			logger.Field{Key: "component", Value: "middleware/logger"},
		)

		reqID := uuid.NewString()
		SetRequestID(c, reqID)

		reqLogger := mLog.With(
			logger.Field{Key: "request_id", Value: reqID},
			logger.Field{Key: "method", Value: c.Request.Method},
			logger.Field{Key: "path", Value: c.FullPath()},
		)

		start := time.Now()

		c.Next()

		defer func() {
			reqLogger.Info("request completed",
				logger.Field{Key: "status", Value: c.Writer.Status()},
				logger.Field{Key: "latency", Value: time.Since(start).String()},
			)
		}()

	}
}
