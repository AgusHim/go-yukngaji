package event_verifier

import (
	"errors"
	"net/http"

	"mainyuk/internal/presence"
	"mainyuk/internal/ratelimit"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type handler struct{ service Service }

func NewHandler(service Service) Handler { return &handler{service: service} }

func writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, ErrUnauthorized):
		status = http.StatusUnauthorized
	case errors.Is(err, ErrDisabled), errors.Is(err, gorm.ErrRecordNotFound):
		status = http.StatusNotFound
	case errors.Is(err, ErrInvalid):
		status = http.StatusBadRequest
	case errors.Is(err, ratelimit.ErrTooManyRequests):
		status = http.StatusTooManyRequests
	default:
		if presenceStatus, _ := presence.ErrorCode(err); presenceStatus != http.StatusInternalServerError {
			status = presenceStatus
		}
	}
	c.JSON(status, gin.H{"error": err.Error()})
}

func (h *handler) AdminStatus(c *gin.Context) {
	value, err := h.service.Status(c, c.Param("event_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *handler) SetPassword(c *gin.Context) {
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password wajib diisi"})
		return
	}
	value, err := h.service.SetPassword(c, c.Param("event_id"), body.Password)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *handler) Disable(c *gin.Context) {
	if err := h.service.Disable(c, c.Param("event_id")); err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "akses verifikator dinonaktifkan"})
}

func (h *handler) Login(c *gin.Context) {
	var body struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password wajib diisi"})
		return
	}
	value, err := h.service.Login(c, c.Param("slug"), body.Password)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *handler) Session(c *gin.Context) {
	value, _, err := h.service.Session(c, c.Param("slug"), TokenFromRequest(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *handler) Tickets(c *gin.Context) {
	value, err := h.service.Tickets(c, c.Param("slug"), TokenFromRequest(c), c.Query("search"), c.Query("ticket_id"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *handler) TicketOptions(c *gin.Context) {
	value, err := h.service.TicketOptions(c, c.Param("slug"), TokenFromRequest(c))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}

func (h *handler) CheckIn(c *gin.Context) {
	var body struct {
		PublicID string `json:"public_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kode tiket wajib diisi"})
		return
	}
	value, err := h.service.CheckIn(c, c.Param("slug"), TokenFromRequest(c), body.PublicID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
