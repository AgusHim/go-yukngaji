package feedback

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
)

type handler struct {
	Service
}

func NewHandler(s Service) Handler {
	return &handler{
		s,
	}
}

func (h *handler) Create(c *gin.Context) {
	var feedback CreateFeedback
	if err := c.ShouldBindJSON(&feedback); err != nil {
		body, _ := io.ReadAll(c.Request.Body)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid JSON",
			"req":   body,
		})
		return
	}

	res, err := h.Service.Create(c, &feedback)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handler) Index(c *gin.Context) {
	res, err := h.Service.Index(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, res)
}
