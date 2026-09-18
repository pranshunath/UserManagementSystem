package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"usermanagementsystem/internal/model"
)

// RoleRepository defines the contract for role persistence and role assignments.
type RoleRepository interface {
	Create(ctx context.Context, role *model.Role) error
	GetByID(ctx context.Context, id int) (*model.Role, error)
	GetByName(ctx context.Context, name string) (*model.Role, error)
	List(ctx context.Context) ([]model.Role, error)
	Update(ctx context.Context, role *model.Role) error
	Delete(ctx context.Context, id int) error

	// Many-to-many user-role assignment operations
	AssignRoleToUser(ctx context.Context, userID int, roleID int) error
	RemoveRoleFromUser(ctx context.Context, userID int, roleID int) error
	GetRolesByUserID(ctx context.Context, userID int) ([]model.Role, error)
}

type postgresRoleRepository struct {
	db *sql.DB
}

// NewRoleRepository instantiates a PostgreSQL-backed RoleRepository.
func NewRoleRepository(db *sql.DB) RoleRepository {
	return &postgresRoleRepository{db: db}
}

func (r *postgresRoleRepository) Create(ctx context.Context, role *model.Role) error {
	query := `
		INSERT INTO roles (name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at;
	`
	now := time.Now()
	err := r.db.QueryRowContext(ctx, query, role.Name, role.Description, now, now).
		Scan(&role.ID, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}
	return nil
}

func (r *postgresRoleRepository) GetByID(ctx context.Context, id int) (*model.Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE id = $1;
	`
	var role model.Role
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get role by id %d: %w", id, err)
	}
	return &role, nil
}

func (r *postgresRoleRepository) GetByName(ctx context.Context, name string) (*model.Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		WHERE name = $1;
	`
	var role model.Role
	err := r.db.QueryRowContext(ctx, query, name).
		Scan(&role.ID, &role.Name, &role.Description, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get role by name %s: %w", name, err)
	}
	return &role, nil
}

func (r *postgresRoleRepository) List(ctx context.Context) ([]model.Role, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM roles
		ORDER BY id ASC;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list roles: %w", err)
	}
	defer rows.Close()

	var roles []model.Role
	for rows.Next() {
		var rl model.Role
		if err := rows.Scan(&rl.ID, &rl.Name, &rl.Description, &rl.CreatedAt, &rl.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role row: %w", err)
		}
		roles = append(roles, rl)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during roles iteration: %w", err)
	}
	return roles, nil
}

func (r *postgresRoleRepository) Update(ctx context.Context, role *model.Role) error {
	query := `
		UPDATE roles
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
		RETURNING updated_at;
	`
	now := time.Now()
	err := r.db.QueryRowContext(ctx, query, role.Name, role.Description, now, role.ID).
		Scan(&role.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to update role %d: %w", role.ID, err)
	}
	return nil
}

func (r *postgresRoleRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM roles WHERE id = $1;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete role %d: %w", id, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *postgresRoleRepository) AssignRoleToUser(ctx context.Context, userID int, roleID int) error {
	query := `
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, role_id) DO NOTHING;
	`
	_, err := r.db.ExecContext(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to assign role %d to user %d: %w", roleID, userID, err)
	}
	return nil
}

func (r *postgresRoleRepository) RemoveRoleFromUser(ctx context.Context, userID int, roleID int) error {
	query := `DELETE FROM user_roles WHERE user_id = $1 AND role_id = $2;`
	_, err := r.db.ExecContext(ctx, query, userID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove role %d from user %d: %w", roleID, userID, err)
	}
	return nil
}

func (r *postgresRoleRepository) GetRolesByUserID(ctx context.Context, userID int) ([]model.Role, error) {
	query := `
		SELECT r.id, r.name, r.description, r.created_at, r.updated_at
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.id ASC;
	`
	rows, err := r.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get roles for user %d: %w", userID, err)
	}
	defer rows.Close()

	var roles []model.Role
	for rows.Next() {
		var rl model.Role
		if err := rows.Scan(&rl.ID, &rl.Name, &rl.Description, &rl.CreatedAt, &rl.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan role row for user: %w", err)
		}
		roles = append(roles, rl)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during user roles iteration: %w", err)
	}
	return roles, nil
}
