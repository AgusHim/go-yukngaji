package utils

import (
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(hash), nil
}

func CheckPassword(password string, hashPassword string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashPassword), []byte(password))
}

func GenerateJWT(userID string) (string, error) {

	claim := jwt.MapClaims{}
	claim["user_id"] = userID

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claim)

	secretKey := []byte(os.Getenv("JWT_SECRET_KEY"))
	signedToken, err := token.SignedString(secretKey)

	if err != nil {
		return "", err
	}

	return signedToken, nil

}

func ValidateJWT(encodeToken string) (*jwt.Token, error) {
	secretKey := os.Getenv("JWT_SECRET_KEY")
	token, err := jwt.Parse(encodeToken, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	return token, nil
}
