package tests

import (
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
	"usermanagementsystem/internal/middleware"
	"usermanagementsystem/internal/model"
	"usermanagementsystem/internal/repository"
	"usermanagementsystem/internal/service"
	v1 "usermanagementsystem/proto"
)

func TestAdvancedSearchFilteringSortingAndPagination(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Clear test tables
	_, _ = db.ExecContext(ctx, "DELETE FROM user_roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM users")
	_, _ = db.ExecContext(ctx, "DELETE FROM roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM departments")

	// 2. Setup Repositories and Services
	deptRepo := repository.NewDepartmentRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)

	jwtSecret := "test-pagination-secret"
	tokenManager := auth.NewTokenManager(jwtSecret, 2*time.Hour)

	deptService := service.NewDepartmentService(deptRepo)
	roleService := service.NewRoleService(roleRepo)
	userService := service.NewUserService(userRepo, deptRepo, roleRepo, nil)
	authService := service.NewAuthService(userRepo, roleRepo, tokenManager)

	// 3. Start gRPC Server
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to bind TCP listener: %v", err)
	}
	defer lis.Close()

	grpcServer := googlegrpc.NewServer()
	serverHandler := grpc.NewServer(userService, deptService, roleService, authService)
	v1.RegisterUserServiceServer(grpcServer, serverHandler)
	v1.RegisterDepartmentServiceServer(grpcServer, serverHandler)
	v1.RegisterRoleServiceServer(grpcServer, serverHandler)

	go func() {
		_ = grpcServer.Serve(lis)
	}()
	defer grpcServer.Stop()

	// 4. Connect gRPC Client
	grpcClient, err := grpc.NewClient(lis.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial gRPC server: %v", err)
	}
	defer grpcClient.Close()

	// 5. Setup Gin Router
	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS())

	userHandler := handler.NewUserHandler(grpcClient.User)
	router.GET("/api/v1/users", userHandler.ListUsers)

	// -------------------------------------------------------------
	// SEED DATA: Departments, Roles, and 15 Users
	// -------------------------------------------------------------
	deptEng, _ := deptService.CreateDepartment(ctx, "Engineering", "Dev team")
	deptSales, _ := deptService.CreateDepartment(ctx, "Sales", "Sales team")

	roleDev, _ := roleService.CreateRole(ctx, "Developer", "Dev role")
	roleLead, _ := roleService.CreateRole(ctx, "Lead", "Lead role")

	createdUserIDs := make([]int, 0, 15)
	for i := 1; i <= 15; i++ {
		deptID := deptEng.ID
		roles := []int{roleDev.ID}
		status := "active"
		if i > 10 {
			deptID = deptSales.ID
			roles = []int{roleLead.ID}
		}
		if i%3 == 0 {
			status = "inactive"
		}

		u, err := userService.CreateUser(ctx, service.CreateUserInput{
			Name:         fmt.Sprintf("User%02d Test", i),
			Email:        fmt.Sprintf("user%02d@system.io", i),
			Password:     "password123",
			DepartmentID: &deptID,
			RoleIDs:      roles,
			Status:       status,
		})
		if err != nil {
			t.Fatalf("Failed to seed user %d: %v", i, err)
		}
		createdUserIDs = append(createdUserIDs, u.ID)
	}

	type ListResponse struct {
		Success bool       `json:"success"`
		Data    []model.User `json:"data"`
		Pagination struct {
			TotalCount int `json:"total_count"`
			Page       int `json:"page"`
			Limit      int `json:"limit"`
			TotalPages int `json:"total_pages"`
		} `json:"pagination"`
	}

	// -------------------------------------------------------------
	// TEST 1: Multi-Page Pagination with Deterministic Offset
	// -------------------------------------------------------------
	// Fetch Page 1 (limit=5)
	wP1 := httptest.NewRecorder()
	reqP1, _ := http.NewRequest(http.MethodGet, "/api/v1/users?page=1&limit=5", nil)
	router.ServeHTTP(wP1, reqP1)
	if wP1.Code != http.StatusOK {
		t.Fatalf("Page 1 request failed: %s", wP1.Body.String())
	}
	var respP1 ListResponse
	_ = json.Unmarshal(wP1.Body.Bytes(), &respP1)

	if respP1.Pagination.TotalCount != 15 {
		t.Fatalf("Expected total_count 15, got %d", respP1.Pagination.TotalCount)
	}
	if respP1.Pagination.TotalPages != 3 {
		t.Fatalf("Expected total_pages 3, got %d", respP1.Pagination.TotalPages)
	}
	if len(respP1.Data) != 5 {
		t.Fatalf("Expected 5 users on page 1, got %d", len(respP1.Data))
	}

	// Fetch Page 2 (limit=5)
	wP2 := httptest.NewRecorder()
	reqP2, _ := http.NewRequest(http.MethodGet, "/api/v1/users?page=2&limit=5", nil)
	router.ServeHTTP(wP2, reqP2)
	var respP2 ListResponse
	_ = json.Unmarshal(wP2.Body.Bytes(), &respP2)

	if len(respP2.Data) != 5 {
		t.Fatalf("Expected 5 users on page 2, got %d", len(respP2.Data))
	}

	// Verify no overlap between Page 1 and Page 2
	p1IDs := make(map[int]bool)
	for _, u := range respP1.Data {
		p1IDs[u.ID] = true
	}
	for _, u := range respP2.Data {
		if p1IDs[u.ID] {
			t.Fatalf("Deterministic pagination failed: user ID %d found on both page 1 and page 2", u.ID)
		}
	}

	// -------------------------------------------------------------
	// TEST 2: Limit Capping & Boundary Protection (limit=500 capped to 100)
	// -------------------------------------------------------------
	wCap := httptest.NewRecorder()
	reqCap, _ := http.NewRequest(http.MethodGet, "/api/v1/users?page=1&limit=500", nil)
	router.ServeHTTP(wCap, reqCap)
	var respCap ListResponse
	_ = json.Unmarshal(wCap.Body.Bytes(), &respCap)
	if respCap.Pagination.Limit != 100 {
		t.Fatalf("Expected limit 500 to be capped at 100, got %d", respCap.Pagination.Limit)
	}

	// -------------------------------------------------------------
	// TEST 3: Multi-Token Search (e.g. search="user03 test")
	// -------------------------------------------------------------
	wSearch := httptest.NewRecorder()
	reqSearch, _ := http.NewRequest(http.MethodGet, "/api/v1/users?search=user03+test", nil)
	router.ServeHTTP(wSearch, reqSearch)
	var respSearch ListResponse
	_ = json.Unmarshal(wSearch.Body.Bytes(), &respSearch)
	if len(respSearch.Data) != 1 || respSearch.Data[0].Email != "user03@system.io" {
		t.Fatalf("Expected 1 result for search 'user03 test', got %d", len(respSearch.Data))
	}

	// -------------------------------------------------------------
	// TEST 4: Dynamic Whitelisted Sorting (name DESC vs name ASC)
	// -------------------------------------------------------------
	wSortDesc := httptest.NewRecorder()
	reqSortDesc, _ := http.NewRequest(http.MethodGet, "/api/v1/users?page=1&limit=5&sort_by=name&sort_order=desc", nil)
	router.ServeHTTP(wSortDesc, reqSortDesc)
	var respSortDesc ListResponse
	_ = json.Unmarshal(wSortDesc.Body.Bytes(), &respSortDesc)

	if len(respSortDesc.Data) < 2 {
		t.Fatalf("Expected multiple records for sort test")
	}
	if respSortDesc.Data[0].Name < respSortDesc.Data[1].Name {
		t.Fatalf("Expected DESC order, but %s is before %s", respSortDesc.Data[0].Name, respSortDesc.Data[1].Name)
	}

	// -------------------------------------------------------------
	// TEST 5: SQL Injection Defense on sort_by parameter
	// -------------------------------------------------------------
	// Malicious sort_by input must be neutralized by the whitelist
	wInjection := httptest.NewRecorder()
	reqInjection, _ := http.NewRequest(http.MethodGet, "/api/v1/users?sort_by=id;+DROP+TABLE+users;+--&sort_order=desc", nil)
	router.ServeHTTP(wInjection, reqInjection)
	if wInjection.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK with neutralized sort parameter, got %d: %s", wInjection.Code, wInjection.Body.String())
	}

	// -------------------------------------------------------------
	// TEST 6: Combined Multi-Field Filtering (Department + Role + Status)
	// -------------------------------------------------------------
	wCombo := httptest.NewRecorder()
	comboURL := fmt.Sprintf("/api/v1/users?department_id=%d&role_id=%d&status=active&sort_by=id&sort_order=asc", deptSales.ID, roleLead.ID)
	reqCombo, _ := http.NewRequest(http.MethodGet, comboURL, nil)
	router.ServeHTTP(wCombo, reqCombo)
	var respCombo ListResponse
	_ = json.Unmarshal(wCombo.Body.Bytes(), &respCombo)

	// Sales users are users 11 to 15. Inactive are those divisible by 3 (12, 15).
	// So active Sales leads are 11, 13, 14 (3 users).
	if len(respCombo.Data) != 3 {
		t.Fatalf("Expected 3 combined filtered users, got %d", len(respCombo.Data))
	}
	for _, u := range respCombo.Data {
		if u.Status != "active" {
			t.Errorf("Expected status active, got %s", u.Status)
		}
		if u.DepartmentID == nil || *u.DepartmentID != deptSales.ID {
			t.Errorf("Expected dept %d, got %v", deptSales.ID, u.DepartmentID)
		}
	}

	t.Log(" Advanced search, multi-token filtering, SQL injection-safe sorting, and deterministic pagination verified successfully!")
}
