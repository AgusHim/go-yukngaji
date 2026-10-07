package community

import (
	"net/http"

	"mainyuk/internal/httperr"

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

func (h *handler) Me(c *gin.Context) {
	res, err := h.Service.Me(c)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": ToOwnProfile(res)})
}

func (h *handler) Update(c *gin.Context) {
	var req UpdateProfile
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
		return
	}

	res, err := h.Service.Update(c, &req)
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": ToOwnProfile(res)})
}

func (h *handler) ShowPublic(c *gin.Context) {
	res, err := h.Service.ShowPublic(c, c.Param("public_id"))
	if err != nil {
		httperr.JSON(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"profile": res})
}
