package service

import (
	"context"
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"usermanagementsystem/internal/auth"
	"usermanagementsystem/internal/model"
	"usermanagementsystem/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrInactiveUser       = errors.New("user account is inactive")
)

// AuthResult represents the successful output of user authentication.
type AuthResult struct {
	Token string
	User  *model.User
	Roles []string
}

// AuthService defines business operations for authentication.
type AuthService interface {
	Login(ctx context.Context, email, password string) (*AuthResult, error)
}

type authService struct {
	userRepo     repository.UserRepository
	roleRepo     repository.RoleRepository
	tokenManager *auth.TokenManager
}

// NewAuthService creates a new AuthService instance.
func NewAuthService(
	userRepo repository.UserRepository,
	roleRepo repository.RoleRepository,
	tokenManager *auth.TokenManager,
) AuthService {
	return &authService{
		userRepo:     userRepo,
		roleRepo:     roleRepo,
		tokenManager: tokenManager,
	}
}

func (s *authService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || password == "" {
		return nil, ErrInvalidCredentials
	}

	// 1. Fetch user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrInvalidCredentials // Prevent user enumeration
		}
		return nil, err
	}

	// 2. Check account status
	if user.Status != "active" {
		return nil, ErrInactiveUser
	}

	// 3. Verify password hash using constant-time comparison
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// 4. Fetch assigned roles
	roles, err := s.roleRepo.GetRolesByUserID(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	user.Roles = roles

	roleNames := make([]string, len(roles))
	for i, r := range roles {
		roleNames[i] = r.Name
	}

	// 5. Generate signed JWT token
	token, err := s.tokenManager.GenerateToken(user.ID, user.Email, roleNames)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		Token: token,
		User:  user,
		Roles: roleNames,
	}, nil
}
