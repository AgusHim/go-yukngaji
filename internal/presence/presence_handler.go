package presence

import (
	"net/http"

	"mainyuk/utils"

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

// writeError mengirim error beserta status HTTP dan kode mesin yang sesuai,
// supaya UI bisa menampilkan pesan spesifik alih-alih "terjadi kesalahan".
func writeError(c *gin.Context, err error) {
	status, code := ErrorCode(err)
	c.JSON(status, gin.H{
		"error": err.Error(),
		"code":  code,
	})
}

func (h *handler) Create(c *gin.Context) {
	var presence CreatePresence
	if err := c.ShouldBindJSON(&presence); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
			"code":  "INVALID_JSON",
		})
		return
	}

	if presence.UserID == nil && presence.User != nil {
		if presence.User.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Nama peserta wajib diisi",
				"code":  "INVALID_REQUEST",
			})
			return
		}
	}

	res, err := h.Service.Create(c, &presence)
	if err != nil {
		writeError(c, err)
		return
	}

	token, err := utils.GenerateJWT(res.User.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error generate token",
			"code":  "INTERNAL_ERROR",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"presence":     res,
		"access_token": token,
	})
}

func (h *handler) Show(c *gin.Context) {
	id := c.Param("id")
	res, err := h.Service.Show(c, id)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handler) Index(c *gin.Context) {
	res, err := h.Service.Index(c)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}

// CreateFromTicket melayani pemindaian tiket oleh ranger.
//
// Respons sukses memuat `already_checked_in` dan `check_in_date` supaya UI
// bisa membedakan "check-in berhasil" dari "tiket ini sudah check-in hari
// ini" tanpa menebak dari panjang riwayat.
func (h *handler) CreateFromTicket(c *gin.Context) {
	slug := c.Param("slug")
	var ticket PresenceFromTicket
	if err := c.ShouldBindJSON(&ticket); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
			"code":  "INVALID_JSON",
		})
		return
	}

	res, err := h.Service.CreateFromTicket(c, slug, ticket.PublicID)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, res)
}
