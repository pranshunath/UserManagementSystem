package tests

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	googlegrpc "google.golang.org/grpc"

	"usermanagementsystem/internal/auth"
	"usermanagementsystem/internal/cache"
	"usermanagementsystem/internal/database"
	"usermanagementsystem/internal/grpc"
	"usermanagementsystem/internal/handler"
	"usermanagementsystem/internal/middleware"
	"usermanagementsystem/internal/repository"
	"usermanagementsystem/internal/service"
	v1 "usermanagementsystem/proto"
)

// setupE2ETestEnvironment boots the full production end-to-end architecture:
// Gin REST Gateway (with all 5 production middlewares) -> gRPC Client -> gRPC Server (with Logging Interceptor) -> Domain Services -> Repositories -> Postgres & Redis.
func setupE2ETestEnvironment(t *testing.T) (*gin.Engine, *sql.DB, cache.UserCache, service.UserService, service.RoleService, func()) {
	gin.SetMode(gin.TestMode)

	// 1. PostgreSQL connection
	dbCfg := database.Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "postgrespassword",
		DBName:   "usermanagement",
		SSLMode:  "disable",
	}
	db, err := database.NewPostgresDB(dbCfg)
	if err != nil {
		t.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Clear test tables to ensure a clean state
	_, _ = db.ExecContext(ctx, "DELETE FROM user_roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM users")
	_, _ = db.ExecContext(ctx, "DELETE FROM roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM departments")

	// 2. Redis Cache connection (port 6380)
	redisCfg := cache.Config{
		Host:     "localhost",
		Port:     "6380",
		Password: "",
		DB:       0,
	}
	userCache, err := cache.NewRedisUserCache(redisCfg)
	if err != nil {
		t.Logf("Notice: Redis not reachable on :6380 (%v). Cache tests will use nil.", err)
		userCache = nil
	}

	// 3. Repositories
	deptRepo := repository.NewDepartmentRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)

	// 4. Token Manager & Domain Services
	jwtSecret := "e2e-super-secret-key-for-testing-purposes-123456"
	tokenManager := auth.NewTokenManager(jwtSecret, 24*time.Hour)

	deptService := service.NewDepartmentService(deptRepo, userRepo, userCache)
	roleService := service.NewRoleService(roleRepo, userRepo, userCache)
	userService := service.NewUserService(userRepo, deptRepo, roleRepo, userCache)
	authService := service.NewAuthService(userRepo, roleRepo, tokenManager)

	// 5. Live gRPC Server with Logging Interceptor
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to bind gRPC TCP listener: %v", err)
	}

	grpcServer := googlegrpc.NewServer(
		googlegrpc.UnaryInterceptor(grpc.UnaryServerLoggingInterceptor()),
	)
	serverHandler := grpc.NewServer(userService, deptService, roleService, authService)
	v1.RegisterUserServiceServer(grpcServer, serverHandler)
	v1.RegisterDepartmentServiceServer(grpcServer, serverHandler)
	v1.RegisterRoleServiceServer(grpcServer, serverHandler)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	// 6. Connect gRPC Client Stub
	grpcClient, err := grpc.NewClient(lis.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial gRPC server: %v", err)
	}

	// 7. Gin Engine with ALL 5 Production Middlewares
	router := gin.New()
	router.Use(middleware.RequestID())
	router.Use(middleware.StructuredLogger())
	router.Use(middleware.CORS())
	router.Use(middleware.Recovery())

	// Handlers
	userHandler := handler.NewUserHandler(grpcClient.User)
	deptHandler := handler.NewDepartmentHandler(grpcClient.Department)
	roleHandler := handler.NewRoleHandler(grpcClient.Role)
	authHandler := handler.NewAuthHandler(grpcClient.User)

	// Route definitions matching cmd/api/main.go
	v1Group := router.Group("/api/v1")
	{
		v1Group.POST("/auth/login", authHandler.Login)

		protected := v1Group.Group("")
		protected.Use(middleware.AuthMiddleware(tokenManager))
		{
			// Users: Read = Auth, Write = Admin/Manager, Delete = Admin
			users := protected.Group("/users")
			{
				users.GET("", userHandler.ListUsers)
				users.GET("/:id", userHandler.GetUser)
				users.POST("", middleware.RequireRole("Admin", "Manager"), userHandler.CreateUser)
				users.PUT("/:id", middleware.RequireRole("Admin", "Manager"), userHandler.UpdateUser)
				users.DELETE("/:id", middleware.RequireRole("Admin"), userHandler.DeleteUser)
			}

			// Departments: Read = Auth, Write = Admin
			departments := protected.Group("/departments")
			{
				departments.GET("", deptHandler.ListDepartments)
				departments.GET("/:id", deptHandler.GetDepartment)
				departments.POST("", middleware.RequireRole("Admin"), deptHandler.CreateDepartment)
				departments.PUT("/:id", middleware.RequireRole("Admin"), deptHandler.UpdateDepartment)
				departments.DELETE("/:id", middleware.RequireRole("Admin"), deptHandler.DeleteDepartment)
			}

			// Roles: Read = Auth, Write = Admin
			roles := protected.Group("/roles")
			{
				roles.GET("", roleHandler.ListRoles)
				roles.GET("/:id", roleHandler.GetRole)
				roles.POST("", middleware.RequireRole("Admin"), roleHandler.CreateRole)
				roles.PUT("/:id", middleware.RequireRole("Admin"), roleHandler.UpdateRole)
				roles.DELETE("/:id", middleware.RequireRole("Admin"), roleHandler.DeleteRole)
			}
		}
	}

	cleanup := func() {
		grpcClient.Close()
		grpcServer.GracefulStop()
		_ = lis.Close()
		_ = db.Close()
	}

	return router, db, userCache, userService, roleService, cleanup
}

