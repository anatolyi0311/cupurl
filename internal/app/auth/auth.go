package auth

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/sirupsen/logrus"
)

// Claims — claims structure that includes standard claims and UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID uint32
}

const (
	TokenExp  = time.Hour * 3
	SecretKey = "SnJSkf123jlLKNfsNln"
)

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

func generateUniqueID() uint32 {
	rand.NewSource(time.Now().UnixNano())
	id := uint32(rand.Intn(1000000))
	logrus.Infof("Generated user id is: %v", id)
	return id
}

func GetUserID(tokenString, secretKey string) (uint32, error) {
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
