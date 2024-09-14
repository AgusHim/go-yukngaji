package payment_method

import (
	"time"

	"github.com/gin-gonic/gin"
)

type PaymentMethod struct {
	ID            string     `json:"id" binding:"required"`
	Type          string     `json:"type"` // type = BANK,E-WALLET, QRIS
	Code          string     `json:"code"`
	Name          string     `json:"name"`
	ImageUrl      string     `json:"image_url"`
	AccountName   string     `json:"account_name"`
	AccountNumber string     `json:"account_number"`
	CreatedAt     time.Time  `json:"-" binding:"required"`
	UpdatedAt     time.Time  `json:"-" binding:"required"`
	DeletedAt     *time.Time `json:"-" gorm:"index" binding:"required"`
}

func (PaymentMethod) TableName() string {
	return "payment_method"
}

type CreatePaymentMethod struct {
	Name          string `json:"name" binding:"required"`
	Type          string `json:"type" binding:"required"`
	Code          string `json:"code" binding:"required"`
	AccountName   string `json:"account_name" binding:"required"`
	AccountNumber string `json:"account_number" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, PaymentMethod *PaymentMethod) (*PaymentMethod, error)
	Show(ctx *gin.Context, id string) (*PaymentMethod, error)
	Index(ctx *gin.Context) ([]*PaymentMethod, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreatePaymentMethod) (*PaymentMethod, error)
	Show(ctx *gin.Context, id string) (*PaymentMethod, error)
	Index(ctx *gin.Context) ([]*PaymentMethod, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	Show(ctx *gin.Context)
	Index(ctx *gin.Context)
}
