package auth

import "github.com/gin-gonic/gin"

type Middleware interface {
	AuthAdmin(c *gin.Context)
	AuthPJ(c *gin.Context)
	AuthRanger(c *gin.Context)
	AuthUser(c *gin.Context)
}
