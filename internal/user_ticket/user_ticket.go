package user_ticket

import (
	"mainyuk/internal/event"
	"mainyuk/internal/ticket"
	"time"

	"github.com/gin-gonic/gin"
)

type UserTicket struct {
	ID         string         `json:"id" binding:"required"`
	PublicID   string         `json:"public_id" binding:"required"`
	UserName   string         `json:"user_name" binding:"required"`
	UserEmail  string         `json:"user_email" binding:"required"`
	UserGender string         `json:"user_gender" binding:"required"`
	UserID     string         `json:"-" gorm:"user_id"`
	User       *User          `json:"user" gorm:"foreignKey:user_id;references:id"`
	OrderID    string         `json:"-" gorm:"order_id"`
	TicketID   string         `json:"-" gorm:"ticket_id"`
	Ticket     *ticket.Ticket `json:"ticket" gorm:"foreignKey:ticket_id;references:id"`
	EventID    string         `json:"-"`
	Event      *event.Event   `json:"event" gorm:"foreignKey:event_id;references:id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"-"`
	DeletedAt  *time.Time     `json:"-"`
}

type User struct {
	ID       string  `json:"id" `
	Name     string  `json:"name" binding:"required"`
	Username string  `json:"username" binding:"required"`
	Gender   string  `json:"gender" binding:"required"`
	Email    *string `json:"email" binding:"required"`
}
type CreateUserTicket struct {
	UserName   string `json:"user_name" binding:"required"`
	UserEmail  string `json:"user_email" binding:"required"`
	UserGender string `json:"user_gender" binding:"required"`
	UserID     string `json:"user_id" binding:"required"`
	OrderID    string `json:"order_id" binding:"required"`
	TicketID   string `json:"ticket_id" binding:"required"`
	EventID    string `json:"event_id" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, userTicket *UserTicket) (*UserTicket, error)
	Update(ctx *gin.Context, id string, userTicket *UserTicket) (*UserTicket, error)
	Show(ctx *gin.Context, id string) (*UserTicket, error)
	ShowByPublicID(ctx *gin.Context, public_id string) (*UserTicket, error)
	Index(ctx *gin.Context) ([]*UserTicket, error)
	IndexByOrderID(ctx *gin.Context, order_id string) ([]*UserTicket, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateUserTicket) (*UserTicket, error)
	Update(ctx *gin.Context, id string, req *CreateUserTicket) (*UserTicket, error)
	Show(ctx *gin.Context, id string) (*UserTicket, error)
	ShowByPublicID(ctx *gin.Context, public_id string) (*UserTicket, error)
	Index(ctx *gin.Context) ([]*UserTicket, error)
	IndexByOrderID(ctx *gin.Context, order_id string) ([]*UserTicket, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	ShowByPublicID(ctx *gin.Context)
	Index(ctx *gin.Context)
}
