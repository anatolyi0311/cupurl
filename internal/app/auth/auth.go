// Package auth provides functions for handling authentication, JWT token creation,
// and validation.
package auth

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
)

// Claims — claims structure that includes standard claims and UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID int64
}

// const for generate token
const (
	TokenExp  = time.Hour * 3
	SecretKey = "SnJSkf123jlLKNfsNln"
)

// BuildJWTString creates a token with the HS256 signature algorithm and Claims statements and returns it as a string.
func BuildJWTString(secretKey string) (string, error) {
	if secretKey == "" {
		secretKey = SecretKey
	}
	userID := generateUniqueID()
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

// generate Unique ID generate a unique UserID from 0 to 999999
func generateUniqueID() int64 {
	id, err := generateRandomInt(100)
		if err != nil {
		logrus.Fatal(err)
	}
	logrus.Infof("Generated user id is: %v", id)
	return id
}

func generateRandomInt(max int64) (int64, error) {
	// big.NewInt(max) creates a new big.Int with value max
	bi, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return bi.Int64(), nil
}

// GetUserID we check the validity of the token and if it is valid, then we get and return the UserID from it
func GetUserID(tokenString, secretKey string) (int64, error) {
	if secretKey == "" {
		secretKey = SecretKey
	}
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
		// if t == nil {
		// }
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signed method: %v", t.Header["alg"])
		}
		return []byte(secretKey), nil
	})
	if err != nil {
		logrus.Error(err)
		return 0, err
	}
	if !token.Valid {
		err = fmt.Errorf("token is not valid")
		logrus.Error(err)
		return 0, err
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
		// if t == nil {
		// }
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
