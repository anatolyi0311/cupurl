package handler

import (
	"context"
	"net/http"

	"github.com/anatolyi0311/cupurl/internal/auth"
	"github.com/anatolyi0311/cupurl/internal/model"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// MiddlewareAuthPublic provides authentication middleware for public routes.
// It manages user tokens, generating new tokens if necessary, and adds user ID to the context.
// This middleware is useful for routes that require user identification but not strict authentication.
func MiddlewareAuthPublic() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		var err error
		var userID uint32

		tokenString, err = c.Cookie("user_token")
		// если токен не найден в куке, то генерируем новый и добавляем его в куки
		if err != nil || !auth.IsValidToken(tokenString) {
			logrus.Info("Cookie not found or token in cookie not found")
			tokenString, err = auth.BuildJWTString()
			if err != nil {
				logrus.Errorf("error generating token: %v", err)
				c.AbortWithStatus(http.StatusInternalServerError)
			}
			c.SetCookie("user_token", tokenString, 0, "/", "", false, true)
		}
		userID, err = auth.GetUserID(tokenString)
		if err != nil {
			logrus.Error(err)
			return
		}

		ctx := context.WithValue(c.Request.Context(), model.UserIDKey, userID)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
