package service

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"usermanagementsystem/internal/cache"
	"usermanagementsystem/internal/database"
	"usermanagementsystem/internal/model"
	"usermanagementsystem/internal/repository"
)

func setupServiceTest(t *testing.T) (UserService, DepartmentService, RoleService) {
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

	// Clear tables
	_, _ = db.ExecContext(ctx, "DELETE FROM user_roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM users")
	_, _ = db.ExecContext(ctx, "DELETE FROM roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM departments")

	deptRepo := repository.NewDepartmentRepository(db)
	roleRepo := repository.NewRoleRepository(db)
	userRepo := repository.NewUserRepository(db)

	redisCfg := cache.Config{
		Host: "localhost",
		Port: "6380",
		DB:   2, // Use test DB 2
	}
	userCache, _ := cache.NewRedisUserCache(redisCfg)

	deptService := NewDepartmentService(deptRepo)
	roleService := NewRoleService(roleRepo)
	userService := NewUserService(userRepo, deptRepo, roleRepo, userCache)

	return userService, deptService, roleService
}

func TestUserService(t *testing.T) {
	userService, deptService, roleService := setupServiceTest(t)
	ctx := context.Background()

	// 1. Create a Department & Role first
	dept, err := deptService.CreateDepartment(ctx, "Design", "Product and UX design")
	if err != nil {
		t.Fatalf("Failed to create department: %v", err)
	}

	role, err := roleService.CreateRole(ctx, "Designer", "Product designer access")
	if err != nil {
		t.Fatalf("Failed to create role: %v", err)
	}

	// 2. Test Invalid Email Validation
	_, err = userService.CreateUser(ctx, CreateUserInput{
		Name:     "Alice",
		Email:    "invalid-email",
		Password: "strongpassword123",
	})
	if err != ErrInvalidEmail {
		t.Errorf("Expected ErrInvalidEmail, got %v", err)
	}

	// 3. Test Password Length Validation
	_, err = userService.CreateUser(ctx, CreateUserInput{
		Name:     "Alice",
		Email:    "alice@company.com",
		Password: "short",
	})
	if err != ErrPasswordTooWeak {
		t.Errorf("Expected ErrPasswordTooWeak, got %v", err)
	}

	// 4. Test Invalid Department Validation
	fakeDeptID := 99999
	_, err = userService.CreateUser(ctx, CreateUserInput{
		Name:         "Alice",
		Email:        "alice@company.com",
		Password:     "strongpassword123",
		DepartmentID: &fakeDeptID,
	})
	if err != ErrDepartmentNotFound {
		t.Errorf("Expected ErrDepartmentNotFound, got %v", err)
	}

	// 5. Test Invalid Role Validation
	fakeRoleID := 99999
	_, err = userService.CreateUser(ctx, CreateUserInput{
		Name:     "Alice",
		Email:    "alice@company.com",
		Password: "strongpassword123",
		RoleIDs:  []int{fakeRoleID},
	})
	if err != ErrRoleNotFound {
		t.Errorf("Expected ErrRoleNotFound, got %v", err)
	}

	// 6. Test Successful User Creation
	user, err := userService.CreateUser(ctx, CreateUserInput{
		Name:         "Alice Designer",
		Email:        "alice@company.com",
		Password:     "supersecret123",
		DepartmentID: &dept.ID,
		RoleIDs:      []int{role.ID},
		Status:       "active",
	})
	if err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}

	// Verify bcrypt hash validity
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte("supersecret123"))
	if err != nil {
		t.Errorf("bcrypt verification failed: %v", err)
	}

	// Verify roles and department association
	if user.Department == nil || user.Department.Name != "Design" {
		t.Errorf("Expected department 'Design', got %+v", user.Department)
	}
	if len(user.Roles) != 1 || user.Roles[0].Name != "Designer" {
		t.Errorf("Expected 1 role 'Designer', got %+v", user.Roles)
	}

	// 7. Test Duplicate Email Prevention
	_, err = userService.CreateUser(ctx, CreateUserInput{
		Name:     "Alice Imposter",
		Email:    "alice@company.com",
		Password: "differentpassword123",
	})
	if err != ErrEmailAlreadyInUse {
		t.Errorf("Expected ErrEmailAlreadyInUse, got %v", err)
	}

	// 8. Test Update User
	updatedUser, err := userService.UpdateUser(ctx, user.ID, UpdateUserInput{
		Name:   "Alice Senior Designer",
		Status: "active",
	})
	if err != nil {
		t.Fatalf("Failed to update user: %v", err)
	}
	if updatedUser.Name != "Alice Senior Designer" {
		t.Errorf("Expected updated name, got %s", updatedUser.Name)
	}

	// 9. Test List Users
	users, total, err := userService.ListUsers(ctx, model.UserFilter{Search: "Alice"})
	if err != nil {
		t.Fatalf("Failed to list users: %v", err)
	}
	if total != 1 || len(users) != 1 {
		t.Errorf("Expected 1 user, got total=%d, len=%d", total, len(users))
	}

	// 10. Test Delete User
	if err := userService.DeleteUser(ctx, user.ID); err != nil {
		t.Fatalf("Failed to delete user: %v", err)
	}

	_, err = userService.GetUser(ctx, user.ID)
	if err != ErrUserNotFound {
		t.Errorf("Expected ErrUserNotFound, got %v", err)
	}

	t.Log(" All Service Layer business logic tests passed successfully!")
}
