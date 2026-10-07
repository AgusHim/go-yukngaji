// Package httperr memetakan error domain ke status HTTP secara konsisten.
package httperr

import (
	"errors"
	"net/http"

	"mainyuk/internal/apperr"
	"mainyuk/internal/ratelimit"

	"github.com/gin-gonic/gin"
)

// Status mengembalikan kode HTTP yang sesuai untuk err.
func Status(err error) int {
	switch {
	case err == nil:
		return http.StatusOK
	case errors.Is(err, ratelimit.ErrTooManyRequests):
		return http.StatusTooManyRequests
	case errors.Is(err, apperr.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, apperr.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, apperr.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, apperr.ErrInvalidRequest):
		return http.StatusBadRequest
	case apperr.IsValidation(err):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}

// JSON menulis err sebagai respons JSON dengan status yang sesuai.
func JSON(c *gin.Context, err error) {
	c.JSON(Status(err), gin.H{"error": err.Error()})
}
