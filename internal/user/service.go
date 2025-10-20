// Lapisan ini menangani logika bisnis, seperti hashing password sebelum disimpan.
package user

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// Service mendefinisikan interface untuk logika bisnis user
type Service interface {
	RegisterUser(payload RegisterPayload) (User, error)
}

type service struct {
	repo Repository
}

// NewService membuat instance service baru
func NewService(repo Repository) Service {
	return &service{repo: repo}
}

// RegisterUser menghash password dan menyimpan pengguna baru
func (s *service) RegisterUser(payload RegisterPayload) (User, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(payload.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("could not hash password: %v", err)
	}

	// Buat user baru
	newUser := User{
		Email:    payload.Email,
		Password: string(hashedPassword),
	}

	// Simpan ke database
	id, err := s.repo.CreateUser(newUser)
	if err != nil {
		return User{}, err
	}

	newUser.ID = id
	newUser.Password = "" // Kosongkan password sebelum dikirim kembali

	return newUser, nil
}
