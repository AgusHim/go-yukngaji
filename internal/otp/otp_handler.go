package otp

import (
	"errors"
	"mainyuk/internal/user"
	"mainyuk/utils"
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

// statusForError memetakan error OTP ke status HTTP yang sesuai, supaya
// client bisa membedakan "coba lagi nanti" dari kegagalan server.
func statusForError(err error) int {
	switch {
	case errors.Is(err, ErrRateLimited):
		return http.StatusTooManyRequests
	case errors.Is(err, ErrEmailInvalid),
		errors.Is(err, ErrOTPNotFound),
		errors.Is(err, ErrOTPUsed),
		errors.Is(err, ErrOTPExpired),
		errors.Is(err, ErrOTPMismatch),
		errors.Is(err, ErrOTPAttempts):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

func (h *handler) RequestOTP(c *gin.Context) {
	var req ReqOtp
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
		})
		return
	}
	res, err := h.Service.RequestOTP(c, req)
	if err != nil {
		c.JSON(statusForError(err), gin.H{
			"error": err.Error(),
		})
		return
	}
	if res == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed request OTP",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Success request OTP. Please check email",
	})
}

func (h *handler) VerifyOTP(c *gin.Context) {
	var req ReqOtp
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
		})
		return
	}
	res, err := h.Service.VerifyOTP(c, req)
	if err != nil {
		c.JSON(statusForError(err), gin.H{
			"error": err.Error(),
		})
		return
	}

	token, err := utils.GenerateJWT(res.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error generate token",
		})
		return
	}

	// DTO akun yang sama dengan Login/Me, supaya kontrak respons konsisten
	// dan data internal tidak ikut terkirim.
	c.JSON(http.StatusOK, gin.H{
		"user":         user.ToAccountResponse(res),
		"access_token": token,
	})
}
