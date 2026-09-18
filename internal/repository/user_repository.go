package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"usermanagementsystem/internal/model"
)

// UserRepository defines the contract for user persistence.
type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	GetByID(ctx context.Context, id int) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	List(ctx context.Context, filter model.UserFilter) ([]model.User, int, error)
	Update(ctx context.Context, user *model.User) error
	Delete(ctx context.Context, id int) error // Soft delete
}

type postgresUserRepository struct {
	db *sql.DB
}

// NewUserRepository instantiates a PostgreSQL-backed UserRepository.
func NewUserRepository(db *sql.DB) UserRepository {
	return &postgresUserRepository{db: db}
}

func (r *postgresUserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (name, email, password_hash, department_id, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at;
	`
	now := time.Now()
	if user.Status == "" {
		user.Status = "active"
	}

	err := r.db.QueryRowContext(
		ctx, query,
		user.Name, user.Email, user.PasswordHash, user.DepartmentID, user.Status, now, now,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if strings.Contains(err.Error(), "duplicate key value") || strings.Contains(err.Error(), "users_email_key") {
			return ErrDuplicateEntry
		}
		return fmt.Errorf("failed to create user: %w", err)
	}
	return nil
}

func (r *postgresUserRepository) GetByID(ctx context.Context, id int) (*model.User, error) {
	query := `
		SELECT u.id, u.name, u.email, u.password_hash, u.department_id, u.status, u.created_at, u.updated_at, u.deleted_at,
		       d.id, d.name, d.description
		FROM users u
		LEFT JOIN departments d ON u.department_id = d.id
		WHERE u.id = $1 AND u.deleted_at IS NULL;
	`
	var user model.User
	var deptID sql.NullInt64
	var deptName, deptDesc sql.NullString

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.DepartmentID, &user.Status,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		&deptID, &deptName, &deptDesc,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by id %d: %w", id, err)
	}

	// Populate joined department if present
	if deptID.Valid {
		user.Department = &model.Department{
			ID:          int(deptID.Int64),
			Name:        deptName.String,
			Description: deptDesc.String,
		}
	}

	return &user, nil
}

func (r *postgresUserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT u.id, u.name, u.email, u.password_hash, u.department_id, u.status, u.created_at, u.updated_at, u.deleted_at,
		       d.id, d.name, d.description
		FROM users u
		LEFT JOIN departments d ON u.department_id = d.id
		WHERE u.email = $1 AND u.deleted_at IS NULL;
	`
	var user model.User
	var deptID sql.NullInt64
	var deptName, deptDesc sql.NullString

	err := r.db.QueryRowContext(ctx, query, email).Scan(
		&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.DepartmentID, &user.Status,
		&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
		&deptID, &deptName, &deptDesc,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get user by email %s: %w", email, err)
	}

	if deptID.Valid {
		user.Department = &model.Department{
			ID:          int(deptID.Int64),
			Name:        deptName.String,
			Description: deptDesc.String,
		}
	}

	return &user, nil
}

// Whitelist mapping of allowed sort fields to prevent SQL injection
var allowedUserSortColumns = map[string]string{
	"id":         "u.id",
	"name":       "u.name",
	"email":      "u.email",
	"status":     "u.status",
	"created_at": "u.created_at",
	"updated_at": "u.updated_at",
	"department": "d.name",
}

func (r *postgresUserRepository) List(ctx context.Context, filter model.UserFilter) ([]model.User, int, error) {
	// Base query with soft-delete filter
	whereClauses := []string{"u.deleted_at IS NULL"}
	args := []interface{}{}
	argIndex := 1

	if searchTrim := strings.TrimSpace(filter.Search); searchTrim != "" {
		tokens := strings.Fields(searchTrim)
		for _, token := range tokens {
			whereClauses = append(whereClauses, fmt.Sprintf("(u.name ILIKE $%d OR u.email ILIKE $%d)", argIndex, argIndex))
			args = append(args, "%"+token+"%")
			argIndex++
		}
	}

	if filter.DepartmentID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("u.department_id = $%d", argIndex))
		args = append(args, *filter.DepartmentID)
		argIndex++
	}

	if filter.Status != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("u.status = $%d", argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.RoleID != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("EXISTS (SELECT 1 FROM user_roles ur WHERE ur.user_id = u.id AND ur.role_id = $%d)", argIndex))
		args = append(args, *filter.RoleID)
		argIndex++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// 1. Get total count for pagination metadata
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users u WHERE %s", whereSQL)
	var totalCount int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		return nil, 0, fmt.Errorf("failed to count users: %w", err)
	}

	// Dynamic Sort Column & Direction (Strictly Whitelisted against SQL Injection)
	sortCol := "u.id"
	if col, ok := allowedUserSortColumns[strings.ToLower(strings.TrimSpace(filter.SortBy))]; ok {
		sortCol = col
	}

	sortOrder := "ASC"
	if strings.ToUpper(strings.TrimSpace(filter.SortOrder)) == "DESC" {
		sortOrder = "DESC"
	}

	// Tie-breaker by u.id for deterministic pagination
	orderBySQL := fmt.Sprintf("%s %s", sortCol, sortOrder)
	if sortCol != "u.id" {
		orderBySQL += ", u.id ASC"
	}

	// 2. Query paginated rows
	selectQuery := fmt.Sprintf(`
		SELECT u.id, u.name, u.email, u.password_hash, u.department_id, u.status, u.created_at, u.updated_at, u.deleted_at,
		       d.id, d.name, d.description
		FROM users u
		LEFT JOIN departments d ON u.department_id = d.id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d;
	`, whereSQL, orderBySQL, argIndex, argIndex+1)

	limit := filter.Limit
	if limit <= 0 {
		limit = 10
	} else if limit > 100 {
		limit = 100
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var user model.User
		var deptID sql.NullInt64
		var deptName, deptDesc sql.NullString

		err := rows.Scan(
			&user.ID, &user.Name, &user.Email, &user.PasswordHash, &user.DepartmentID, &user.Status,
			&user.CreatedAt, &user.UpdatedAt, &user.DeletedAt,
			&deptID, &deptName, &deptDesc,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user row: %w", err)
		}

		if deptID.Valid {
			user.Department = &model.Department{
				ID:          int(deptID.Int64),
				Name:        deptName.String,
				Description: deptDesc.String,
			}
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("error during users iteration: %w", err)
	}

	return users, totalCount, nil
}

func (r *postgresUserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET name = $1, email = $2, department_id = $3, status = $4, updated_at = $5
		WHERE id = $6 AND deleted_at IS NULL
		RETURNING updated_at;
	`
	now := time.Now()
	err := r.db.QueryRowContext(
		ctx, query,
		user.Name, user.Email, user.DepartmentID, user.Status, now, user.ID,
	).Scan(&user.UpdatedAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if strings.Contains(err.Error(), "duplicate key value") || strings.Contains(err.Error(), "users_email_key") {
			return ErrDuplicateEntry
		}
		return fmt.Errorf("failed to update user %d: %w", user.ID, err)
	}
	return nil
}

func (r *postgresUserRepository) Delete(ctx context.Context, id int) error {
	// Soft delete: record the timestamp when the user was marked deleted
	query := `
		UPDATE users
		SET deleted_at = $1, status = 'inactive'
		WHERE id = $2 AND deleted_at IS NULL;
	`
	now := time.Now()
	res, err := r.db.ExecContext(ctx, query, now, id)
	if err != nil {
		return fmt.Errorf("failed to delete user %d: %w", id, err)
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
