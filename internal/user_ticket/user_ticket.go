package user_ticket

import (
	"mainyuk/internal/ticket"
	"os/user"
	"time"

	"github.com/gin-gonic/gin"
)

type UserTicket struct {
	ID         string        `json:"id" binding:"required"`
	PublicID   string        `json:"public_id" binding:"required"`
	UserName   string        `json:"user_name" binding:"required"`
	UserEmail  string        `json:"user_email" binding:"required"`
	UserGender string        `json:"user_gender" binding:"required"`
	UserID     string        `json:"-"`
	User       user.User     `json:"user"`
	OrderID    string        `json:"-"`
	TicketID   string        `json:"-"`
	Ticket     ticket.Ticket `json:"ticket"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"-"`
	DeletedAt  *time.Time    `json:"-"`
}

type CreateUserTicket struct {
	UserName   string `json:"user_name" binding:"required"`
	UserEmail  string `json:"user_email" binding:"required"`
	UserGender string `json:"user_gender" binding:"required"`
	UserID     string `json:"user_id" binding:"required"`
	OrderID    string `json:"order_id" binding:"required"`
	TicketID   string `json:"ticket_id" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, userTicket *UserTicket) (*UserTicket, error)
	Update(ctx *gin.Context, id string, userTicket *UserTicket) (*UserTicket, error)
	Show(ctx *gin.Context, id string) (*UserTicket, error)
	Index(ctx *gin.Context) ([]*UserTicket, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateUserTicket) (*UserTicket, error)
	Update(ctx *gin.Context, id string, req *CreateUserTicket) (*UserTicket, error)
	Show(ctx *gin.Context, id string) (*UserTicket, error)
	Index(ctx *gin.Context) ([]*UserTicket, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	Show(ctx *gin.Context)
	Index(ctx *gin.Context)
}
