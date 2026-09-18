package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"usermanagementsystem/internal/cache"
	"usermanagementsystem/internal/model"
	"usermanagementsystem/internal/repository"
)

// CreateUserInput carries payload data for user creation.
type CreateUserInput struct {
	Name         string
	Email        string
	Password     string
	DepartmentID *int
	RoleIDs      []int
	Status       string
}

// UpdateUserInput carries payload data for updating an existing user.
type UpdateUserInput struct {
	Name         string
	Email        string
	DepartmentID *int
	RoleIDs      []int
	Status       string
}

// UserService defines the business logic contract for user management.
type UserService interface {
	CreateUser(ctx context.Context, input CreateUserInput) (*model.User, error)
	GetUser(ctx context.Context, id int) (*model.User, error)
	ListUsers(ctx context.Context, filter model.UserFilter) ([]model.User, int, error)
	UpdateUser(ctx context.Context, id int, input UpdateUserInput) (*model.User, error)
	DeleteUser(ctx context.Context, id int) error
}

type userService struct {
	userRepo repository.UserRepository
	deptRepo repository.DepartmentRepository
	roleRepo repository.RoleRepository
	cache    cache.UserCache
}

// NewUserService instantiates a UserService with injected repository and cache dependencies.
func NewUserService(
	userRepo repository.UserRepository,
	deptRepo repository.DepartmentRepository,
	roleRepo repository.RoleRepository,
	cache cache.UserCache,
) UserService {
	return &userService{
		userRepo: userRepo,
		deptRepo: deptRepo,
		roleRepo: roleRepo,
		cache:    cache,
	}
}

func (s *userService) CreateUser(ctx context.Context, input CreateUserInput) (*model.User, error) {
	// 1. Validate name
	input.Name = strings.TrimSpace(input.Name)
	if input.Name == "" {
		return nil, ErrEmptyName
	}

	// 2. Validate email format
	input.Email = strings.TrimSpace(strings.ToLower(input.Email))
	if _, err := mail.ParseAddress(input.Email); err != nil {
		return nil, ErrInvalidEmail
	}

	// 3. Validate password strength
	if len(input.Password) < 8 {
		return nil, ErrPasswordTooWeak
	}

	// 4. Check for duplicate email
	existing, err := s.userRepo.GetByEmail(ctx, input.Email)
	if err == nil && existing != nil {
		return nil, ErrEmailAlreadyInUse
	}

	// 5. Validate department existence if provided
	if input.DepartmentID != nil {
		_, err := s.deptRepo.GetByID(ctx, *input.DepartmentID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrDepartmentNotFound
			}
			return nil, err
		}
	}

	// 6. Validate roles existence if provided
	for _, roleID := range input.RoleIDs {
		_, err := s.roleRepo.GetByID(ctx, roleID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrRoleNotFound
			}
			return nil, err
		}
	}

	// 7. Validate status
	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status == "" {
		status = "active"
	} else if status != "active" && status != "inactive" {
		return nil, ErrInvalidStatus
	}

	// 8. Hash password securely using bcrypt
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Name:         input.Name,
		Email:        input.Email,
		PasswordHash: string(hashedBytes),
		DepartmentID: input.DepartmentID,
		Status:       status,
	}

	// 9. Persist user
	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, repository.ErrDuplicateEntry) {
			return nil, ErrEmailAlreadyInUse
		}
		return nil, err
	}

	// 10. Assign roles
	for _, roleID := range input.RoleIDs {
		if err := s.roleRepo.AssignRoleToUser(ctx, user.ID, roleID); err != nil {
			return nil, err
		}
	}

	// Load full user details with roles and department
	return s.GetUser(ctx, user.ID)
}

