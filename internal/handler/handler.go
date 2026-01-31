package handler

// import (
// 	"net/http"

// 	"github.com/gin-gonic/gin"
// 	"github.com/sirupsen/logrus"
// )

// func (h Handlers) DelUserURLS(c *gin.Context) {
// 	ctx := c.Request.Context()
// 	var URLSToDel []string
// 	if err := c.ShouldBindJSON(&URLSToDel); err != nil {
// 		logrus.Error(err)
// 		c.Status(http.StatusBadRequest)
// 		return
// 	}
// 	c.Status(http.StatusAccepted)
// 	h.service.AsyncDeleteUserURLs(ctx, URLSToDel)
// }
