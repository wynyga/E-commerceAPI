// Komponen ini bertanggung jawab untuk semua interaksi dengan database (query SQL)
package user

import (
	"database/sql"
	"fmt"
)

type Repository interface {
	CreateUser(user User) (int, error)
	GetUserByEmail(email string) (User, error)
}

type repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return &repository{db: db}
}

// CreatreUser menyimpan user baru ke database
func (r *repository) CreateUser(user User) (int, error) {
	query := "INSERT INTO users (email, password) VALUES (?, ?)"
	result, err := r.db.Exec(query, user.Email, user.Password)
	if err != nil {
		return 0, fmt.Errorf("failed to create user: %v", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to retrieve last insert id: %v", err)
	}
	return int(id), nil
}

// GetUserByEmail mencari pengguna di database berdasarkan email
func (r *repository) GetUserByEmail(email string) (User, error) {
	var user User
	query := "SELECT id, email, password, created_at, updated_at FROM users WHERE email = ?"

	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return User{}, fmt.Errorf("user not found")
		}
		return User{}, fmt.Errorf("failed to get user by email: %v", err)
	}
	return user, nil
}
