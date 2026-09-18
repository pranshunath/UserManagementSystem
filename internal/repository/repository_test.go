package repository

import (
	"context"
	"testing"
	"time"

	"usermanagementsystem/internal/database"
	"usermanagementsystem/internal/model"
)

func setupTestDB(t *testing.T) *database.Config {
	return &database.Config{
		Host:     "localhost",
		Port:     "5432",
		User:     "postgres",
		Password: "postgrespassword",
		DBName:   "usermanagement",
		SSLMode:  "disable",
	}
}

func TestRepositoriesCRUD(t *testing.T) {
	cfg := setupTestDB(t)
	db, err := database.NewPostgresDB(*cfg)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	deptRepo := NewDepartmentRepository(db)
	roleRepo := NewRoleRepository(db)
	userRepo := NewUserRepository(db)

	// Clean up test data if previous tests left any
	_, _ = db.ExecContext(ctx, "DELETE FROM user_roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM users")
	_, _ = db.ExecContext(ctx, "DELETE FROM roles")
	_, _ = db.ExecContext(ctx, "DELETE FROM departments")

	// 1. Test Department CRUD
	dept := &model.Department{
		Name:        "Engineering",
		Description: "Software development and infrastructure",
	}
	if err := deptRepo.Create(ctx, dept); err != nil {
		t.Fatalf("Failed to create department: %v", err)
	}
	if dept.ID == 0 {
		t.Fatalf("Expected department ID to be set by RETURNING clause")
	}

	fetchedDept, err := deptRepo.GetByID(ctx, dept.ID)
	if err != nil {
		t.Fatalf("Failed to get department by ID: %v", err)
	}
	if fetchedDept.Name != dept.Name {
		t.Errorf("Expected dept name %s, got %s", dept.Name, fetchedDept.Name)
	}

	// 2. Test Role CRUD
	role := &model.Role{
		Name:        "Admin",
		Description: "Full administrative access",
	}
	if err := roleRepo.Create(ctx, role); err != nil {
		t.Fatalf("Failed to create role: %v", err)
	}
	if role.ID == 0 {
		t.Fatalf("Expected role ID to be set")
	}

	// 3. Test User CRUD
	user := &model.User{
		Name:         "Pranshu",
		Email:        "pranshu@company.com",
		PasswordHash: "hashed_secret_password",
		DepartmentID: &dept.ID,
		Status:       "active",
	}
	if err := userRepo.Create(ctx, user); err != nil {
		t.Fatalf("Failed to create user: %v", err)
	}
	if user.ID == 0 {
		t.Fatalf("Expected user ID to be set")
	}

	// 4. Test Role Assignment (Many-to-Many)
	if err := roleRepo.AssignRoleToUser(ctx, user.ID, role.ID); err != nil {
		t.Fatalf("Failed to assign role to user: %v", err)
	}

	userRoles, err := roleRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to fetch user roles: %v", err)
	}
	if len(userRoles) != 1 || userRoles[0].Name != "Admin" {
		t.Fatalf("Expected 1 role 'Admin', got %+v", userRoles)
	}

	// 5. Test User GetByID with Department Joined
	fetchedUser, err := userRepo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("Failed to fetch user by ID: %v", err)
	}
	if fetchedUser.Department == nil || fetchedUser.Department.Name != "Engineering" {
		t.Errorf("Expected joined department 'Engineering', got %+v", fetchedUser.Department)
	}

	// 6. Test User Filtering & Pagination
	filter := model.UserFilter{
		Search: "pranshu",
		Limit:  10,
		Offset: 0,
	}
	users, totalCount, err := userRepo.List(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to list users with filter: %v", err)
	}
	if totalCount != 1 || len(users) != 1 {
		t.Errorf("Expected 1 user in list, got count=%d, len=%d", totalCount, len(users))
	}

	// 7. Test Soft Delete
	if err := userRepo.Delete(ctx, user.ID); err != nil {
		t.Fatalf("Failed to soft-delete user: %v", err)
	}

	// Verify GetByID returns ErrNotFound after soft deletion
	_, err = userRepo.GetByID(ctx, user.ID)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound after soft deletion, got: %v", err)
	}

	t.Log(" All Repository layer CRUD tests passed successfully!")
}
