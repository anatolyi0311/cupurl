// Package url provides HTTP request handlers and middleware for the URL shortening application.
package url

import (
	"errors"
	"net/http"

	"github.com/anatolyi0311/cupurl/internal/audit"
	"github.com/anatolyi0311/cupurl/internal/auth"
	"github.com/anatolyi0311/cupurl/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GetOriginalURL retrieves the original URL from a shortened URL ID.
// The shortened URL ID is expected as a URL parameter.
// Redirects to the original URL using HTTP 307 Temporary Redirect.
// Returns HTTP status 410 Gone if the URL is marked as deleted,
// or HTTP status 400 Bad Request for other errors.
func (h Handlers) GetOriginalURL(c *gin.Context) {
	ctx := c.Request.Context()
	shortURL := c.Param("id")
	originURL, err := h.service.GetOriginalURL(ctx, shortURL)
	if err != nil {
		if errors.Is(err, models.ErrURLDeleted) {
			c.Status(http.StatusGone)
			return
		}
		c.Status(http.StatusBadRequest)
		return
	}

	if h.audit != nil {
		tokenString, err := c.Cookie("user_token")
		if err != nil {
			logrus.Error(err)
			return
		}
		userID, err := auth.GetUserID(tokenString, h.SecretKey)
		if err != nil {
			logrus.Error(err)
			return
		}
		h.sendEvent(audit.CreateEvent(int(userID), audit.Follow, originURL))
	}

	c.Header("Location", originURL)
	c.Status(http.StatusTemporaryRedirect)
}
