package user

import (
	"fmt"
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

func (h *handler) Register(c *gin.Context) {
	var u CreateUser
	if err := c.ShouldBindJSON(&u); err != nil {
		if err.Error() == "EmailRegistered" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Email already registered",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	res, err := h.Service.Register(c, &u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, res)
}

func (h *handler) Login(c *gin.Context) {
	var u Login
	if err := c.ShouldBindJSON(&u); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	res, err := h.Service.Login(c, &u)
	if err != nil {
		if err.Error() == "EmailNotFound" || err.Error() == "PasswordNotMatch" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Wrong email or password",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
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

	c.JSON(http.StatusOK, gin.H{
		"user":         res,
		"access_token": token,
	})
}

func (h *handler) UpdateByAdmin(c *gin.Context) {
	id := c.Param("id")
	var u CreateUser
	if err := c.ShouldBindJSON(&u); err != nil {
		if err.Error() == "EmailRegistered" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Email already registered",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	res, err := h.Service.UpdateByAdmin(c, id, &u)
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
	token, err := utils.GenerateJWT(res.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "error generate token",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user":         res,
		"access_token": token,
	})
}

func (h *handler) UpdateAuth(c *gin.Context) {
	authUser, exists := c.Get("currentUser")
	if !exists {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Not Authorized",
		})
		return
	}
	currentUser, ok := authUser.(User)

	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Error parsing current user",
		})
		return
	}

	var u CreateUser
	if err := c.ShouldBindJSON(&u); err != nil {
		if err.Error() == "EmailRegistered" {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Email already registered",
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON",
		})
		return
	}

	res, err := h.Service.UpdateByAdmin(c, currentUser.ID, &u)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintln(err.Error()),
		})
		return
	}
	c.JSON(http.StatusOK, res)
}
