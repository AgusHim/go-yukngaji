package user

import (
	"time"

	"github.com/gin-gonic/gin"
)

type User struct {
	ID        string     `json:"id" `
	Name      string     `json:"name" binding:"required"`
	Username  string     `json:"username" binding:"required"`
	Gender    string     `json:"gender" binding:"required"`
	Age       int        `json:"age" binding:"required"`
	Phone     string     `json:"phone" binding:"required"`
	Email     string     `json:"email" binding:"required"`
	Address   string     `json:"address" binding:"required"`
	Password  string     `json:"-" binding:"required"`
	Role      string     `json:"role"`
	CreatedAt time.Time  `json:"created_at" `
	UpdatedAt time.Time  `json:"-" `
	DeletedAt *time.Time `json:"-" `
}

type Login struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateUser struct {
	Name     string `json:"name" binding:"required"`
	Gender   string `json:"gender"`
	Age      string    `json:"age"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Address  string `json:"address"`
	Password string `json:"password"`
}

type Repository interface {
	CreateUser(c *gin.Context, user *User) (*User, error)
	GetUserByEmail(c *gin.Context, email string) (*User, error)
	DeleteByID(c *gin.Context, id string) error
	Show(c *gin.Context, id string) (*User, error)
}

type Service interface {
	Register(c *gin.Context, user *CreateUser) (*User, error)
	Login(c *gin.Context, user *Login) (*User, error)
	GetUserByEmail(c *gin.Context, email string) (*User, error)
	Show(c *gin.Context, id string) (*User, error)
	Presence(c *gin.Context, user *CreateUser) (*User, error)
	DeleteByID(c *gin.Context, id string) error
}

type Handler interface {
	Register(c *gin.Context)
	Login(c *gin.Context)
}
