package presence

import (
	"mainyuk/internal/event"
	"mainyuk/internal/user"
	"mainyuk/internal/user_ticket"
	"time"

	"github.com/gin-gonic/gin"
)

type Presence struct {
	ID           string                 `json:"id"`
	UserID       string                 `json:"-" binding:"required"`
	EventID      string                 `json:"-"  binding:"required"`
	User         *user.User             `json:"user"`
	Event        *event.Event           `json:"event"`
	UserTicketID *string                `json:"-"`
	UserTicket   user_ticket.UserTicket `json:"user_ticket" gorm:"foreignKey:user_ticket_id;references:id"`
	AdminID      *string                `json:"admin_id"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"-"`
	DeletedAt    *time.Time             `json:"-"`
}

type ResScanTicket struct {
	UserTicket user_ticket.UserTicket `json:"user_ticket" gorm:"foreignKey:ticket_id;references:id"`
	Presences  []*time.Time           `json:"presences"`
}

func presenceToPresences(presences []*Presence) (res []*time.Time) {
	var filtered []*time.Time
	for _, p := range presences {
		filtered = append(filtered, &p.CreatedAt)
	}
	return filtered
}

func (Presence) TableName() string {
	return "presence"
}

type CreatePresence struct {
	EventID      string           `json:"event_id" binding:"required"`
	UserID       *string          `json:"user_id" `
	User         *user.CreateUser `json:"user"`
	UserTicketID *string          `json:"user_ticket_id"`
}

type PresenceFromTicket struct {
	PublicID string `json:"public_id" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, event *Presence) (*Presence, error)
	Show(ctx *gin.Context, id string) (*Presence, error)
	Index(ctx *gin.Context) ([]*Presence, error)
	IndexByUserTicket(ctx *gin.Context, user_ticket_id string) ([]*Presence, error)
	FindByUserID(ctx *gin.Context, id string, eventID string) (*Presence, error)
	FindByUserTicketID(ctx *gin.Context, id string, eventID string) (*Presence, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreatePresence) (*Presence, error)
	CreateFromTicket(ctx *gin.Context, slug string, public_id string) (*ResScanTicket, error)
	Show(ctx *gin.Context, id string) (*Presence, error)
	Index(ctx *gin.Context) ([]*Presence, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	Show(ctx *gin.Context)
	Index(ctx *gin.Context)
	CreateFromTicket(ctx *gin.Context)
}
