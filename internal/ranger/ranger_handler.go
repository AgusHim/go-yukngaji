package ranger

import (
	"fmt"
	"mainyuk/internal/user"
	"net/http"
	"strings"

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
	var ranger CreateRanger
	if err := c.ShouldBindJSON(&ranger); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	res, err := h.Service.Create(c, &ranger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handler) Show(c *gin.Context) {
	if strings.Contains(c.FullPath(), "rangers/me") {
		u, exists := c.Get("currentUser")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error": "Not Authorized",
			})
			return
		}

		currentUser, ok := u.(user.User)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "FailedParsing: current user",
			})
			return
		}

		ranger, err := h.Service.ShowByUserID(c, currentUser.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Not Found",
			})
			return
		}
		c.JSON(http.StatusOK, ranger)
		return
	}
	id := c.Param("id")
	res, err := h.Service.Show(c, id)
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

func (h *handler) Update(c *gin.Context) {
	id := c.Param("id")
	var ranger CreateRanger
	if err := c.ShouldBindJSON(&ranger); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	res, err := h.Service.Update(c, id, &ranger)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handler) Delete(c *gin.Context) {
	id := c.Param("id")
	err := h.Service.Delete(c, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": "Success delete ranger",
	})
}
