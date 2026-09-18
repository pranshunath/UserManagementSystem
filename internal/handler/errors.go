package handler

import (
	"github.com/gin-gonic/gin"

	"usermanagementsystem/internal/response"
)

// HandleGRPCError converts a gRPC status error into a standardized HTTP error response.
func HandleGRPCError(c *gin.Context, err error) {
	response.HandleGRPCError(c, err)
}

