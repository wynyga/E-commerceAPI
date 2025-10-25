package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func GenerateToken(userID int) (string, error) {
	secretKey := os.Getenv("JWT_SECRET_KEY")
	if secretKey == "" {
		return "", fmt.Errorf("JWT secret key is not set in environment variables")
	}

	//Buat claim token di sini menggunakan userID dan secretKey
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(time.Hour * 24).Unix(), // Token berlaku selama 24 jam
		"iat":     time.Now().Unix(),
	}

	//Buat token baru dengan klaim yang telah ditentukan
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	//Tandatangani token dengan secret key
	signedToken, err := token.SignedString([]byte(secretKey))
	if err != nil {
		return "", fmt.Errorf("tidak bisa sign token: %v", err)
	}
	return signedToken, nil
}
