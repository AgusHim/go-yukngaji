package auth

import (
	"log"
	"mainyuk/internal/user"
	"mainyuk/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type middleware struct {
	UserService user.Service
}

func NewMiddleware(us user.Service) Middleware {
	return &middleware{
		UserService: us,
	}
}

func (m *middleware) AuthAdmin(c *gin.Context) {

	/*Check header Bearer*/
	authHeader := c.GetHeader("Authorization")

	if !strings.Contains(authHeader, "Bearer") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize bearer",
		})
		return
	}

	tokenString := ""
	arrayToken := strings.Split(authHeader, " ")
	if len(arrayToken) == 2 {
		tokenString = arrayToken[1]
	}

	token, err := utils.ValidateJWT(tokenString)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize validate",
		})
		return
	}

	claim, ok := token.Claims.(jwt.MapClaims)

	if !ok || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize claim",
		})
		return
	}

	userID := claim["user_id"].(string)

	user, errUser := m.UserService.Show(c, userID)
	if errUser != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize not found",
		})
		return
	}
	if user.Role != "admin" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize role",
		})
		return
	}

	c.Set("currentUser", *user)
}

func (m *middleware) AuthPJ(c *gin.Context) {

	/*Check header Bearer*/
	authHeader := c.GetHeader("Authorization")

	if !strings.Contains(authHeader, "Bearer") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize bearer",
		})
		return
	}

	tokenString := ""
	arrayToken := strings.Split(authHeader, " ")
	if len(arrayToken) == 2 {
		tokenString = arrayToken[1]
	}

	token, err := utils.ValidateJWT(tokenString)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize validate",
		})
		return
	}

	claim, ok := token.Claims.(jwt.MapClaims)

	if !ok || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize claim",
		})
		return
	}

	userID := claim["user_id"].(string)

	user, errUser := m.UserService.Show(c, userID)
	if errUser != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize not found",
		})
		return
	}
	if user.Role != "admin" && user.Role != "pj" {
		log.Println("User role =", user.Role)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize role",
		})
		return
	}

	c.Set("currentUser", *user)
}

func (m *middleware) AuthRanger(c *gin.Context) {

	/*Check header Bearer*/
	authHeader := c.GetHeader("Authorization")

	if !strings.Contains(authHeader, "Bearer") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize bearer",
		})
		return
	}

	tokenString := ""
	arrayToken := strings.Split(authHeader, " ")
	if len(arrayToken) == 2 {
		tokenString = arrayToken[1]
	}

	token, err := utils.ValidateJWT(tokenString)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize validate",
		})
		return
	}

	claim, ok := token.Claims.(jwt.MapClaims)

	if !ok || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize claim",
		})
		return
	}

	userID := claim["user_id"].(string)

	user, errUser := m.UserService.Show(c, userID)
	if errUser != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize not found",
		})
		return
	}
	if user.Role != "ranger" && user.Role != "pj" {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize role",
		})
		return
	}

	c.Set("currentUser", *user)
}

func (m *middleware) AuthUser(c *gin.Context) {

	/*Check header Bearer*/
	authHeader := c.GetHeader("Authorization")

	if !strings.Contains(authHeader, "Bearer") {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize bearer",
		})
		return
	}

	tokenString := ""
	arrayToken := strings.Split(authHeader, " ")
	if len(arrayToken) == 2 {
		tokenString = arrayToken[1]
	}

	token, err := utils.ValidateJWT(tokenString)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize validate",
		})
		return
	}

	claim, ok := token.Claims.(jwt.MapClaims)

	if !ok || !token.Valid {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize claim",
		})
		return
	}

	userID := claim["user_id"].(string)

	user, errUser := m.UserService.Show(c, userID)
	if errUser != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorize not found",
		})
		return
	}
	c.Set("currentUser", *user)
}
