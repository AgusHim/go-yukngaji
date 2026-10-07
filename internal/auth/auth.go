package auth

import (
	"mainyuk/internal/user"

	"github.com/gin-gonic/gin"
)

type Middleware interface {
	AuthAdmin(c *gin.Context)
	AuthPJ(c *gin.Context)
	AuthRanger(c *gin.Context)
	AuthUser(c *gin.Context)
	AuthOptionalUser(c *gin.Context)
	// UserFromToken memvalidasi token JWT di luar header Authorization.
	// Dipakai WebSocket, yang menerima token lewat query saat handshake.
	UserFromToken(c *gin.Context, token string) (*user.User, error)
}
