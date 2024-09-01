package order

import (
	"fmt"
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
	var order CreateOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	res, err := h.Service.Create(c, &order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handler) Show(c *gin.Context) {
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

func (h *handler) ShowByPublicID(c *gin.Context) {
	public_id := c.Param("public_id")
	res, err := h.Service.Show(c, public_id)
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
