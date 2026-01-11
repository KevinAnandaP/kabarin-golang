package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

// Claims represents JWT claims
type Claims struct {
	UserID   string `json:"userId"`
	Email    string `json:"email"`
	UniqueID string `json:"uniqueId"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID, email, uniqueID string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour) // 24 hours

	claims := &Claims{
		UserID:   userID,
		Email:    email,
		UniqueID: uniqueID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// ValidateToken validates and parses a JWT token
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, jwt.ErrSignatureInvalid
	}

	return claims, nil
}
