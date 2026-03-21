// Package url provides HTTP request handlers and middleware for the URL shortening application.
package url

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/anatolyi0311/cupurl/internal/audit"
	"github.com/anatolyi0311/cupurl/internal/auth"
	"github.com/anatolyi0311/cupurl/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// GetShortURL converts a long URL to its shortened version.
// It reads the raw URL from the request body.
// Returns the shortened URL on success with HTTP status 201 Created.
// On failure, returns HTTP status 400 Bad Request for invalid input or URL format,
// or HTTP status 409 Conflict if the URL is already shortened.
func (h Handlers) GetShortURL(c *gin.Context) {
	ctx := c.Request.Context()
	link, err := c.GetRawData()
	if err != nil {
		c.Status(http.StatusBadRequest)
		return
	}

	linkString := string(link)
	parsedLinc, err := url.Parse(linkString)
	if err != nil || parsedLinc.Scheme == "" || parsedLinc.Host == "" {
		c.Status(http.StatusBadRequest)
		return
	}
	shortURL, err := h.service.GetShortURL(ctx, linkString)
	if err != nil {
		if errors.Is(err, models.ErrURLFound) {
			c.String(http.StatusConflict, shortURL)
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
		h.sendEvent(audit.CreateEvent(int(userID), audit.Shorten, string(link)))
	}

	c.String(http.StatusCreated, shortURL)
}
