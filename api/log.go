package api

import (
	"time"

	"github.com/google/uuid"
	"github.com/kod2ulz/gostart/logr"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

const (
	requestIdHeader1 = "Request-Id"
	requestIdHeader2 = "X-Request-Id"
)

func JSONLogMiddleware(log *logr.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {

		start := time.Now()

		// Write request ID to context
		var requestId string
		if requestId = c.GetHeader(requestIdHeader1); requestId == "" {
			if requestId = c.GetHeader(requestIdHeader2); requestId == "" {
				requestId = uuid.New().String()
			}
		}
		c.Set(requestIdHeader1, requestId)
		c.Writer.Header().Set(requestIdHeader1, requestId)

		c.Next()

		fields := logrus.Fields{
			"client_ip":  c.ClientIP(),
			"duration":   time.Since(start).Milliseconds(),
			"method":     c.Request.Method,
			"path":       c.Request.RequestURI,
			"status":     c.Writer.Status(),
			"referrer":   c.Request.Referer(),
			"request_id": c.Writer.Header().Get(requestIdHeader1),
			"size":       c.Writer.Size(),
			// "api_version": util.ApiVersion,
		}

		if user, ok := c.Get("auth.User"); ok {
			fields["user_id"] = user.(User).ID()
		}

		entry := log.WithFields(fields)
		if c.Writer.Status() >= 500 {
			entry.Error(c.Errors.String())
		} else if c.Writer.Status() >= 400 {
			entry.WithField("errors", c.Errors).Warn("")
		} else {
			entry.Info("")
		}
	}
}
