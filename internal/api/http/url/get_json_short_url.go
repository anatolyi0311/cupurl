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

// GetJSONShortURL converts a long URL to its shortened version using JSON input.
// Expects a JSON object with a 'URL' field in the request body.
// Returns a JSON object containing the shortened URL on success.
// Sends HTTP status 400 Bad Request for malformed JSON or invalid URL,
// or HTTP status 409 Conflict if the URL is already shortened.
func (h Handlers) GetJSONShortURL(c *gin.Context) {
	ctx := c.Request.Context()
	var dataURL URLProcessing
	if err := c.ShouldBindJSON(&dataURL); err != nil {
		c.Status(http.StatusBadRequest)
		return
	}
	result, err := h.service.GetShortURL(ctx, dataURL.URL)
	if err != nil {
		if errors.Is(err, models.ErrURLFound) {
			c.JSON(http.StatusConflict, gin.H{"result": result})
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
		h.sendEvent(audit.CreateEvent(int(userID), audit.Follow, result))
	}

	c.JSON(http.StatusCreated, gin.H{"result": result})
}
