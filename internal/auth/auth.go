package auth

import "github.com/gin-gonic/gin"

type Middleware interface {
	AuthAdmin(c *gin.Context)
}