// Helper to make HTTP requests
func performRequest(router *gin.Engine, method, path, token string, body []byte) *httptest.ResponseRecorder {
	var bodyReader *bytes.Buffer
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	} else {
		bodyReader = bytes.NewBuffer(nil)
	}

	req, _ := http.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ---------------------------------------------------------------------------
// TEST 1: Full Authentication & Multi-Role RBAC Authorization Matrix
// ---------------------------------------------------------------------------
func TestComprehensive_AuthAndRBACMatrix(t *testing.T) {
	router, _, _, userService, roleService, cleanup := setupE2ETestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	// 1. Seed Roles: Admin, Manager, Employee
	roleAdmin, err := roleService.CreateRole(ctx, "Admin", "System Administrator")
	if err != nil {
		t.Fatalf("Failed to create Admin role: %v", err)
	}
	roleManager, err := roleService.CreateRole(ctx, "Manager", "Department Manager")
	if err != nil {
		t.Fatalf("Failed to create Manager role: %v", err)
	}
	roleEmployee, err := roleService.CreateRole(ctx, "Employee", "Standard Employee")
	if err != nil {
		t.Fatalf("Failed to create Employee role: %v", err)
	}

	// 2. Seed Admin user via userService
	_, err = userService.CreateUser(ctx, service.CreateUserInput{
		Name:     "Super Admin",
		Email:    "admin@enterprise.com",
		Password: "AdminPassword123!",
		RoleIDs:  []int{roleAdmin.ID},
		Status:   "active",
	})
	if err != nil {
		t.Fatalf("Failed to create admin user: %v", err)
	}

	// 3. Login as Admin
	loginPayload, _ := json.Marshal(map[string]string{
		"email":    "admin@enterprise.com",
		"password": "AdminPassword123!",
	})
	w := performRequest(router, http.MethodPost, "/api/v1/auth/login", "", loginPayload)
	if w.Code != http.StatusOK {
		t.Fatalf("Admin login failed with code %d: %s", w.Code, w.Body.String())
	}

	var loginResp struct {
		Success bool `json:"success"`
		Data    struct {
			Token string   `json:"token"`
			Roles []string `json:"roles"`
		} `json:"data"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &loginResp)
	adminToken := loginResp.Data.Token
	if adminToken == "" {
		t.Fatal("Admin token was empty")
	}

	// Verify X-Request-ID response header
	reqID := w.Header().Get(middleware.HeaderRequestID)
	if reqID == "" {
		t.Error("Expected X-Request-ID header on HTTP response")
	}

	// 4. Test Unauthenticated Access
	wUnauth := performRequest(router, http.MethodGet, "/api/v1/users", "", nil)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized without token, got %d", wUnauth.Code)
	}

	// 5. Test Invalid Token
	wInvalid := performRequest(router, http.MethodGet, "/api/v1/users", "invalid.bogus.token", nil)
	if wInvalid.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized for invalid token, got %d", wInvalid.Code)
	}

	// 6. Admin creates a Manager and an Employee through the REST API
	createMgrPayload, _ := json.Marshal(map[string]interface{}{
		"name":        "Department Manager",
		"email":       "manager@enterprise.com",
		"password":    "ManagerPass123!",
		"role_ids":    []int{roleManager.ID},
		"status":      "active",
	})
	wCreateMgr := performRequest(router, http.MethodPost, "/api/v1/users", adminToken, createMgrPayload)
	if wCreateMgr.Code != http.StatusCreated {
		t.Fatalf("Failed to create Manager via REST: %d: %s", wCreateMgr.Code, wCreateMgr.Body.String())
	}

	createEmpPayload, _ := json.Marshal(map[string]interface{}{
		"name":        "Staff Employee",
		"email":       "employee@enterprise.com",
		"password":    "EmployeePass123!",
		"role_ids":    []int{roleEmployee.ID},
		"status":      "active",
	})
	wCreateEmp := performRequest(router, http.MethodPost, "/api/v1/users", adminToken, createEmpPayload)
	if wCreateEmp.Code != http.StatusCreated {
		t.Fatalf("Failed to create Employee via REST: %d: %s", wCreateEmp.Code, wCreateEmp.Body.String())
	}

	var empResp struct {
		Data struct {
			ID int `json:"id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wCreateEmp.Body.Bytes(), &empResp)
	empID := empResp.Data.ID

	// 7. Login as Manager and Employee to get respective JWTs
	mgrLogin, _ := json.Marshal(map[string]string{"email": "manager@enterprise.com", "password": "ManagerPass123!"})
	wMgrLog := performRequest(router, http.MethodPost, "/api/v1/auth/login", "", mgrLogin)
	_ = json.Unmarshal(wMgrLog.Body.Bytes(), &loginResp)
	managerToken := loginResp.Data.Token

	empLogin, _ := json.Marshal(map[string]string{"email": "employee@enterprise.com", "password": "EmployeePass123!"})
	wEmpLog := performRequest(router, http.MethodPost, "/api/v1/auth/login", "", empLogin)
	_ = json.Unmarshal(wEmpLog.Body.Bytes(), &loginResp)
	employeeToken := loginResp.Data.Token

	// 8. RBAC Matrix Verification:
	// a) Employee attempts to CREATE user -> Must be 403 Forbidden
	dummyUser, _ := json.Marshal(map[string]interface{}{
		"name": "Intruder", "email": "intruder@evil.org", "password": "Password123!", "role_ids": []int{roleEmployee.ID}, "status": "active",
	})
	wEmpCreate := performRequest(router, http.MethodPost, "/api/v1/users", employeeToken, dummyUser)
	if wEmpCreate.Code != http.StatusForbidden {
		t.Errorf("Employee expected 403 Forbidden on create user, got %d", wEmpCreate.Code)
	}

	// b) Employee attempts to DELETE user -> Must be 403 Forbidden
	wEmpDel := performRequest(router, http.MethodDelete, fmt.Sprintf("/api/v1/users/%d", empID), employeeToken, nil)
	if wEmpDel.Code != http.StatusForbidden {
		t.Errorf("Employee expected 403 Forbidden on delete user, got %d", wEmpDel.Code)
	}

	// c) Employee CAN READ users list -> Must be 200 OK
	wEmpRead := performRequest(router, http.MethodGet, "/api/v1/users", employeeToken, nil)
	if wEmpRead.Code != http.StatusOK {
		t.Errorf("Employee expected 200 OK on list users, got %d", wEmpRead.Code)
	}

	// d) Manager CAN CREATE user -> Must be 201 Created
	contractorPayload, _ := json.Marshal(map[string]interface{}{
		"name": "Contractor", "email": "contractor@enterprise.com", "password": "Password123!", "role_ids": []int{roleEmployee.ID}, "status": "active",
	})
	wMgrCreate := performRequest(router, http.MethodPost, "/api/v1/users", managerToken, contractorPayload)
	if wMgrCreate.Code != http.StatusCreated {
		t.Errorf("Manager expected 201 Created on create user, got %d: %s", wMgrCreate.Code, wMgrCreate.Body.String())
	}

	// e) Manager CANNOT DELETE user -> Must be 403 Forbidden
	wMgrDel := performRequest(router, http.MethodDelete, fmt.Sprintf("/api/v1/users/%d", empID), managerToken, nil)
	if wMgrDel.Code != http.StatusForbidden {
		t.Errorf("Manager expected 403 Forbidden on delete user, got %d", wMgrDel.Code)
	}

	// f) Admin CAN DELETE user -> Must be 200 OK
	wAdminDel := performRequest(router, http.MethodDelete, fmt.Sprintf("/api/v1/users/%d", empID), adminToken, nil)
	if wAdminDel.Code != http.StatusOK {
		t.Errorf("Admin expected 200 OK on delete user, got %d: %s", wAdminDel.Code, wAdminDel.Body.String())
	}
}

// ---------------------------------------------------------------------------
// TEST 2: Redis Cache Invalidation & Cache Consistency
// ---------------------------------------------------------------------------
func TestComprehensive_RedisCacheConsistency(t *testing.T) {
	router, _, userCache, userService, roleService, cleanup := setupE2ETestEnvironment(t)
	defer cleanup()

	if userCache == nil {
		t.Skip("Redis cache is nil; skipping cache consistency test")
	}

	ctx := context.Background()

	// Seed Admin
	roleAdmin, _ := roleService.CreateRole(ctx, "Admin", "Admin")
	_, _ = userService.CreateUser(ctx, service.CreateUserInput{
		Name:     "Admin",
		Email:    "admin_cache@test.com",
		Password: "AdminPass123!",
		RoleIDs:  []int{roleAdmin.ID},
		Status:   "active",
	})

	// Login
	loginPayload, _ := json.Marshal(map[string]string{"email": "admin_cache@test.com", "password": "AdminPass123!"})
	wLogin := performRequest(router, http.MethodPost, "/api/v1/auth/login", "", loginPayload)
	var loginResp struct {
		Data struct{ Token string } `json:"data"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.Token

	// 1. Create a user
	createPayload, _ := json.Marshal(map[string]interface{}{
		"name":     "Initial Name",
		"email":    "cached_user@test.com",
		"password": "Password123!",
		"role_ids": []int{roleAdmin.ID},
		"status":   "active",
	})
	wCreate := performRequest(router, http.MethodPost, "/api/v1/users", token, createPayload)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("Failed to create user: %s", wCreate.Body.String())
	}

	var userResp struct {
		Data struct {
			ID   int    `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &userResp)
	userID := userResp.Data.ID

	// 2. First GET: Cache Miss -> loads from DB and populates Redis
	wGet1 := performRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/users/%d", userID), token, nil)
	if wGet1.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on 1st GET, got %d", wGet1.Code)
	}

	// Verify key exists in Redis cache directly
	cachedUser, err := userCache.GetUser(ctx, userID)
	if err != nil {
		t.Fatalf("Expected user to be cached in Redis after GET, got err: %v", err)
	}
	if cachedUser.Name != "Initial Name" {
		t.Errorf("Cached user name mismatch: got %s, want Initial Name", cachedUser.Name)
	}

	// 3. Second GET: Cache Hit (Serviced from RAM)
	wGet2 := performRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/users/%d", userID), token, nil)
	if wGet2.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on 2nd GET (cache hit), got %d", wGet2.Code)
	}

	// 4. Update user via PUT: Must invalidate Redis cache and write-through reload
	updatePayload, _ := json.Marshal(map[string]interface{}{
		"name":   "Brand New Updated Name",
		"email":  "cached_user@test.com",
		"status": "active",
	})
	wPut := performRequest(router, http.MethodPut, fmt.Sprintf("/api/v1/users/%d", userID), token, updatePayload)
	if wPut.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on PUT update, got %d: %s", wPut.Code, wPut.Body.String())
	}

	// Verify Redis cache now contains the NEW updated name (write-through consistency)
	cachedUserAfterPut, err := userCache.GetUser(ctx, userID)
	if err != nil {
		t.Fatalf("Expected user to be re-cached with updated values, got err: %v", err)
	}
	if cachedUserAfterPut.Name != "Brand New Updated Name" {
		t.Errorf("Expected cached user name to be 'Brand New Updated Name', got: %s", cachedUserAfterPut.Name)
	}

	// 5. Delete user: Must evict key from Redis
	wDel := performRequest(router, http.MethodDelete, fmt.Sprintf("/api/v1/users/%d", userID), token, nil)
	if wDel.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on DELETE, got %d: %s", wDel.Code, wDel.Body.String())
	}

	// Verify cache key was completely removed from Redis
	_, err = userCache.GetUser(ctx, userID)
	if err == nil {
		t.Error("Expected cache MISS after DeleteUser, but found key in Redis")
	}
}

