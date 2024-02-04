package comment

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Comment struct {
	ID        string     `json:"id"`
	EventID   string     `json:"event_id"`
	UserID    string     `json:"-"`
	User      *User      `json:"user"`
	Comment   string     `json:"comment"`
	Like      int        `json:"like"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `json:"-"`
}

type CreateComment struct {
	EventID string `json:"event_id" binding:"required"`
	UserID  string `json:"user_id" binding:"required"`
	Comment string `json:"comment" binding:"required"`
}

type User struct {
	ID       string `json:"id" `
	Username string `json:"username" binding:"required"`
	Gender   string `json:"gender" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, comment *Comment) (*Comment, error)
	Show(ctx *gin.Context, id string) (*Comment, error)
	Index(ctx *gin.Context) ([]*Comment, error)
	Update(ctx *gin.Context, comment *Comment) (*Comment, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateComment) (*Comment, error)
	Show(ctx *gin.Context, id string) (*Comment, error)
	Index(ctx *gin.Context) ([]*Comment, error)
	Update(ctx *gin.Context, comment *Comment) (*Comment, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	Index(ctx *gin.Context)
}
