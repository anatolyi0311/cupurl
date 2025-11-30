package jwt

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/anatolyi0311/cupurl/internal/model"
	"github.com/golang-jwt/jwt/v4"
)

const (
	secretKey = "super_secret_key_for_shortener"
)

var userIDCounter int

type Claims struct {
	UserID int `json:"user_id"`
	jwt.RegisteredClaims
}

func createJWT(res http.ResponseWriter) error {
	userIDCounter++
	userID := userIDCounter

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
	return nil
}

func validateJWT(res http.ResponseWriter, req *http.Request) error {
	cookie, err := req.Cookie("jwt_token")
	if err != nil {
		return createJWT(res)
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

func GetUserID(req *http.Request) (int, error) {
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

func Cookies(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := validateJWT(w, r); err != nil {
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
