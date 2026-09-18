package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"usermanagementsystem/internal/response"
	v1 "usermanagementsystem/proto"
)

// RoleHandler handles HTTP requests by delegating to gRPC RoleService.
type RoleHandler struct {
	grpcClient v1.RoleServiceClient
}

// NewRoleHandler creates a new RoleHandler instance.
func NewRoleHandler(grpcClient v1.RoleServiceClient) *RoleHandler {
	return &RoleHandler{grpcClient: grpcClient}
}

func (h *RoleHandler) CreateRole(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}

	fmt.Printf("[HTTP] POST /api/v1/roles -> Calling gRPC RoleService.CreateRole\n")
	res, err := h.grpcClient.CreateRole(c.Request.Context(), &v1.CreateRoleRequest{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": res.Role})
}

func (h *RoleHandler) GetRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "ID must be integer")
		return
	}

	res, err := h.grpcClient.GetRole(c.Request.Context(), &v1.GetRoleRequest{Id: int32(id)})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": res.Role})
}

func (h *RoleHandler) ListRoles(c *gin.Context) {
	res, err := h.grpcClient.ListRoles(c.Request.Context(), &v1.ListRolesRequest{})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": res.Roles, "count": len(res.Roles)})
}

func (h *RoleHandler) UpdateRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "ID must be integer")
		return
	}

	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}

	res, err := h.grpcClient.UpdateRole(c.Request.Context(), &v1.UpdateRoleRequest{
		Id:          int32(id),
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": res.Role})
}

func (h *RoleHandler) DeleteRole(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "ID must be integer")
		return
	}

	res, err := h.grpcClient.DeleteRole(c.Request.Context(), &v1.DeleteRoleRequest{Id: int32(id)})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": res.Message})
}

