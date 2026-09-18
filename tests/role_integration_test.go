package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	googlegrpc "google.golang.org/grpc"

	"usermanagementsystem/internal/auth"
	"usermanagementsystem/internal/database"
	"usermanagementsystem/internal/grpc"
	"usermanagementsystem/internal/handler"
	"usermanagementsystem/internal/repository"
	"usermanagementsystem/internal/service"
	v1 "usermanagementsystem/proto"
)

func TestRoleManagementAndManyToManyAssignment(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 1. Connect to PostgreSQL
	cfg := database.Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "postgrespassword",
		DBName:   "usermanagement",
		SSLMode:  "disable",
	}
	db, err := database.NewPostgresDB(cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Clear test tables
	_, _ = db.ExecContext(ctx, "DELETE FROM user_roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM users")
	_, _ = db.ExecContext(ctx, "DELETE FROM roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM departments")

	// 2. Repositories and Services
	deptRepo := repository.NewDepartmentRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)

	deptService := service.NewDepartmentService(deptRepo)
	roleService := service.NewRoleService(roleRepo)
	userService := service.NewUserService(userRepo, deptRepo, roleRepo, nil)

	// 3. gRPC Server
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to bind TCP listener: %v", err)
	}
	tokenManager := auth.NewTokenManager("test-jwt-secret", 24*time.Hour)
	authService := service.NewAuthService(userRepo, roleRepo, tokenManager)

	grpcServer := googlegrpc.NewServer()
	srv := grpc.NewServer(userService, deptService, roleService, authService)
	v1.RegisterUserServiceServer(grpcServer, srv)
	v1.RegisterDepartmentServiceServer(grpcServer, srv)
	v1.RegisterRoleServiceServer(grpcServer, srv)

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.GracefulStop()

	// 4. gRPC Client and HTTP Gateway
	grpcClient, err := grpc.NewClient(lis.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial gRPC: %v", err)
	}
	defer grpcClient.Close()

	router := gin.New()
	roleHandler := handler.NewRoleHandler(grpcClient.Role)
	userHandler := handler.NewUserHandler(grpcClient.User)

	v1Group := router.Group("/api/v1")
	{
		v1Group.POST("/roles", roleHandler.CreateRole)
		v1Group.GET("/roles", roleHandler.ListRoles)
		v1Group.GET("/roles/:id", roleHandler.GetRole)
		v1Group.PUT("/roles/:id", roleHandler.UpdateRole)
		v1Group.DELETE("/roles/:id", roleHandler.DeleteRole)

		v1Group.POST("/users", userHandler.CreateUser)
		v1Group.GET("/users", userHandler.ListUsers)
		v1Group.GET("/users/:id", userHandler.GetUser)
		v1Group.PUT("/users/:id", userHandler.UpdateUser)
	}

	// -------------------------------------------------------------
	// TEST 1: Create 3 Roles: Admin, Developer, Auditor
	// -------------------------------------------------------------
	createRole := func(name, desc string) int {
		payload := fmt.Sprintf(`{"name":"%s","description":"%s"}`, name, desc)
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodPost, "/api/v1/roles", bytes.NewBufferString(payload))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("Failed to create role %s: %s", name, w.Body.String())
		}
		var resp struct {
			Data struct {
				ID int `json:"id"`
			} `json:"data"`
		}
		_ = json.Unmarshal(w.Body.Bytes(), &resp)
		return resp.Data.ID
	}

	adminRoleID := createRole("Admin", "Full administrative access")
	devRoleID := createRole("Developer", "Code commit and deployment")
	auditorRoleID := createRole("Auditor", "Read-only compliance audit")

	// -------------------------------------------------------------
	// TEST 2: Create User "Pranshu" with Multiple Roles (Admin + Developer)
	// -------------------------------------------------------------
	userPayload := fmt.Sprintf(`{
		"name": "Pranshu",
		"email": "pranshu@company.com",
		"password": "securepassword123",
		"role_ids": [%d, %d]
	}`, adminRoleID, devRoleID)

	wUser := httptest.NewRecorder()
	reqUser, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(userPayload))
	reqUser.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wUser, reqUser)
	if wUser.Code != http.StatusCreated {
		t.Fatalf("Failed to create user: %s", wUser.Body.String())
	}

	var userResp struct {
		Data struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Roles []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"roles"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wUser.Body.Bytes(), &userResp)
	pranshuID := userResp.Data.ID

	if len(userResp.Data.Roles) != 2 {
		t.Fatalf("Expected 2 roles assigned to Pranshu, got: %+v", userResp.Data.Roles)
	}

	// -------------------------------------------------------------
	// TEST 3: Verify Many-to-Many Junction Table `user_roles` in PostgreSQL
	// -------------------------------------------------------------
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_roles WHERE user_id = $1", pranshuID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query user_roles: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 rows in user_roles table for user %d, got %d", pranshuID, count)
	}

	// -------------------------------------------------------------
	// TEST 4: Filter Users by Role (role_id)
	// -------------------------------------------------------------
	// Query users with role = Developer
	wFilterDev := httptest.NewRecorder()
	reqFilterDev, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users?role_id=%d", devRoleID), nil)
	router.ServeHTTP(wFilterDev, reqFilterDev)
	if wFilterDev.Code != http.StatusOK {
		t.Fatalf("Failed to filter by Developer role: %s", wFilterDev.Body.String())
	}

	var devListResp struct {
		Pagination struct {
			TotalCount int `json:"total_count"`
		} `json:"pagination"`
	}
	_ = json.Unmarshal(wFilterDev.Body.Bytes(), &devListResp)
	if devListResp.Pagination.TotalCount != 1 {
		t.Errorf("Expected 1 user with Developer role, got %d", devListResp.Pagination.TotalCount)
	}

	// Query users with role = Auditor (Pranshu is not an Auditor)
	wFilterAud := httptest.NewRecorder()
	reqFilterAud, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users?role_id=%d", auditorRoleID), nil)
	router.ServeHTTP(wFilterAud, reqFilterAud)
	var audListResp struct {
		Pagination struct {
			TotalCount int `json:"total_count"`
		} `json:"pagination"`
	}
	_ = json.Unmarshal(wFilterAud.Body.Bytes(), &audListResp)
	if audListResp.Pagination.TotalCount != 0 {
		t.Errorf("Expected 0 users with Auditor role, got %d", audListResp.Pagination.TotalCount)
	}

	// -------------------------------------------------------------
	// TEST 5: Role Synchronization on User Update (Switch Admin -> Auditor)
	// -------------------------------------------------------------
	updatePayload := fmt.Sprintf(`{
		"role_ids": [%d, %d]
	}`, devRoleID, auditorRoleID) // Remove Admin, keep Developer, add Auditor

	wUpdate := httptest.NewRecorder()
	reqUpdate, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/users/%d", pranshuID), bytes.NewBufferString(updatePayload))
	reqUpdate.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wUpdate, reqUpdate)
	if wUpdate.Code != http.StatusOK {
		t.Fatalf("Failed to update user roles: %s", wUpdate.Body.String())
	}

	var updatedUserResp struct {
		Data struct {
			Roles []struct {
				ID   int    `json:"id"`
				Name string `json:"name"`
			} `json:"roles"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wUpdate.Body.Bytes(), &updatedUserResp)
	roleNames := make([]string, 0)
	for _, r := range updatedUserResp.Data.Roles {
		roleNames = append(roleNames, r.Name)
	}

	// Verify Pranshu now has Developer & Auditor, and Admin was removed
	if len(roleNames) != 2 {
		t.Errorf("Expected 2 updated roles, got: %+v", roleNames)
	}

	// -------------------------------------------------------------
	// TEST 6: Cascade Deletion on Roles Table
	// -------------------------------------------------------------
	// If we delete the "Auditor" role, PostgreSQL's ON DELETE CASCADE
	// should cleanly remove the row from `user_roles`
	wDelRole := httptest.NewRecorder()
	reqDelRole, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/roles/%d", auditorRoleID), nil)
	router.ServeHTTP(wDelRole, reqDelRole)
	if wDelRole.Code != http.StatusOK {
		t.Fatalf("Failed to delete Auditor role: %s", wDelRole.Body.String())
	}

	// Verify Pranshu now only has 1 remaining role in user_roles (Developer)
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM user_roles WHERE user_id = $1", pranshuID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to query user_roles: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 role remaining after Auditor role deletion, got %d", count)
	}

	t.Log(" Many-to-many role management, assignment, synchronization, and cascading verified successfully!")
}
