package presence

import (
	"mainyuk/internal/event"
	"mainyuk/internal/user"
	"time"

	"github.com/gin-gonic/gin"
)

type Presence struct {
	ID        string       `json:"id"`
	UserID    string       `json:"-" binding:"required"`
	EventID   string       `json:"-"  binding:"required"`
	User      *user.User   `json:"user"`
	Event     *event.Event `json:"event"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"-"`
	DeletedAt *time.Time   `json:"-"`
}

func (Presence) TableName() string {
	return "presence"
}

type CreatePresence struct {
	EventID string           `json:"event_id" binding:"required"`
	UserID  *string          `json:"user_id" `
	User    *user.CreateUser `json:"user"`
}

type Repository interface {
	Create(ctx *gin.Context, event *Presence) (*Presence, error)
	Show(ctx *gin.Context, id string) (*Presence, error)
	Index(ctx *gin.Context) ([]*Presence, error)
	FindByUserID(ctx *gin.Context, id string) (*Presence, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreatePresence) (*Presence, error)
	Show(ctx *gin.Context, id string) (*Presence, error)
	Index(ctx *gin.Context) ([]*Presence, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	Show(ctx *gin.Context)
	Index(ctx *gin.Context)
}
