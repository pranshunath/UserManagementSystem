package service

import (
	"context"
	"errors"
	"strings"

	"usermanagementsystem/internal/cache"
	"usermanagementsystem/internal/model"
	"usermanagementsystem/internal/repository"
)

// DepartmentService defines business logic operations for departments.
type DepartmentService interface {
	CreateDepartment(ctx context.Context, name, description string) (*model.Department, error)
	GetDepartment(ctx context.Context, id int) (*model.Department, error)
	ListDepartments(ctx context.Context) ([]model.Department, error)
	UpdateDepartment(ctx context.Context, id int, name, description string) (*model.Department, error)
	DeleteDepartment(ctx context.Context, id int) error
}

type departmentService struct {
	deptRepo repository.DepartmentRepository
	userRepo repository.UserRepository
	cache    cache.UserCache
}

// NewDepartmentService creates a new DepartmentService instance with injected repository and optional userRepo/cache.
func NewDepartmentService(deptRepo repository.DepartmentRepository, extras ...any) DepartmentService {
	s := &departmentService{deptRepo: deptRepo}
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

func (s *departmentService) CreateDepartment(ctx context.Context, name, description string) (*model.Department, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyName
	}

	// Check for existing department with same name
	existing, err := s.deptRepo.GetByName(ctx, name)
	if err == nil && existing != nil {
		return nil, repository.ErrDuplicateEntry
	}

	dept := &model.Department{
		Name:        name,
		Description: strings.TrimSpace(description),
	}

	if err := s.deptRepo.Create(ctx, dept); err != nil {
		return nil, err
	}

	return dept, nil
}

func (s *departmentService) GetDepartment(ctx context.Context, id int) (*model.Department, error) {
	dept, err := s.deptRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}
	return dept, nil
}

func (s *departmentService) ListDepartments(ctx context.Context) ([]model.Department, error) {
	return s.deptRepo.List(ctx)
}

func (s *departmentService) UpdateDepartment(ctx context.Context, id int, name, description string) (*model.Department, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, ErrEmptyName
	}

	dept, err := s.deptRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrDepartmentNotFound
		}
		return nil, err
	}

	dept.Name = name
	dept.Description = strings.TrimSpace(description)

	if err := s.deptRepo.Update(ctx, dept); err != nil {
		return nil, err
	}

	return dept, nil
}

func (s *departmentService) DeleteDepartment(ctx context.Context, id int) error {
	// Invalidate cache for any users currently assigned to this department
	if s.userRepo != nil && s.cache != nil {
		users, _, err := s.userRepo.List(ctx, model.UserFilter{DepartmentID: &id, Limit: 1000})
		if err == nil {
			for _, u := range users {
				_ = s.cache.DeleteUser(ctx, u.ID)
			}
		}
	}

	err := s.deptRepo.Delete(ctx, id)
	if err != nil && errors.Is(err, repository.ErrNotFound) {
		return ErrDepartmentNotFound
	}
	return err
}
