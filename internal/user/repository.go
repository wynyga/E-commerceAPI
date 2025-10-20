// Lapisan ini bertanggung jawab untuk semua interaksi dengan database (query SQL)
package user

import (
	"database/sql"
	"fmt"
)

type Repository interface {
	CreateUser(user User) (int, error)
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