// ---------------------------------------------------------------------------
// TEST 3: Relational Invariants (ON DELETE SET NULL & ON DELETE CASCADE)
// ---------------------------------------------------------------------------
func TestComprehensive_RelationalInvariants(t *testing.T) {
	router, _, _, userService, roleService, cleanup := setupE2ETestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	// Seed Admin
	roleAdmin, _ := roleService.CreateRole(ctx, "Admin", "Admin")
	_, _ = userService.CreateUser(ctx, service.CreateUserInput{
		Name:     "Admin",
		Email:    "admin_rel@test.com",
		Password: "AdminPass123!",
		RoleIDs:  []int{roleAdmin.ID},
		Status:   "active",
	})

	// Login
	loginPayload, _ := json.Marshal(map[string]string{"email": "admin_rel@test.com", "password": "AdminPass123!"})
	wLogin := performRequest(router, http.MethodPost, "/api/v1/auth/login", "", loginPayload)
	var loginResp struct {
		Data struct{ Token string } `json:"data"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.Token

	// 1. Invariant A: Department ON DELETE SET NULL
	// Create department
	deptPayload, _ := json.Marshal(map[string]string{"name": "DevSecOps", "description": "Security ops"})
	wDept := performRequest(router, http.MethodPost, "/api/v1/departments", token, deptPayload)
	var deptResp struct {
		Data struct{ ID int } `json:"data"`
	}
	_ = json.Unmarshal(wDept.Body.Bytes(), &deptResp)
	deptID := deptResp.Data.ID

	// Create user in DevSecOps department
	userPayload, _ := json.Marshal(map[string]interface{}{
		"name":          "Sec Lead",
		"email":         "seclead@test.com",
		"password":      "Password123!",
		"department_id": deptID,
		"role_ids":      []int{roleAdmin.ID},
		"status":        "active",
	})
	wUser := performRequest(router, http.MethodPost, "/api/v1/users", token, userPayload)
	var userResp struct {
		Data struct{ ID int } `json:"data"`
	}
	_ = json.Unmarshal(wUser.Body.Bytes(), &userResp)
	userID := userResp.Data.ID

	// Delete department
	wDelDept := performRequest(router, http.MethodDelete, fmt.Sprintf("/api/v1/departments/%d", deptID), token, nil)
	if wDelDept.Code != http.StatusOK {
		t.Fatalf("Failed to delete department: %s", wDelDept.Body.String())
	}

	// Query user: verify user still exists and department_id is null
	wGetUser := performRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/users/%d", userID), token, nil)
	if wGetUser.Code != http.StatusOK {
		t.Fatalf("Expected user to still exist after department deletion, got %d", wGetUser.Code)
	}
	var checkUser struct {
		Data struct {
			DepartmentID *int `json:"department_id"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wGetUser.Body.Bytes(), &checkUser)
	if checkUser.Data.DepartmentID != nil {
		t.Errorf("Expected user's department_id to be NULL (ON DELETE SET NULL), got: %v", *checkUser.Data.DepartmentID)
	}

	// 2. Invariant B: Role ON DELETE CASCADE
	// Create custom role
	rolePayload, _ := json.Marshal(map[string]string{"name": "ExternalConsultant", "description": "Contractor"})
	wRole := performRequest(router, http.MethodPost, "/api/v1/roles", token, rolePayload)
	var customRoleResp struct {
		Data struct{ ID int } `json:"data"`
	}
	_ = json.Unmarshal(wRole.Body.Bytes(), &customRoleResp)
	customRoleID := customRoleResp.Data.ID

	// Assign custom role to user
	updateRolePayload, _ := json.Marshal(map[string]interface{}{
		"role_ids": []int{customRoleID},
	})
	performRequest(router, http.MethodPut, fmt.Sprintf("/api/v1/users/%d", userID), token, updateRolePayload)

	// Delete custom role
	wDelRole := performRequest(router, http.MethodDelete, fmt.Sprintf("/api/v1/roles/%d", customRoleID), token, nil)
	if wDelRole.Code != http.StatusOK {
		t.Fatalf("Failed to delete custom role: %s", wDelRole.Body.String())
	}

	// Query user: verify user exists and user_roles association was cleanly cascaded away
	wGetUserAfterRole := performRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/users/%d", userID), token, nil)
	var checkUserRoles struct {
		Data struct {
			Roles []struct{ ID int } `json:"roles"`
		} `json:"data"`
	}
	_ = json.Unmarshal(wGetUserAfterRole.Body.Bytes(), &checkUserRoles)
	for _, r := range checkUserRoles.Data.Roles {
		if r.ID == customRoleID {
			t.Errorf("Expected role %d to be cascaded away from user, but still found in user roles", customRoleID)
		}
	}
}

