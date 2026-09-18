package middleware

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// StructuredLogger returns a Gin middleware that logs HTTP requests in a production-ready,
// structured format with timestamp, status, latency, client IP, method, path, and request ID.
func StructuredLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		method := c.Request.Method
		reqID := c.GetString("request_id")

		fullPath := path
		if raw != "" {
			fullPath = path + "?" + raw
		}

		userTag := ""
		if email := c.GetString("email"); email != "" {
			userTag = fmt.Sprintf(" user=%s", email)
		}

		// Categorize log level by HTTP status
		level := "INFO"
		if statusCode >= 500 {
			level = "ERROR"
		} else if statusCode >= 400 {
			level = "WARN"
		}

		fmt.Printf("[%s] [HTTP] %3d | %10v | %15s | %-7s %s | req_id=%s%s\n",
			level,
			statusCode,
			latency,
			clientIP,
			method,
			fullPath,
			reqID,
			userTag,
		)

		// Print any internal Gin errors if present
		for _, e := range c.Errors {
			fmt.Printf("       ↳ [GIN ERROR] %v\n", e.Err)
		}
	}
}
