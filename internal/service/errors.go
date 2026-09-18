package service

import "errors"

var (
	// Domain validation errors
	ErrEmptyName          = errors.New("name cannot be empty")
	ErrInvalidEmail       = errors.New("invalid email address format")
	ErrPasswordTooWeak    = errors.New("password must be at least 8 characters long")
	ErrEmailAlreadyInUse  = errors.New("email is already registered")
	ErrDepartmentNotFound = errors.New("referenced department does not exist")
	ErrRoleNotFound       = errors.New("referenced role does not exist")
	ErrInvalidStatus      = errors.New("status must be either 'active' or 'inactive'")
	ErrUserNotFound       = errors.New("user not found")
)
