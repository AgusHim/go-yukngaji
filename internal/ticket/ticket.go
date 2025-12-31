package ticket

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Ticket struct {
	ID            string     `json:"id" binding:"required"`
	Visibility    string     `json:"visibility"`
	Name          string     `json:"name" binding:"required"`
	Description   string     `json:"description" binding:"required"`
	Price         int        `json:"price" binding:"required"`
	EventID       string     `json:"event_id" binding:"required"`
	StartAt       time.Time  `json:"start_at" binding:"required"`
	EndAt         time.Time  `json:"end_at" binding:"required"`
	PaxMultiplier int        `json:"pax_multiplier"`
	MinOrderPax   *int       `json:"min_order_pax"`
	MaxOrderPax   *int       `json:"max_order_pax"`
	IsFull        bool       `json:"isFull" gorm:"-"`
	MaxPax        int        `json:"max_pax"`
	SoldPax       int        `json:"sold_pax" gorm:"-"`
	GenderAllowed string     `json:"gender_allowed"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"-"`
	DeletedAt     *time.Time `json:"-"`
}

type CreateTicket struct {
	Name          string  `json:"name" binding:"required"`
	Description   string  `json:"description" binding:"required"`
	Price         int     `json:"price"`
	EventID       string  `json:"event_id" binding:"required"`
	StartAt       string  `json:"start_at" binding:"required"`
	EndAt         string  `json:"end_at" binding:"required"`
	PaxMultiplier int     `json:"pax_multiplier" binding:"required"`
	MinOrderPax   *int    `json:"min_order_pax" `
	MaxOrderPax   *int    `json:"max_order_pax" `
	MaxPax        int     `json:"max_pax"`
	Visibility    *string `json:"visibility"`
	GenderAllowed *string `json:"gender_allowed"`
}

type Repository interface {
	Create(ctx *gin.Context, ticket *Ticket) (*Ticket, error)
	Update(ctx *gin.Context, id string, ticket *Ticket) (*Ticket, error)
	Show(ctx *gin.Context, id string) (*Ticket, error)
	Index(ctx *gin.Context) ([]*Ticket, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateTicket) (*Ticket, error)
	Update(ctx *gin.Context, id string, req *CreateTicket) (*Ticket, error)
	Show(ctx *gin.Context, id string) (*Ticket, error)
	Index(ctx *gin.Context) ([]*Ticket, error)
	Delete(ctx *gin.Context, id string) error
}

type Handler interface {
	Create(ctx *gin.Context)
	Show(ctx *gin.Context)
	Index(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
}
