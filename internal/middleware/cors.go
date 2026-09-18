package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORSOptions configures the CORS middleware.
type CORSOptions struct {
	AllowedOrigins []string
}

// CORS returns a middleware handling Cross-Origin Resource Sharing (CORS).
// Supports preflight OPTIONS requests, credentials, custom headers (including X-Request-ID),
// and allows seamless communication with the upcoming React/Vite frontend.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if origin == "" {
			origin = "*"
		}

		// According to the Fetch/CORS spec: if credentials are true, Allow-Origin cannot be "*"
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Accept, Cache-Control, X-Requested-With, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Max-Age", "43200") // 12 hours cached in browser preflight cache

		// Handle preflight browser OPTIONS requests immediately
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
