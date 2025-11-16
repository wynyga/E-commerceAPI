// Komponen ini menangani logika bisnis, seperti hashing password sebelum disimpan.
package user

import (
	"fmt"

	"auth-service/internal/auth"

	"golang.org/x/crypto/bcrypt"
)

// Service mendefinisikan interface untuk logika bisnis user
type Service interface {
	RegisterUser(payload RegisterPayload) (User, error)
	LoginUser(payload LoginPayLoad) (string, error)
	GetUserProfile(userID int) (User, error)
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

// LoginUser memverifikasi kredensial pengguna dan mengembalikan JWT
func (s *service) LoginUser(payload LoginPayLoad) (string, error) {
	// 1. Cari user berdasarkan email
	user, err := s.repo.GetUserByEmail(payload.Email)
	if err != nil {
		return "", fmt.Errorf("invalid credentials") // Pesan error generik
	}

	// 2. Bandingkan password yang di-hash dengan password dari payload
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(payload.Password))
	if err != nil {
		// Jika error, berarti password tidak cocok
		return "", fmt.Errorf("invalid credentials")
	}

	// 3. Jika cocok, buat token JWT
	token, err := auth.GenerateToken(user.ID)
	if err != nil {
		return "", fmt.Errorf("could not generate token: %v", err)
	}

	return token, nil
}

// GetUserProfile mengambil data profil pengguna berdasarkan ID
func (s *service) GetUserProfile(userID int) (User, error) {
	user, err := s.repo.GetUserByID(userID)
	if err != nil {
		return User{}, err
	}
	return user, nil
}
