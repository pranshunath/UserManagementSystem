package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"usermanagementsystem/internal/response"
	v1 "usermanagementsystem/proto"
)

// DepartmentHandler handles HTTP requests by delegating to gRPC DepartmentService.
type DepartmentHandler struct {
	grpcClient v1.DepartmentServiceClient
}

// NewDepartmentHandler creates a new DepartmentHandler instance.
func NewDepartmentHandler(grpcClient v1.DepartmentServiceClient) *DepartmentHandler {
	return &DepartmentHandler{grpcClient: grpcClient}
}

func (h *DepartmentHandler) CreateDepartment(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}

	fmt.Printf("[HTTP] POST /api/v1/departments -> Calling gRPC DepartmentService.CreateDepartment\n")
	res, err := h.grpcClient.CreateDepartment(c.Request.Context(), &v1.CreateDepartmentRequest{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": res.Department})
}

func (h *DepartmentHandler) GetDepartment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "ID must be integer")
		return
	}

	res, err := h.grpcClient.GetDepartment(c.Request.Context(), &v1.GetDepartmentRequest{Id: int32(id)})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": res.Department})
}

func (h *DepartmentHandler) ListDepartments(c *gin.Context) {
	res, err := h.grpcClient.ListDepartments(c.Request.Context(), &v1.ListDepartmentsRequest{})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": res.Departments, "count": len(res.Departments)})
}

func (h *DepartmentHandler) UpdateDepartment(c *gin.Context) {
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

	res, err := h.grpcClient.UpdateDepartment(c.Request.Context(), &v1.UpdateDepartmentRequest{
		Id:          int32(id),
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": res.Department})
}

func (h *DepartmentHandler) DeleteDepartment(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "ID must be integer")
		return
	}

	res, err := h.grpcClient.DeleteDepartment(c.Request.Context(), &v1.DeleteDepartmentRequest{Id: int32(id)})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "message": res.Message})
}

