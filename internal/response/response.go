package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Response defines the standard unified API payload structure.
type Response struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

// APIError represents the structured error details in an API response.
type APIError struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

// MetaPagination provides metadata about paginated result sets.
type MetaPagination struct {
	Total      int64 `json:"total"`
	Page       int32 `json:"page"`
	Limit      int32 `json:"limit"`
	TotalPages int32 `json:"total_pages"`
}

// getRequestID retrieves the Request ID from the Gin context or response headers.
func getRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if reqID := c.GetString("request_id"); reqID != "" {
		return reqID
	}
	return c.Writer.Header().Get("X-Request-ID")
}

// Success sends a standardized 2xx success response.
func Success(c *gin.Context, statusCode int, data interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Data:    data,
	})
}

// SuccessWithMeta sends a standardized success response containing metadata (e.g. pagination).
func SuccessWithMeta(c *gin.Context, statusCode int, data interface{}, meta interface{}) {
	c.JSON(statusCode, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Error sends a standardized error response with the request ID.
func Error(c *gin.Context, statusCode int, code string, message string, details ...interface{}) {
	var det interface{}
	if len(details) > 0 {
		det = details[0]
	}

	c.JSON(statusCode, Response{
		Success: false,
		Error: &APIError{
			Code:      code,
			Message:   message,
			Details:   det,
			RequestID: getRequestID(c),
		},
	})
}

// AbortError aborts the Gin middleware chain and sends a standardized error response.
func AbortError(c *gin.Context, statusCode int, code string, message string, details ...interface{}) {
	var det interface{}
	if len(details) > 0 {
		det = details[0]
	}

	c.AbortWithStatusJSON(statusCode, Response{
		Success: false,
		Error: &APIError{
			Code:      code,
			Message:   message,
			Details:   det,
			RequestID: getRequestID(c),
		},
	})
}

// HandleGRPCError converts a gRPC status error into a standardized HTTP error response.
func HandleGRPCError(c *gin.Context, err error) {
	st, ok := status.FromError(err)
	if !ok {
		Error(c, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	var httpStatus int
	var errorCode string

	switch st.Code() {
	case codes.NotFound:
		httpStatus = http.StatusNotFound
		errorCode = "NOT_FOUND"

	case codes.AlreadyExists:
		httpStatus = http.StatusConflict
		errorCode = "ALREADY_EXISTS"

	case codes.InvalidArgument:
		httpStatus = http.StatusBadRequest
		errorCode = "BAD_REQUEST"

	case codes.Unauthenticated:
		httpStatus = http.StatusUnauthorized
		errorCode = "UNAUTHORIZED"

	case codes.PermissionDenied:
		httpStatus = http.StatusForbidden
		errorCode = "FORBIDDEN"

	default:
		httpStatus = http.StatusInternalServerError
		errorCode = "INTERNAL_ERROR"
	}

	Error(c, httpStatus, errorCode, st.Message())
}
