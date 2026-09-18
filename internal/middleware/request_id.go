package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

const HeaderRequestID = "X-Request-ID"
const ContextKeyRequestID = "request_id"

// RequestID middleware generates or preserves a unique Request ID for every incoming HTTP request.
// It sets the header in the response, stores it in the Gin context, and injects it into
// the outgoing gRPC metadata context for cross-service distributed tracing.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		reqID := c.GetHeader(HeaderRequestID)
		if reqID == "" {
			reqID = uuid.NewString()
		}

		// 1. Set response header so clients can quote this ID in bug reports or support tickets
		c.Header(HeaderRequestID, reqID)

		// 2. Set in Gin context for handlers, loggers, and error formatters
		c.Set(ContextKeyRequestID, reqID)

		// 3. Inject into gRPC outgoing context for distributed tracing across microservices
		outgoingCtx := metadata.AppendToOutgoingContext(c.Request.Context(), "x-request-id", reqID)
		c.Request = c.Request.WithContext(outgoingCtx)

		c.Next()
	}
}
