package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"usermanagementsystem/internal/auth"
	"usermanagementsystem/internal/database"
	"usermanagementsystem/internal/repository"
	"usermanagementsystem/internal/service"
	v1 "usermanagementsystem/proto"
)

func setupGRPCTestServer(t *testing.T) (*grpc.Server, string, func()) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// Clear test data
	_, _ = db.ExecContext(ctx, "DELETE FROM user_roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM users")
	_, _ = db.ExecContext(ctx, "DELETE FROM roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM departments")

	deptRepo := repository.NewDepartmentRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)

	deptService := service.NewDepartmentService(deptRepo)
	roleService := service.NewRoleService(roleRepo)
	userService := service.NewUserService(userRepo, deptRepo, roleRepo, nil)

	tokenManager := auth.NewTokenManager("test-jwt-secret", 24*time.Hour)
	authService := service.NewAuthService(userRepo, roleRepo, tokenManager)

	// Listen on dynamic random available TCP port
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to bind TCP listener: %v", err)
	}

	grpcServer := grpc.NewServer()
	srv := NewServer(userService, deptService, roleService, authService)

	v1.RegisterUserServiceServer(grpcServer, srv)
	v1.RegisterDepartmentServiceServer(grpcServer, srv)
	v1.RegisterRoleServiceServer(grpcServer, srv)

	go func() {
		_ = grpcServer.Serve(lis)
	}()

	teardown := func() {
		grpcServer.GracefulStop()
		_ = lis.Close()
		_ = db.Close()
	}

	return grpcServer, lis.Addr().String(), teardown
}

func TestGRPCUserLifecycle(t *testing.T) {
	_, addr, teardown := setupGRPCTestServer(t)
	defer teardown()

	client, err := NewClient(addr)
	if err != nil {
		t.Fatalf("Failed to connect gRPC client: %v", err)
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Create Department via gRPC
	deptResp, err := client.Department.CreateDepartment(ctx, &v1.CreateDepartmentRequest{
		Name:        "Platform Engineering",
		Description: "Cloud and Infrastructure Operations",
	})
	if err != nil {
		t.Fatalf("CreateDepartment RPC failed: %v", err)
	}
	deptID := deptResp.Department.Id

	// 2. Create Role via gRPC
	roleResp, err := client.Role.CreateRole(ctx, &v1.CreateRoleRequest{
		Name:        "DevOps",
		Description: "DevOps Engineer permissions",
	})
	if err != nil {
		t.Fatalf("CreateRole RPC failed: %v", err)
	}
	roleID := roleResp.Role.Id

	// 3. Create User via gRPC
	createResp, err := client.User.CreateUser(ctx, &v1.CreateUserRequest{
		Name:         "DevOps Lead",
		Email:        "devops@cloud.io",
		Password:     "supersecret123",
		DepartmentId: &deptID,
		RoleIds:      []int32{roleID},
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("CreateUser RPC failed: %v", err)
	}

	user := createResp.User
	if user.Id == 0 || user.Email != "devops@cloud.io" {
		t.Errorf("Unexpected user response: %+v", user)
	}
	if user.Department == nil || user.Department.Name != "Platform Engineering" {
		t.Errorf("Expected Department 'Platform Engineering', got %+v", user.Department)
	}
	if len(user.Roles) != 1 || user.Roles[0].Name != "DevOps" {
		t.Errorf("Expected 1 role 'DevOps', got %+v", user.Roles)
	}

	// 4. Test Duplicate Email returns codes.AlreadyExists
	_, err = client.User.CreateUser(ctx, &v1.CreateUserRequest{
		Name:     "Duplicate",
		Email:    "devops@cloud.io",
		Password: "supersecret123",
	})
	if status.Code(err) != codes.AlreadyExists {
		t.Errorf("Expected codes.AlreadyExists, got %v (code: %v)", err, status.Code(err))
	}

	// 5. Get User via gRPC
	getResp, err := client.User.GetUser(ctx, &v1.GetUserRequest{Id: user.Id})
	if err != nil {
		t.Fatalf("GetUser RPC failed: %v", err)
	}
	if getResp.User.Id != user.Id {
		t.Errorf("Expected user ID %d, got %d", user.Id, getResp.User.Id)
	}

	// 6. List Users via gRPC with search
	listResp, err := client.User.ListUsers(ctx, &v1.ListUsersRequest{
		Page:   1,
		Limit:  10,
		Search: "DevOps",
	})
	if err != nil {
		t.Fatalf("ListUsers RPC failed: %v", err)
	}
	if listResp.TotalCount != 1 || len(listResp.Users) != 1 {
		t.Errorf("Expected 1 user in ListUsers, got count=%d, len=%d", listResp.TotalCount, len(listResp.Users))
	}

	// 7. Update User via gRPC
	updateResp, err := client.User.UpdateUser(ctx, &v1.UpdateUserRequest{
		Id:     user.Id,
		Name:   "Principal DevOps Lead",
		Status: "active",
	})
	if err != nil {
		t.Fatalf("UpdateUser RPC failed: %v", err)
	}
	if updateResp.User.Name != "Principal DevOps Lead" {
		t.Errorf("Expected updated name, got %s", updateResp.User.Name)
	}

	// 8. Delete User via gRPC
	delResp, err := client.User.DeleteUser(ctx, &v1.DeleteUserRequest{Id: user.Id})
	if err != nil {
		t.Fatalf("DeleteUser RPC failed: %v", err)
	}
	if !delResp.Success {
		t.Errorf("Expected success = true")
	}

	// 9. Verify GetUser on deleted user returns codes.NotFound
	_, err = client.User.GetUser(ctx, &v1.GetUserRequest{Id: user.Id})
	if status.Code(err) != codes.NotFound {
		t.Errorf("Expected codes.NotFound after delete, got %v (code: %v)", err, status.Code(err))
	}

	t.Log(" All gRPC RPCs (CreateUser, GetUser, ListUsers, UpdateUser, DeleteUser) tested and passed!")
}
