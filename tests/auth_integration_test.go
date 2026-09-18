package tests

import (
	"bytes"
	"context"
	"encoding/json"
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
	"usermanagementsystem/internal/middleware"
	"usermanagementsystem/internal/repository"
	"usermanagementsystem/internal/service"
	v1 "usermanagementsystem/proto"
)

func TestAuthenticationAndRoleBasedAccessControl(t *testing.T) {
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

	jwtSecret := "test-secret-key-12345"
	tokenManager := auth.NewTokenManager(jwtSecret, 2*time.Hour)

	deptService := service.NewDepartmentService(deptRepo)
	roleService := service.NewRoleService(roleRepo)
	userService := service.NewUserService(userRepo, deptRepo, roleRepo, nil)
	authService := service.NewAuthService(userRepo, roleRepo, tokenManager)

	// 3. gRPC Server
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to bind TCP listener: %v", err)
	}
	grpcServer := googlegrpc.NewServer()
	srv := grpc.NewServer(userService, deptService, roleService, authService)
	v1.RegisterUserServiceServer(grpcServer, srv)
	v1.RegisterDepartmentServiceServer(grpcServer, srv)
	v1.RegisterRoleServiceServer(grpcServer, srv)

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.GracefulStop()

	// 4. gRPC Client and REST Gateway
	grpcClient, err := grpc.NewClient(lis.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial gRPC: %v", err)
	}
	defer grpcClient.Close()

	router := gin.New()
	authHandler := handler.NewAuthHandler(grpcClient.User)
	userHandler := handler.NewUserHandler(grpcClient.User)
	deptHandler := handler.NewDepartmentHandler(grpcClient.Department)

	v1Group := router.Group("/api/v1")
	{
		// Public
		v1Group.POST("/auth/login", authHandler.Login)

		// Protected
		protected := v1Group.Group("")
		protected.Use(middleware.AuthMiddleware(tokenManager))
		{
			// Users
			protected.GET("/users", userHandler.ListUsers)
			protected.POST("/users", middleware.RequireRole("Admin", "Manager"), userHandler.CreateUser)
			protected.DELETE("/users/:id", middleware.RequireRole("Admin"), userHandler.DeleteUser)

			// Departments (Admin only for modifications)
			protected.GET("/departments", deptHandler.ListDepartments)
			protected.POST("/departments", middleware.RequireRole("Admin"), deptHandler.CreateDepartment)
		}
	}

	// -------------------------------------------------------------
	// SETUP: Create Roles (Admin, Employee) and Users directly via Service
	// -------------------------------------------------------------
	adminRole, _ := roleService.CreateRole(ctx, "Admin", "Administrator")
	employeeRole, _ := roleService.CreateRole(ctx, "Employee", "Standard Employee")

	_, err = userService.CreateUser(ctx, service.CreateUserInput{
		Name:     "Admin User",
		Email:    "admin@system.io",
		Password: "adminpassword123",
		RoleIDs:  []int{adminRole.ID},
		Status:   "active",
	})
	if err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	_, err = userService.CreateUser(ctx, service.CreateUserInput{
		Name:     "Staff Member",
		Email:    "staff@system.io",
		Password: "staffpassword123",
		RoleIDs:  []int{employeeRole.ID},
		Status:   "active",
	})
	if err != nil {
		t.Fatalf("Failed to create staff user: %v", err)
	}

	// -------------------------------------------------------------
	// TEST 1: Login with Wrong Password -> 401 Unauthorized
	// -------------------------------------------------------------
	wrongLoginBody := bytes.NewBufferString(`{"email":"admin@system.io","password":"wrongpassword"}`)
	wWrong := httptest.NewRecorder()
	reqWrong, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", wrongLoginBody)
	reqWrong.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wWrong, reqWrong)

	if wWrong.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 Unauthorized for wrong password, got %d: %s", wWrong.Code, wWrong.Body.String())
	}

	// -------------------------------------------------------------
	// TEST 2: Successful Admin Login -> 200 OK with signed JWT
	// -------------------------------------------------------------
	adminLoginBody := bytes.NewBufferString(`{"email":"admin@system.io","password":"adminpassword123"}`)
	wAdmin := httptest.NewRecorder()
	reqAdmin, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", adminLoginBody)
	reqAdmin.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wAdmin, reqAdmin)

	if wAdmin.Code != http.StatusOK {
		t.Fatalf("Admin login failed: %s", wAdmin.Body.String())
	}

	var adminLoginResp struct {
		Data struct {
			Token string   `json:"token"`
			Roles []string `json:"roles"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wAdmin.Body.Bytes(), &adminLoginResp)
	adminToken := adminLoginResp.Data.Token
	if adminToken == "" {
		t.Fatalf("Expected JWT token string, got empty")
	}

	// -------------------------------------------------------------
	// TEST 3: Successful Staff Login
	// -------------------------------------------------------------
	staffLoginBody := bytes.NewBufferString(`{"email":"staff@system.io","password":"staffpassword123"}`)
	wStaff := httptest.NewRecorder()
	reqStaff, _ := http.NewRequest(http.MethodPost, "/api/v1/auth/login", staffLoginBody)
	reqStaff.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wStaff, reqStaff)

	var staffLoginResp struct {
		Data struct {
			Token string `json:"token"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wStaff.Body.Bytes(), &staffLoginResp)
	staffToken := staffLoginResp.Data.Token

	// -------------------------------------------------------------
	// TEST 4: Access Protected Route Without Token -> 401 Unauthorized
	// -------------------------------------------------------------
	wNoToken := httptest.NewRecorder()
	reqNoToken, _ := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
	router.ServeHTTP(wNoToken, reqNoToken)

	if wNoToken.Code != http.StatusUnauthorized {
		t.Fatalf("Expected 401 without token, got %d: %s", wNoToken.Code, wNoToken.Body.String())
	}

	// -------------------------------------------------------------
	// TEST 5: Access Protected Route With Valid Token -> 200 OK
	// -------------------------------------------------------------
	wWithToken := httptest.NewRecorder()
	reqWithToken, _ := http.NewRequest(http.MethodGet, "/api/v1/users", nil)
	reqWithToken.Header.Set("Authorization", "Bearer "+staffToken)
	router.ServeHTTP(wWithToken, reqWithToken)

	if wWithToken.Code != http.StatusOK {
		t.Fatalf("Expected 200 with staff token, got %d: %s", wWithToken.Code, wWithToken.Body.String())
	}

	// -------------------------------------------------------------
	// TEST 6: RBAC Forbidden Test (Staff tries to create Department -> 403)
	// -------------------------------------------------------------
	deptBody := bytes.NewBufferString(`{"name":"Forbidden Dept","description":"Should fail"}`)
	wForbidden := httptest.NewRecorder()
	reqForbidden, _ := http.NewRequest(http.MethodPost, "/api/v1/departments", deptBody)
	reqForbidden.Header.Set("Content-Type", "application/json")
	reqForbidden.Header.Set("Authorization", "Bearer "+staffToken) // Staff token (Lacks Admin)
	router.ServeHTTP(wForbidden, reqForbidden)

	if wForbidden.Code != http.StatusForbidden {
		t.Fatalf("Expected 403 Forbidden for staff creating department, got %d: %s", wForbidden.Code, wForbidden.Body.String())
	}

	// -------------------------------------------------------------
	// TEST 7: RBAC Allowed Test (Admin creates Department -> 201 Created)
	// -------------------------------------------------------------
	wAllowed := httptest.NewRecorder()
	reqAllowed, _ := http.NewRequest(http.MethodPost, "/api/v1/departments", bytes.NewBufferString(`{"name":"Operations","description":"Admin allowed"}`))
	reqAllowed.Header.Set("Content-Type", "application/json")
	reqAllowed.Header.Set("Authorization", "Bearer "+adminToken) // Admin token
	router.ServeHTTP(wAllowed, reqAllowed)

	if wAllowed.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for admin creating department, got %d: %s", wAllowed.Code, wAllowed.Body.String())
	}

	t.Log(" Authentication, JWT signing, AuthMiddleware, and RBAC Role enforcement verified successfully!")
}
