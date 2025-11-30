package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

// Claims — структура утверждений, которая включает стандартные утверждения и
// одно пользовательское UserID
type Claims struct {
	jwt.RegisteredClaims
	UserID int
}

const TokenExp = time.Hour * 3
const SekretKey = "supersecretkey"

func SetJWT(userID int, logger zap.SugaredLogger) (string, error) {
	tokenString, err := BuildJWTString(userID, logger)
	if err != nil {
		logger.Fatal(err)
	}
	// logger.Info(tokenString)
	return tokenString, err
}

// BuildJWTString создаёт токен и возвращает его в виде строки.
func BuildJWTString(userID int, logger zap.SugaredLogger) (string, error) {
	// создаём новый токен с алгоритмом подписи HS256 и утверждениями — Claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			// когда создан токен
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(TokenExp)),
		},
		// собственное утверждение
		UserID: userID,
	})
	// создаём строку токена
	tokenString, err := token.SignedString([]byte(SekretKey))
	if err != nil {
		return "", err
	}
	// возвращаем строку токена
	return tokenString, nil
}

// func getUserID(tokenString string) int {
// 	// создаём экземпляр структуры с утверждениями
// 	claims := &Claims{}
// 	// парсим из строки токена tokenString в структуру claims
// 	jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
// 		return []byte(SECRET_KEY), nil
// 	})
// 	// возвращаем ID пользователя в читаемом виде
// 	return claims.UserID
// }

func GetUserID(tokenString string, logger zap.SugaredLogger) int {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(SekretKey), nil
		})
	if err != nil {
		return -1
	}
	if !token.Valid {
		logger.Info("Token is not valid")
		return -1
	}
	logger.Info("Token is valid")
	return claims.UserID
}
