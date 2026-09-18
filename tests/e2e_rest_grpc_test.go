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

func TestEndToEndRESTToGRPC(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Clear test tables
	_, _ = db.ExecContext(ctx, "DELETE FROM user_roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM users")
	_, _ = db.ExecContext(ctx, "DELETE FROM roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM departments")

	// 2. Initialize Repositories and Services
	deptRepo := repository.NewDepartmentRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)

	deptService := service.NewDepartmentService(deptRepo)
	roleService := service.NewRoleService(roleRepo)
	userService := service.NewUserService(userRepo, deptRepo, roleRepo, nil)

	// 3. Start live gRPC Server
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

	// 4. Initialize gRPC Client in the API Gateway
	grpcClient, err := grpc.NewClient(lis.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial gRPC server: %v", err)
	}
	defer grpcClient.Close()

	// 5. Initialize Gin Router with HTTP Handlers
	router := gin.New()
	userHandler := handler.NewUserHandler(grpcClient.User)
	deptHandler := handler.NewDepartmentHandler(grpcClient.Department)

	v1Group := router.Group("/api/v1")
	{
		v1Group.POST("/departments", deptHandler.CreateDepartment)
		v1Group.POST("/users", userHandler.CreateUser)
		v1Group.GET("/users/:id", userHandler.GetUser)
		v1Group.GET("/users", userHandler.ListUsers)
		v1Group.PUT("/users/:id", userHandler.UpdateUser)
		v1Group.DELETE("/users/:id", userHandler.DeleteUser)
	}

	// 6. Test HTTP POST /api/v1/departments -> gRPC -> Service -> DB
	deptBody := bytes.NewBufferString(`{"name":"Security","description":"Cybersecurity operations"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/api/v1/departments", deptBody)
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for department, got %d: %s", w.Code, w.Body.String())
	}

	var deptResponse struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &deptResponse)
	deptID := deptResponse.Data.ID

	// 7. Test HTTP POST /api/v1/users -> gRPC -> Service -> DB
	userPayload := fmt.Sprintf(`{"name":"Marcus","email":"marcus@cyber.org","password":"supersecret123","department_id":%d}`, deptID)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(userPayload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for user, got %d: %s", w.Code, w.Body.String())
	}

	var userResponse struct {
		Data struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &userResponse)
	userID := userResponse.Data.ID
	if userID == 0 || userResponse.Data.Email != "marcus@cyber.org" {
		t.Errorf("Unexpected user payload: %s", w.Body.String())
	}

	// 8. Test HTTP GET /api/v1/users/:id -> gRPC -> Service -> DB
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%d", userID), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d: %s", w.Code, w.Body.String())
	}

	// 9. Test HTTP GET /api/v1/users (List with search)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, "/api/v1/users?search=marcus", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for list, got %d: %s", w.Code, w.Body.String())
	}

	// 10. Test HTTP DELETE /api/v1/users/:id
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/users/%d", userID), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK for delete, got %d: %s", w.Code, w.Body.String())
	}

	// 11. Test HTTP GET on deleted user returns 404 Not Found (gRPC codes.NotFound translated to HTTP 404)
	w = httptest.NewRecorder()
	req, _ = http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users/%d", userID), nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found for deleted user, got %d: %s", w.Code, w.Body.String())
	}

	t.Log(" Complete End-to-End [HTTP -> Gin -> gRPC Client -> gRPC Server -> Service -> Repo -> DB] verified!")
}
