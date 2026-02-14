// Package auth provides functions for handling authentication, JWT token creation,
// and validation.
package auth

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Claims is a structure that includes standard JWT claims and UserID.
type Claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID
}

// const for generate token
const (
	// TokenExp defines the expiration duration for JWT tokens.
	TokenExp = time.Hour * 3
	// SecretKey is the secret key used for signing JWT tokens.
	SecretKey = "SnJSkf123jlLKNfsNln"
)

// BuildJWTString creates a token with the HS256 signature algorithm and Claims statements and returns it as a string.
func BuildJWTString(secretKey string) (string, error) {
	if secretKey == "" {
		secretKey = SecretKey
	}
	userID := GenerateUniqueID()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		UserID: userID,
	})
	// создаём строку токена
	tokenString, err := token.SignedString([]byte(secretKey))
	if err != nil {
		logrus.Error(err)
		return "", err
	}
	return tokenString, nil
}

// GenerateUniqueID генерирует UUID при помощи библиотеки golang.org/x/crypto/bcrypt
func GenerateUniqueID() uuid.UUID {
	return uuid.New()
}

// GetUserID we check the validity of the token and if it is valid, then we get and return the UserID from it
func GetUserID(tokenString, secretKey string) (uuid.UUID, error) {
	if secretKey == "" {
		secretKey = SecretKey
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signed method: %v", t.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		logrus.Error(err)
		return uuid.Nil, err
	}
	if !token.Valid {
		err = fmt.Errorf("token is not valid")
		logrus.Error(err)
		return uuid.Nil, err
	}
	logrus.Infof("Token is valid, userID: %v", claims.UserID)
	return claims.UserID, nil
}

// IsValidToken method to check the token for validity, we return bool
func IsValidToken(tokenString, secretKey string) bool {
	if secretKey == "" {
		secretKey = SecretKey
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signed method: %v", t.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		logrus.Error(err)
		return false
	}
	if !token.Valid {
		err = fmt.Errorf("token is not valid")
		logrus.Error(err)
		return false
	}
	return true
}
