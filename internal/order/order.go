package order

import (
	"mainyuk/internal/divisi"
	"mainyuk/internal/event"
	"mainyuk/internal/payment_method"
	"mainyuk/internal/user"
	"mainyuk/internal/user_ticket"
	"time"

	"github.com/gin-gonic/gin"
)

type Order struct {
	ID              string                        `json:"id" binding:"required"`
	PublicID        string                        `json:"public_id"`
	Amount          int                           `json:"amount" binding:"required"`
	Donation        int                           `json:"donation" binding:"required"`
	AdminFee        int                           `json:"admin_fee" binding:"required"`
	Status          string                        `json:"status"`
	InvoiceUrl      *string                       `json:"invoice_url"`
	InvoiceImageUrl *string                       `json:"invoice_image_url"`
	UserID          string                        `json:"-"`
	User            *user.User                    `json:"user" binding:"required"`
	PaymentMethodID *string                       `json:"-"`
	PaymentMethod   *payment_method.PaymentMethod `json:"payment_method" gorm:"foreignKey:payment_method_id;references:id"`
	EventID         string                        `json:"-"`
	Event           *event.Event                  `json:"event" gorm:"foreignKey:event_id;references:id"`
	UserTickets     []*user_ticket.UserTicket     `json:"user_tickets" gorm:"foreignKey:order_id"`
	ExpiredAt       *time.Time                    `json:"expired_at"`
	CreatedAt       time.Time                     `json:"created_at"`
	UpdatedAt       time.Time                     `json:"-"`
	DeletedAt       *time.Time                    `json:"-"`
}

type Event struct {
	ID       string         `json:"id"`
	Slug     string         `json:"slug" `
	Code     string         `json:"code" `
	Title    string         `json:"title" binding:"required"`
	Desc     string         `json:"desc" binding:"required"`
	ImageUrl string         `json:"image_url" binding:"required"`
	Speaker  string         `json:"speaker" binding:"required"`
	DivisiID string         `json:"-" binding:"required"`
	Divisi   *divisi.Divisi `json:"divisi"`
	StartAt  time.Time      `json:"start_at" binding:"required"`
	EndAt    time.Time      `json:"end_at" binding:"required"`
}

type CreateOrder struct {
	EventID         string       `json:"event_id"`
	PaymentMethodID string       `json:"payment_method_id"`
	UserTickets     []UserTicket `json:"user_tickets" gorm:"-" binding:"required"`
	Donation        *int         `json:"donation"`
	AdminFee        *int         `json:"admin_fee"`
}

type UpdateOrder struct {
	Status string `json:"status"`
}

type UserTicket struct {
	UserName   string `json:"user_name" binding:"required"`
	UserEmail  string `json:"user_email" binding:"required"`
	UserGender string `json:"user_gender" binding:"required"`
	TicketID   string `json:"ticket_id" binding:"required"`
	EventID    string `json:"event_id" binding:"required"`
	UserID     string `json:"user_id" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, order *Order) (*Order, error)
	Show(ctx *gin.Context, id string) (*Order, error)
	ShowByPublicID(ctx *gin.Context, public_id string, user_id *string) (*Order, error)
	Index(ctx *gin.Context, user_id *string) ([]*Order, error)
	Update(ctx *gin.Context, order *Order) (*Order, error)
	Participants(ctx *gin.Context, event_id string) ([]*Order, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateOrder) (*Order, error)
	Show(ctx *gin.Context, id string) (*Order, error)
	ShowByPublicID(ctx *gin.Context, public_id string) (*Order, error)
	Index(ctx *gin.Context) ([]*Order, error)
	IndexAdmin(ctx *gin.Context) ([]*Order, error)
	VerifyOrder(ctx *gin.Context, id string, status string) (*Order, error)
	Participants(ctx *gin.Context, event_id string) ([]*Order, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	ShowByPublicID(ctx *gin.Context)
	Show(ctx *gin.Context)
	Index(ctx *gin.Context)
	IndexAdmin(ctx *gin.Context)
	VerifyOrder(ctx *gin.Context)
	Participants(ctx *gin.Context)
}
