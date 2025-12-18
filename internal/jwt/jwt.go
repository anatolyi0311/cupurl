package jwt

import (
	"errors"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/anatolyi0311/cupurl/internal/model"
	"github.com/golang-jwt/jwt/v4"
)

var userIDCounter atomic.Int64

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

func createJWT(res http.ResponseWriter, secretKey string) error {
	userID := int(userIDCounter.Add(1))

	expirationTime := time.Now().Add(24 * time.Hour) // Токен на 24 часа

	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenStr, err := token.SignedString([]byte(secretKey)) // Подписание токена секретным ключом
	if err != nil {
		return fmt.Errorf("failed to create token %w", err)
	}

	http.SetCookie(res, &http.Cookie{
		Name:    "jwt_token",
		Value:   tokenStr,
		Expires: expirationTime,
		// Path:     "/",
	})
	fmt.Println("...createJWT.tokenStr", tokenStr, "userID", userID)
	return nil
}

func validateJWT(res http.ResponseWriter, req *http.Request, secretKey string) error {
	cookie, err := req.Cookie("jwt_token")
	if err != nil {
		return createJWT(res, secretKey)
	}

	tokenStr := cookie.Value
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		},
	)

	id := claims.UserID
	if id < 1 {
		return model.ErrEmptyUserID
	}
	if err != nil {
		return fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return fmt.Errorf("invalid token")
	}

	return nil
}

func GetUserID(res http.ResponseWriter, req *http.Request, secretKey string) (int, error) {
	cookie, err := req.Cookie("jwt_token")
	if err != nil {
		return 0, err
	}
	tokenStr := cookie.Value
	claims := &Claims{}

	_, err = jwt.ParseWithClaims(
		tokenStr,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secretKey), nil
		},
	)
	if err != nil {
		return 0, err
	}

	id := claims.UserID
	if id < 1 {
		return 0, model.ErrEmptyUserID
	}
	return id, nil
}

func Cookies(next http.Handler, secretKey string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := validateJWT(w, r, secretKey); err != nil {
			if errors.Is(err, model.ErrEmptyUserID) {
				http.Error(w, "invalid JWT", http.StatusUnauthorized)
				return
			}
			http.Error(w, "invalid JWT", http.StatusBadRequest)
			return
		}

		next.ServeHTTP(w, r)
	})
}
