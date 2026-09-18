package model

import "time"

// User represents the central user entity.
type User struct {
	ID           int        `json:"id"`
	Name         string     `json:"name"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // Security: Never serialize password_hash to JSON!
	DepartmentID *int       `json:"department_id,omitempty"`
	Status       string     `json:"status"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at,omitempty"`

	// Relational associations (populated when joined)
	Department *Department `json:"department,omitempty"`
	Roles      []Role      `json:"roles,omitempty"`
}

// UserFilter represents query parameters for filtering and pagination.
type UserFilter struct {
	Search       string
	DepartmentID *int
	RoleID       *int
	Status       string
	SortBy       string
	SortOrder    string
	Limit        int
	Offset       int
}

