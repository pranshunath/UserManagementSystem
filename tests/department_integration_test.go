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

func TestDepartmentManagementAndFiltering(t *testing.T) {
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
	deptHandler := handler.NewDepartmentHandler(grpcClient.Department)
	userHandler := handler.NewUserHandler(grpcClient.User)

	v1Group := router.Group("/api/v1")
	{
		v1Group.POST("/departments", deptHandler.CreateDepartment)
		v1Group.GET("/departments", deptHandler.ListDepartments)
		v1Group.GET("/departments/:id", deptHandler.GetDepartment)
		v1Group.PUT("/departments/:id", deptHandler.UpdateDepartment)
		v1Group.DELETE("/departments/:id", deptHandler.DeleteDepartment)

		v1Group.POST("/users", userHandler.CreateUser)
		v1Group.GET("/users", userHandler.ListUsers)
		v1Group.GET("/users/:id", userHandler.GetUser)
	}

	// -------------------------------------------------------------
	// TEST 1: Create Two Departments (Engineering and Marketing)
	// -------------------------------------------------------------
	dept1Payload := `{"name":"Engineering","description":"Software and Systems"}`
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest(http.MethodPost, "/api/v1/departments", bytes.NewBufferString(dept1Payload))
	req1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("Failed to create Engineering dept: %s", w1.Body.String())
	}
	var dept1Resp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w1.Body.Bytes(), &dept1Resp)
	engID := dept1Resp.Data.ID

	dept2Payload := `{"name":"Marketing","description":"Growth and Branding"}`
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest(http.MethodPost, "/api/v1/departments", bytes.NewBufferString(dept2Payload))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)
	var dept2Resp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w2.Body.Bytes(), &dept2Resp)
	mktID := dept2Resp.Data.ID

	// -------------------------------------------------------------
	// TEST 2: Validate User Cannot Reference Nonexistent Department
	// -------------------------------------------------------------
	invalidUserPayload := `{"name":"Ghost","email":"ghost@company.com","password":"password123","department_id":99999}`
	wInvalid := httptest.NewRecorder()
	reqInvalid, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(invalidUserPayload))
	reqInvalid.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wInvalid, reqInvalid)

	// Should be rejected by Service layer validation
	if wInvalid.Code != http.StatusNotFound {
		t.Fatalf("Expected 404 Not Found for invalid department reference, got %d: %s", wInvalid.Code, wInvalid.Body.String())
	}

	// -------------------------------------------------------------
	// TEST 3: Create Users in Respective Departments
	// -------------------------------------------------------------
	// User 1 in Engineering
	u1Payload := fmt.Sprintf(`{"name":"Eng User 1","email":"eng1@company.com","password":"password123","department_id":%d}`, engID)
	wU1 := httptest.NewRecorder()
	reqU1, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(u1Payload))
	reqU1.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wU1, reqU1)
	if wU1.Code != http.StatusCreated {
		t.Fatalf("Failed to create Eng User 1: %s", wU1.Body.String())
	}

	// User 2 in Engineering
	u2Payload := fmt.Sprintf(`{"name":"Eng User 2","email":"eng2@company.com","password":"password123","department_id":%d}`, engID)
	wU2 := httptest.NewRecorder()
	reqU2, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(u2Payload))
	reqU2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wU2, reqU2)
	if wU2.Code != http.StatusCreated {
		t.Fatalf("Failed to create Eng User 2: %s", wU2.Body.String())
	}

	// User 3 in Marketing
	u3Payload := fmt.Sprintf(`{"name":"Mkt User 1","email":"mkt1@company.com","password":"password123","department_id":%d}`, mktID)
	wU3 := httptest.NewRecorder()
	reqU3, _ := http.NewRequest(http.MethodPost, "/api/v1/users", bytes.NewBufferString(u3Payload))
	reqU3.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wU3, reqU3)
	if wU3.Code != http.StatusCreated {
		t.Fatalf("Failed to create Mkt User 1: %s", wU3.Body.String())
	}

	// -------------------------------------------------------------
	// TEST 4: Department-Based User Filtering
	// -------------------------------------------------------------
	// Filter by Engineering Department
	wFilterEng := httptest.NewRecorder()
	reqFilterEng, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users?department_id=%d", engID), nil)
	router.ServeHTTP(wFilterEng, reqFilterEng)
	if wFilterEng.Code != http.StatusOK {
		t.Fatalf("Filter by Eng failed: %s", wFilterEng.Body.String())
	}
	var engFilterResp struct {
		Data []struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
		Pagination struct {
			TotalCount int `json:"total_count"`
		} `json:"pagination"`
	}
	_ = json.Unmarshal(wFilterEng.Body.Bytes(), &engFilterResp)
	if engFilterResp.Pagination.TotalCount != 2 || len(engFilterResp.Data) != 2 {
		t.Errorf("Expected 2 users in Engineering, got %d", engFilterResp.Pagination.TotalCount)
	}

	// Filter by Marketing Department
	wFilterMkt := httptest.NewRecorder()
	reqFilterMkt, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("/api/v1/users?department_id=%d", mktID), nil)
	router.ServeHTTP(wFilterMkt, reqFilterMkt)
	var mktFilterResp struct {
		Pagination struct {
			TotalCount int `json:"total_count"`
		} `json:"pagination"`
	}
	_ = json.Unmarshal(wFilterMkt.Body.Bytes(), &mktFilterResp)
	if mktFilterResp.Pagination.TotalCount != 1 {
		t.Errorf("Expected 1 user in Marketing, got %d", mktFilterResp.Pagination.TotalCount)
	}

	// -------------------------------------------------------------
	// TEST 5: Update Department
	// -------------------------------------------------------------
	updatePayload := `{"name":"Engineering & Architecture","description":"Updated description"}`
	wUpdate := httptest.NewRecorder()
	reqUpdate, _ := http.NewRequest(http.MethodPut, fmt.Sprintf("/api/v1/departments/%d", engID), bytes.NewBufferString(updatePayload))
	reqUpdate.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(wUpdate, reqUpdate)
	if wUpdate.Code != http.StatusOK {
		t.Fatalf("Failed to update department: %s", wUpdate.Body.String())
	}

	// -------------------------------------------------------------
	// TEST 6: Delete Department (Verifies ON DELETE SET NULL)
	// -------------------------------------------------------------
	wDel := httptest.NewRecorder()
	reqDel, _ := http.NewRequest(http.MethodDelete, fmt.Sprintf("/api/v1/departments/%d", engID), nil)
	router.ServeHTTP(wDel, reqDel)
	if wDel.Code != http.StatusOK {
		t.Fatalf("Failed to delete department: %s", wDel.Body.String())
	}

	// Verify that user 1 still exists, but their department is now null
	var userDeptID *int
	err = db.QueryRowContext(ctx, "SELECT department_id FROM users WHERE email = 'eng1@company.com'").Scan(&userDeptID)
	if err != nil {
		t.Fatalf("Failed to query user after department deletion: %v", err)
	}
	if userDeptID != nil {
		t.Errorf("Expected user's department_id to be NULL after department deletion, got %v", *userDeptID)
	}

	t.Log(" Department CRUD, User-Department validation, and filtering verified successfully!")
}
