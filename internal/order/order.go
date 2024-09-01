package order

import (
	"mainyuk/internal/user"
	"mainyuk/internal/user_ticket"
	"time"

	"github.com/gin-gonic/gin"
)

type Order struct {
	ID              string                   `json:"id" binding:"required"`
	PublicID        string                   `json:"public_id"`
	Amount          int                      `json:"amount" binding:"required"`
	Donation        int                      `json:"donation" binding:"required"`
	AdminFee        int                      `json:"description" binding:"required"`
	Status          string                   `json:"status"`
	InvoiceUrl      *string                  `json:"invoice_url"`
	InvoiceImageUrl *string                  `json:"invoice_image_url"`
	UserID          string                   `json:"-"`
	User            *user.User               `json:"user" binding:"required"`
	UserTickets     []user_ticket.UserTicket `json:"user_tickets" gorm:"-"`
	ExpiredAt       time.Time                `json:"expired_at"`
	CreatedAt       time.Time                `json:"created_at"`
	UpdatedAt       time.Time                `json:"-"`
	DeletedAt       *time.Time               `json:"-"`
}

type CreateOrder struct {
	UserID      string       `json:"user_id" binding:"required"`
	UserTickets []UserTicket `json:"user_tickets" gorm:"-"`
	Donation    *int         `json:"donation"`
}

type UserTicket struct {
	UserName   string `json:"user_name" binding:"required"`
	UserEmail  string `json:"user_email" binding:"required"`
	UserGender string `json:"user_gender" binding:"required"`
	TicketID   string `json:"-"`
}

type Repository interface {
	Create(ctx *gin.Context, order *Order) (*Order, error)
	Show(ctx *gin.Context, id string) (*Order, error)
	ShowByPublicID(ctx *gin.Context, public_id string) (*Order, error)
	Index(ctx *gin.Context) ([]*Order, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateOrder) (*Order, error)
	Show(ctx *gin.Context, id string) (*Order, error)
	ShowByPublicID(ctx *gin.Context, public_id string) (*Order, error)
	Index(ctx *gin.Context) ([]*Order, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	ShowByPublicID(ctx *gin.Context)
	Show(ctx *gin.Context)
	Index(ctx *gin.Context)
}
