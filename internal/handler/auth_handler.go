package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"usermanagementsystem/internal/response"
	v1 "usermanagementsystem/proto"
)

// AuthHandler manages authentication endpoints by delegating to gRPC.
type AuthHandler struct {
	grpcClient v1.UserServiceClient
}

// NewAuthHandler creates a new AuthHandler instance.
func NewAuthHandler(grpcClient v1.UserServiceClient) *AuthHandler {
	return &AuthHandler{grpcClient: grpcClient}
}

// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Name         string  `json:"name" binding:"required,min=2,max=100"`
		Email        string  `json:"email" binding:"required,email"`
		Password     string  `json:"password" binding:"required,min=8"`
		DepartmentID *int32  `json:"department_id"`
		RoleIDs      []int32 `json:"role_ids"`
		Status       string  `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}

	// New registrations are active by default.
	status := req.Status
	if status == "" {
		status = "active"
	}

	fmt.Printf("[HTTP] POST /api/v1/auth/register -> Registering %s via gRPC\n", req.Email)

	res, err := h.grpcClient.RegisterUser(
		c.Request.Context(),
		&v1.CreateUserRequest{
			Name:         req.Name,
			Email:        req.Email,
			Password:     req.Password,
			DepartmentId: req.DepartmentID,
			RoleIds:      req.RoleIDs,
			Status:       status,
		},
	)
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	fmt.Printf("[HTTP] 201 Created -> User registered successfully with ID: %d\n", res.User.Id)

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"data": gin.H{
			"user": res.User,
		},
		"message": "User registered successfully",
	})
}

// Login handles POST /api/v1/auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_FAILED", err.Error())
		return
	}

	fmt.Printf("[HTTP] POST /api/v1/auth/login -> Authenticating %s via gRPC\n", req.Email)

	res, err := h.grpcClient.Login(c.Request.Context(), &v1.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		HandleGRPCError(c, err)
		return
	}

	fmt.Printf("[HTTP] 200 OK -> Authentication successful for user ID: %d\n", res.User.Id)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"token": res.Token,
			"user":  res.User,
			"roles": res.Roles,
		},
	})
}
