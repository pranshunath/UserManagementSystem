package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"usermanagementsystem/internal/model"
)

// DepartmentRepository defines the contract for department persistence.
type DepartmentRepository interface {
	Create(ctx context.Context, dept *model.Department) error
	GetByID(ctx context.Context, id int) (*model.Department, error)
	GetByName(ctx context.Context, name string) (*model.Department, error)
	List(ctx context.Context) ([]model.Department, error)
	Update(ctx context.Context, dept *model.Department) error
	Delete(ctx context.Context, id int) error
}

type postgresDepartmentRepository struct {
	db *sql.DB
}

// NewDepartmentRepository instantiates a PostgreSQL-backed DepartmentRepository.
func NewDepartmentRepository(db *sql.DB) DepartmentRepository {
	return &postgresDepartmentRepository{db: db}
}

func (r *postgresDepartmentRepository) Create(ctx context.Context, dept *model.Department) error {
	query := `
		INSERT INTO departments (name, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at;
	`
	now := time.Now()
	err := r.db.QueryRowContext(ctx, query, dept.Name, dept.Description, now, now).
		Scan(&dept.ID, &dept.CreatedAt, &dept.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create department: %w", err)
	}
	return nil
}

func (r *postgresDepartmentRepository) GetByID(ctx context.Context, id int) (*model.Department, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM departments
		WHERE id = $1;
	`
	var dept model.Department
	err := r.db.QueryRowContext(ctx, query, id).
		Scan(&dept.ID, &dept.Name, &dept.Description, &dept.CreatedAt, &dept.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get department by id %d: %w", id, err)
	}
	return &dept, nil
}

func (r *postgresDepartmentRepository) GetByName(ctx context.Context, name string) (*model.Department, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM departments
		WHERE name = $1;
	`
	var dept model.Department
	err := r.db.QueryRowContext(ctx, query, name).
		Scan(&dept.ID, &dept.Name, &dept.Description, &dept.CreatedAt, &dept.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get department by name %s: %w", name, err)
	}
	return &dept, nil
}

func (r *postgresDepartmentRepository) List(ctx context.Context) ([]model.Department, error) {
	query := `
		SELECT id, name, description, created_at, updated_at
		FROM departments
		ORDER BY id ASC;
	`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list departments: %w", err)
	}
	defer rows.Close()

	var depts []model.Department
	for rows.Next() {
		var d model.Department
		if err := rows.Scan(&d.ID, &d.Name, &d.Description, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan department row: %w", err)
		}
		depts = append(depts, d)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error during departments iteration: %w", err)
	}
	return depts, nil
}

func (r *postgresDepartmentRepository) Update(ctx context.Context, dept *model.Department) error {
	query := `
		UPDATE departments
		SET name = $1, description = $2, updated_at = $3
		WHERE id = $4
		RETURNING updated_at;
	`
	now := time.Now()
	err := r.db.QueryRowContext(ctx, query, dept.Name, dept.Description, now, dept.ID).
		Scan(&dept.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to update department %d: %w", dept.ID, err)
	}
	return nil
}

func (r *postgresDepartmentRepository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM departments WHERE id = $1;`
	res, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete department %d: %w", id, err)
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
