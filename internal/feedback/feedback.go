package feedback

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Feedback struct {
	ID        string     `json:"id"`
	EventID   string     `json:"-"`
	Event     *Event     `json:"event"`
	UserID    string     `json:"-"`
	User      *User      `json:"user"`
	Message   string     `json:"message"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `json:"-"`
}

func (Feedback) TableName() string {
	return "feedback"
}

type CreateFeedback struct {
	EventID string `json:"event_id" binding:"required"`
	UserID  string `json:"user_id" binding:"required"`
	Message string `json:"message" binding:"required"`
}

type Event struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Speaker string    `json:"speaker"`
	StartAt time.Time `json:"start_at"`
}

type User struct {
	ID     string `json:"id" `
	Name   string `json:"name" binding:"required"`
	Gender string `json:"gender" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, feedback *Feedback) (*Feedback, error)
	Index(ctx *gin.Context) ([]*Feedback, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateFeedback) (*Feedback, error)
	Index(ctx *gin.Context) ([]*Feedback, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	Index(ctx *gin.Context)
}