// ---------------------------------------------------------------------------
// TEST 4: Edge Cases, Validations, and Soft Deletion
// ---------------------------------------------------------------------------
func TestComprehensive_EdgeCasesAndSoftDelete(t *testing.T) {
	router, db, _, userService, roleService, cleanup := setupE2ETestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	// Seed Admin
	roleAdmin, _ := roleService.CreateRole(ctx, "Admin", "Admin")
	_, _ = userService.CreateUser(ctx, service.CreateUserInput{
		Name:     "Admin",
		Email:    "admin_edge@test.com",
		Password: "AdminPass123!",
		RoleIDs:  []int{roleAdmin.ID},
		Status:   "active",
	})

	// Login
	loginPayload, _ := json.Marshal(map[string]string{"email": "admin_edge@test.com", "password": "AdminPass123!"})
	wLogin := performRequest(router, http.MethodPost, "/api/v1/auth/login", "", loginPayload)
	var loginResp struct {
		Data struct{ Token string } `json:"data"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.Token

	// 1. Create a user
	createPayload, _ := json.Marshal(map[string]interface{}{
		"name":     "Target User",
		"email":    "unique_edge@test.com",
		"password": "Password123!",
		"role_ids": []int{roleAdmin.ID},
		"status":   "active",
	})
	wCreate := performRequest(router, http.MethodPost, "/api/v1/users", token, createPayload)
	if wCreate.Code != http.StatusCreated {
		t.Fatalf("Failed to create user: %s", wCreate.Body.String())
	}
	var userResp struct {
		Data struct{ ID int } `json:"data"`
	}
	_ = json.Unmarshal(wCreate.Body.Bytes(), &userResp)
	userID := userResp.Data.ID

	// 2. Duplicate Email creation -> Must return 409 Conflict
	wDup := performRequest(router, http.MethodPost, "/api/v1/users", token, createPayload)
	if wDup.Code != http.StatusConflict {
		t.Errorf("Expected 409 Conflict for duplicate email, got %d: %s", wDup.Code, wDup.Body.String())
	}

	// 3. Short Password (< 8 chars) -> Must return 400 Bad Request
	shortPassPayload, _ := json.Marshal(map[string]interface{}{
		"name": "Short Pass", "email": "short@test.com", "password": "123", "role_ids": []int{roleAdmin.ID}, "status": "active",
	})
	wShort := performRequest(router, http.MethodPost, "/api/v1/users", token, shortPassPayload)
	if wShort.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for short password, got %d", wShort.Code)
	}

	// 4. Non-existent ID -> Must return 404 Not Found
	wNotFound := performRequest(router, http.MethodGet, "/api/v1/users/999999", token, nil)
	if wNotFound.Code != http.StatusNotFound {
		t.Errorf("Expected 404 Not Found for non-existent user, got %d", wNotFound.Code)
	}

	// 5. Soft-Delete Execution
	wDel := performRequest(router, http.MethodDelete, fmt.Sprintf("/api/v1/users/%d", userID), token, nil)
	if wDel.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK on soft delete, got %d: %s", wDel.Code, wDel.Body.String())
	}

	// Soft-deleted user must not be found via GET /users/:id
	wGetDeleted := performRequest(router, http.MethodGet, fmt.Sprintf("/api/v1/users/%d", userID), token, nil)
	if wGetDeleted.Code != http.StatusNotFound {
		t.Errorf("Expected 404 Not Found for soft-deleted user, got %d", wGetDeleted.Code)
	}

	// Soft-deleted user cannot authenticate
	deletedLogin, _ := json.Marshal(map[string]string{"email": "unique_edge@test.com", "password": "Password123!"})
	wDeletedLogin := performRequest(router, http.MethodPost, "/api/v1/auth/login", "", deletedLogin)
	if wDeletedLogin.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized for soft-deleted user login, got %d", wDeletedLogin.Code)
	}

	// Raw DB check: deleted_at column must NOT be null (soft delete proof)
	var deletedAt *time.Time
	err := db.QueryRowContext(ctx, "SELECT deleted_at FROM users WHERE id = $1", userID).Scan(&deletedAt)
	if err != nil {
		t.Fatalf("Failed to query raw deleted user from database: %v", err)
	}
	if deletedAt == nil {
		t.Error("Expected deleted_at to be populated in database, but got NULL")
	}
}

// ---------------------------------------------------------------------------
// TEST 5: Concurrency and Connection Pool Stress Test
// ---------------------------------------------------------------------------
func TestComprehensive_ConcurrencyAndStress(t *testing.T) {
	router, _, _, userService, roleService, cleanup := setupE2ETestEnvironment(t)
	defer cleanup()

	ctx := context.Background()

	// Seed Admin
	roleAdmin, _ := roleService.CreateRole(ctx, "Admin", "Admin")
	_, _ = userService.CreateUser(ctx, service.CreateUserInput{
		Name:     "Admin",
		Email:    "admin_stress@test.com",
		Password: "AdminPass123!",
		RoleIDs:  []int{roleAdmin.ID},
		Status:   "active",
	})

	// Login
	loginPayload, _ := json.Marshal(map[string]string{"email": "admin_stress@test.com", "password": "AdminPass123!"})
	wLogin := performRequest(router, http.MethodPost, "/api/v1/auth/login", "", loginPayload)
	var loginResp struct {
		Data struct{ Token string } `json:"data"`
	}
	_ = json.Unmarshal(wLogin.Body.Bytes(), &loginResp)
	token := loginResp.Data.Token

	// Seed 5 sample users
	for i := 1; i <= 5; i++ {
		p, _ := json.Marshal(map[string]interface{}{
			"name":     fmt.Sprintf("Stress User %d", i),
			"email":    fmt.Sprintf("stress_%d@test.com", i),
			"password": "Password123!",
			"role_ids": []int{roleAdmin.ID},
			"status":   "active",
		})
		performRequest(router, http.MethodPost, "/api/v1/users", token, p)
	}

	// Concurrently dispatch 25 requests across 25 goroutines
	var wg sync.WaitGroup
	errChan := make(chan error, 25)

	for i := 0; i < 25; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			path := "/api/v1/users?limit=10"
			if workerID%2 == 0 {
				path = "/api/v1/users?search=Stress"
			}
			w := performRequest(router, http.MethodGet, path, token, nil)
			if w.Code != http.StatusOK {
				errChan <- fmt.Errorf("worker %d received status %d: %s", workerID, w.Code, w.Body.String())
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	for err := range errChan {
		t.Errorf("Concurrent request failure: %v", err)
	}
}
