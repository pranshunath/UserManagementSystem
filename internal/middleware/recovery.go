package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"usermanagementsystem/internal/response"
)

// Recovery returns a middleware that recovers from any panics within handlers or downstream middlewares.
// It logs the full stack trace tagged with the Request ID and returns a sanitized, structured 500 error.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				reqID := c.GetString(ContextKeyRequestID)
				stack := debug.Stack()

				fmt.Printf("[PANIC RECOVERED] [req_id=%s] panic: %v\n%s\n", reqID, r, string(stack))

				response.AbortError(
					c,
					http.StatusInternalServerError,
					"INTERNAL_SERVER_ERROR",
					"An unexpected internal error occurred. Please contact support with the request ID.",
				)
			}
		}()

		c.Next()
	}
}
