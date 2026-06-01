package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/avantifellows/nex-gen-cms/internal/models"
)

var ErrUserNotFound = errors.New("cms user not found")

type CmsUserRepo struct {
	db *sql.DB
}

func NewCmsUserRepo(db *sql.DB) *CmsUserRepo {
	return &CmsUserRepo{db: db}
}

const selectColumns = `id, email, role, full_name, is_active, last_login_at, inserted_at, updated_at`

func scanUser(row interface{ Scan(...any) error }) (*models.CmsUser, error) {
	var u models.CmsUser
	if err := row.Scan(&u.ID, &u.Email, &u.Role, &u.FullName, &u.IsActive, &u.LastLoginAt, &u.InsertedAt, &u.UpdatedAt); err != nil {
		return nil, err
	}
	return &u, nil
}

// GetByEmail returns the user row for the given email (case-insensitive) or ErrUserNotFound.
func (r *CmsUserRepo) GetByEmail(ctx context.Context, email string) (*models.CmsUser, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+selectColumns+` FROM cms_user_permission WHERE LOWER(email) = LOWER($1)`, email)
	u, err := scanUser(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrUserNotFound
	}
	return u, err
}

// List returns all users ordered by role (admin → editor → viewer) then email.
func (r *CmsUserRepo) List(ctx context.Context) ([]*models.CmsUser, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+selectColumns+` FROM cms_user_permission
		 ORDER BY CASE role WHEN 'admin' THEN 1 WHEN 'editor' THEN 2 ELSE 3 END, LOWER(email)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*models.CmsUser
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// Create inserts a new user. Returns the inserted ID.
func (r *CmsUserRepo) Create(ctx context.Context, email, role string, fullName *string) (int64, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return 0, fmt.Errorf("email is required")
	}

	var id int64
	err := r.db.QueryRowContext(ctx,
		`INSERT INTO cms_user_permission (email, role, full_name, is_active, inserted_at, updated_at)
		 VALUES ($1, $2, $3, true, NOW(), NOW())
		 RETURNING id`,
		email, role, fullName).Scan(&id)
	return id, err
}

// SetActive toggles is_active (soft delete/restore).
func (r *CmsUserRepo) SetActive(ctx context.Context, id int64, active bool) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE cms_user_permission SET is_active = $1, updated_at = NOW() WHERE id = $2`, active, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrUserNotFound
	}
	return nil
}

// UpdateLastLogin stamps last_login_at = NOW() for the given user.
func (r *CmsUserRepo) UpdateLastLogin(ctx context.Context, id int64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE cms_user_permission SET last_login_at = NOW(), updated_at = NOW() WHERE id = $1`, id)
	return err
}

// UpdateRole changes a user's role.
func (r *CmsUserRepo) UpdateRole(ctx context.Context, id int64, role string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE cms_user_permission SET role = $1, updated_at = NOW() WHERE id = $2`, role, id)
	return err
}
