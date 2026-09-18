package handler

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"usermanagementsystem/internal/response"
	v1 "usermanagementsystem/proto"
)

// UserHandler handles HTTP requests by delegating to the gRPC UserService client.
type UserHandler struct {
	grpcClient v1.UserServiceClient
}

// NewUserHandler instantiates a UserHandler with an injected gRPC client.
func NewUserHandler(grpcClient v1.UserServiceClient) *UserHandler {
	return &UserHandler{grpcClient: grpcClient}
}

// CreateUser handles POST /api/v1/users.
func (h *UserHandler) CreateUser(c *gin.Context) {
	fmt.Printf("[HTTP] POST %s -> Ingress Request Received\n", c.Request.URL.Path)

	var req struct {
		Name         string  `json:"name" binding:"required"`
		Email        string  `json:"email" binding:"required,email"`
		Password     string  `json:"password" binding:"required,min=8"`
		DepartmentID *int32  `json:"department_id"`
		RoleIDs      []int32 `json:"role_ids"`
		Status       string  `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Printf("[HTTP] 400 Bad Request: %v\n", err)
		response.Error(c, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}

	fmt.Println("[HTTP] Calling gRPC client -> UserService.CreateUser")
	grpcReq := &v1.CreateUserRequest{
		Name:         req.Name,
		Email:        req.Email,
		Password:     req.Password,
		DepartmentId: req.DepartmentID,
		RoleIds:      req.RoleIDs,
		Status:       req.Status,
	}

	// Delegate to the internal microservice via gRPC
	res, err := h.grpcClient.CreateUser(c.Request.Context(), grpcReq)
	if err != nil {
		fmt.Printf("[HTTP] gRPC error received: %v\n", err)
		HandleGRPCError(c, err)
		return
	}

	fmt.Printf("[HTTP] 201 Created -> Returning User ID: %d\n", res.User.Id)
	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data":    res.User,
	})
}

// GetUser handles GET /api/v1/users/:id.
func (h *UserHandler) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "User ID must be an integer")
		return
	}

	fmt.Printf("[HTTP] GET /api/v1/users/%d -> Calling gRPC client UserService.GetUser\n", id)
	res, err := h.grpcClient.GetUser(c.Request.Context(), &v1.GetUserRequest{Id: int32(id)})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	fmt.Printf("[HTTP] 200 OK -> User %d found\n", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    res.User,
	})
}

// ListUsers handles GET /api/v1/users with search, filtering, and pagination.
func (h *UserHandler) ListUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	search := c.Query("search")
	status := c.Query("status")
	sortBy := c.DefaultQuery("sort_by", "id")
	sortOrder := c.DefaultQuery("sort_order", "asc")

	var deptID *int32
	if dStr := c.Query("department_id"); dStr != "" {
		if d, err := strconv.Atoi(dStr); err == nil {
			id := int32(d)
			deptID = &id
		}
	}

	var roleID *int32
	if rStr := c.Query("role_id"); rStr != "" {
		if r, err := strconv.Atoi(rStr); err == nil {
			id := int32(r)
			roleID = &id
		}
	}

	fmt.Printf("[HTTP] GET /api/v1/users -> Calling gRPC client UserService.ListUsers (page=%d, limit=%d, sort=%s %s)\n", page, limit, sortBy, sortOrder)
	grpcReq := &v1.ListUsersRequest{
		Page:         int32(page),
		Limit:        int32(limit),
		Search:       search,
		DepartmentId: deptID,
		RoleId:       roleID,
		Status:       status,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
	}

	res, err := h.grpcClient.ListUsers(c.Request.Context(), grpcReq)
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	fmt.Printf("[HTTP] 200 OK -> Returned %d users (Total: %d)\n", len(res.Users), res.TotalCount)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    res.Users,
		"pagination": gin.H{
			"total_count": res.TotalCount,
			"page":        res.Page,
			"limit":       res.Limit,
			"total_pages": res.TotalPages,
		},
	})
}

// UpdateUser handles PUT /api/v1/users/:id.
func (h *UserHandler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "User ID must be an integer")
		return
	}

	var req struct {
		Name         string  `json:"name"`
		Email        string  `json:"email"`
		DepartmentID *int32  `json:"department_id"`
		RoleIDs      []int32 `json:"role_ids"`
		Status       string  `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}

	fmt.Printf("[HTTP] PUT /api/v1/users/%d -> Calling gRPC client UserService.UpdateUser\n", id)
	grpcReq := &v1.UpdateUserRequest{
		Id:           int32(id),
		Name:         req.Name,
		Email:        req.Email,
		DepartmentId: req.DepartmentID,
		RoleIds:      req.RoleIDs,
		Status:       req.Status,
	}

	res, err := h.grpcClient.UpdateUser(c.Request.Context(), grpcReq)
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	fmt.Printf("[HTTP] 200 OK -> User %d updated\n", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    res.User,
	})
}

// DeleteUser handles DELETE /api/v1/users/:id.
func (h *UserHandler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "INVALID_ID", "User ID must be an integer")
		return
	}

	fmt.Printf("[HTTP] DELETE /api/v1/users/%d -> Calling gRPC client UserService.DeleteUser\n", id)
	res, err := h.grpcClient.DeleteUser(c.Request.Context(), &v1.DeleteUserRequest{Id: int32(id)})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	fmt.Printf("[HTTP] 200 OK -> User %d deleted\n", id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": res.Message,
	})
}
