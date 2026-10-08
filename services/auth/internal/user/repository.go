package user

import (
	"context"
	"database/sql"
	"errors"

	"github.com/wynyga/ecommerce/shared/apperr"
	"github.com/wynyga/ecommerce/shared/db"
)

type repository struct {
	db *sql.DB
}

func NewRepository(conn *sql.DB) Repository {
	return &repository{db: conn}
}

const columns = `id, email, password, role, created_at`

func (r *repository) scan(row *sql.Row) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.Role, &u.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return User{}, apperr.NotFound("user not found")
	}
	return u, err
}

func (r *repository) FindByEmail(ctx context.Context, email string) (User, error) {
	return r.scan(r.db.QueryRowContext(ctx, `SELECT `+columns+` FROM users WHERE email = ?`, email))
}

func (r *repository) FindByID(ctx context.Context, id uint) (User, error) {
	return r.scan(r.db.QueryRowContext(ctx, `SELECT `+columns+` FROM users WHERE id = ?`, id))
}

func (r *repository) Create(ctx context.Context, u User) (uint, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO users (email, password, role) VALUES (?, ?, ?)`, u.Email, u.PasswordHash, u.Role)
	if err != nil {
		if db.IsDuplicate(err) {
			return 0, apperr.Conflict("email is already registered")
		}
		return 0, err
	}
	id, err := res.LastInsertId()
	return uint(id), err
}