func (s *userService) GetUser(ctx context.Context, id int) (*model.User, error) {
	// 1. Check Redis Cache first (Cache Read)
	if s.cache != nil {
		cachedUser, err := s.cache.GetUser(ctx, id)
		if err == nil && cachedUser != nil {
			return cachedUser, nil
		}
	}

	// 2. Cache Miss: Query PostgreSQL
	fmt.Printf("[DB] Querying user %d from PostgreSQL...\n", id)
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Load roles associated with the user
	roles, err := s.roleRepo.GetRolesByUserID(ctx, id)
	if err == nil {
		user.Roles = roles
	}

	// 3. Populate Redis Cache with 10-Minute TTL
	if s.cache != nil {
		_ = s.cache.SetUser(ctx, user, 10*time.Minute)
	}

	return user, nil
}

func (s *userService) ListUsers(ctx context.Context, filter model.UserFilter) ([]model.User, int, error) {
	users, totalCount, err := s.userRepo.List(ctx, filter)
	if err != nil {
		return nil, 0, err
	}

	// Populate roles for each user in the page
	for i := range users {
		roles, err := s.roleRepo.GetRolesByUserID(ctx, users[i].ID)
		if err == nil {
			users[i].Roles = roles
		}
	}

	return users, totalCount, nil
}

func (s *userService) UpdateUser(ctx context.Context, id int, input UpdateUserInput) (*model.User, error) {
	user, err := s.userRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	// Validate name
	input.Name = strings.TrimSpace(input.Name)
	if input.Name != "" {
		user.Name = input.Name
	}

	// Validate email if changed
	if input.Email != "" {
		input.Email = strings.TrimSpace(strings.ToLower(input.Email))
		if _, err := mail.ParseAddress(input.Email); err != nil {
			return nil, ErrInvalidEmail
		}
		if input.Email != user.Email {
			existing, err := s.userRepo.GetByEmail(ctx, input.Email)
			if err == nil && existing != nil && existing.ID != id {
				return nil, ErrEmailAlreadyInUse
			}
			user.Email = input.Email
		}
	}

	// Validate department if provided
	if input.DepartmentID != nil {
		_, err := s.deptRepo.GetByID(ctx, *input.DepartmentID)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return nil, ErrDepartmentNotFound
			}
			return nil, err
		}
		user.DepartmentID = input.DepartmentID
	}

	// Validate status if provided
	if input.Status != "" {
		status := strings.ToLower(strings.TrimSpace(input.Status))
		if status != "active" && status != "inactive" {
			return nil, ErrInvalidStatus
		}
		user.Status = status
	}

	// Persist user update
	if err := s.userRepo.Update(ctx, user); err != nil {
		if errors.Is(err, repository.ErrDuplicateEntry) {
			return nil, ErrEmailAlreadyInUse
		}
		return nil, err
	}

	// Update role assignments if provided
	if input.RoleIDs != nil {
		// Verify roles exist
		for _, roleID := range input.RoleIDs {
			if _, err := s.roleRepo.GetByID(ctx, roleID); err != nil {
				return nil, ErrRoleNotFound
			}
		}

		// Sync roles: fetch current, remove unselected, add new
		currentRoles, err := s.roleRepo.GetRolesByUserID(ctx, id)
		if err == nil {
			for _, cr := range currentRoles {
				_ = s.roleRepo.RemoveRoleFromUser(ctx, id, cr.ID)
			}
		}
		for _, roleID := range input.RoleIDs {
			_ = s.roleRepo.AssignRoleToUser(ctx, id, roleID)
		}
	}

	// Invalidate Redis cache on update
	if s.cache != nil {
		_ = s.cache.DeleteUser(ctx, id)
	}

	return s.GetUser(ctx, id)
}

func (s *userService) DeleteUser(ctx context.Context, id int) error {
	err := s.userRepo.Delete(ctx, id)
	if err != nil && errors.Is(err, repository.ErrNotFound) {
		return ErrUserNotFound
	}

	// Invalidate Redis cache on delete
	if s.cache != nil {
		_ = s.cache.DeleteUser(ctx, id)
	}

	return err
}
