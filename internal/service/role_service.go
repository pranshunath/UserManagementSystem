package service

import (
	"context"
	"errors"
	"strings"

	"usermanagementsystem/internal/cache"
	"usermanagementsystem/internal/model"
	"usermanagementsystem/internal/repository"
)

// RoleService defines business logic operations for roles.
type RoleService interface {
	CreateRole(ctx context.Context, name, description string) (*model.Role, error)
	GetRole(ctx context.Context, id int) (*model.Role, error)
	ListRoles(ctx context.Context) ([]model.Role, error)
	UpdateRole(ctx context.Context, id int, name, description string) (*model.Role, error)
	DeleteRole(ctx context.Context, id int) error
}

type roleService struct {
	roleRepo repository.RoleRepository
	userRepo repository.UserRepository
	cache    cache.UserCache
}

// NewRoleService creates a new RoleService instance with injected repository and optional userRepo/cache.
func NewRoleService(roleRepo repository.RoleRepository, extras ...any) RoleService {
	s := &roleService{roleRepo: roleRepo}
	for _, extra := range extras {
		switch e := extra.(type) {
		case repository.UserRepository:
			s.userRepo = e
		case cache.UserCache:
			s.cache = e
		}
	}
	return s
}

func (s *roleService) CreateRole(ctx context.Context, name, description string) (*model.Role, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyName
	}

	existing, err := s.roleRepo.GetByName(ctx, name)
	if err == nil && existing != nil {
		return nil, repository.ErrDuplicateEntry
	}

	role := &model.Role{
		Name:        name,
		Description: strings.TrimSpace(description),
	}

	if err := s.roleRepo.Create(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

func (s *roleService) GetRole(ctx context.Context, id int) (*model.Role, error) {
	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}
	return role, nil
}

func (s *roleService) ListRoles(ctx context.Context) ([]model.Role, error) {
	return s.roleRepo.List(ctx)
}

func (s *roleService) UpdateRole(ctx context.Context, id int, name, description string) (*model.Role, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyName
	}

	role, err := s.roleRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrRoleNotFound
		}
		return nil, err
	}

	role.Name = name
	role.Description = strings.TrimSpace(description)

	if err := s.roleRepo.Update(ctx, role); err != nil {
		return nil, err
	}

	return role, nil
}

func (s *roleService) DeleteRole(ctx context.Context, id int) error {
	// Invalidate cache for any users currently assigned this role
	if s.userRepo != nil && s.cache != nil {
		users, _, err := s.userRepo.List(ctx, model.UserFilter{RoleID: &id, Limit: 1000})
		if err == nil {
			for _, u := range users {
				_ = s.cache.DeleteUser(ctx, u.ID)
			}
		}
	}

	err := s.roleRepo.Delete(ctx, id)
	if err != nil && errors.Is(err, repository.ErrNotFound) {
		return ErrRoleNotFound
	}
	return err
}
