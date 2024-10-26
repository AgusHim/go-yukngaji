package otp

import (
	"mainyuk/internal/user"
	"time"

	"github.com/gin-gonic/gin"
)

type Otp struct {
	ID        string `gorm:"primaryKey"`
	Email     string
	Code      string
	ExpiresAt time.Time
	CreatedAt time.Time
}

func (Otp) TableName() string {
	return "otp_tx"
}

type ReqOtp struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type Repository interface {
	Create(c *gin.Context, otp *Otp) (*Otp, error)
	Show(c *gin.Context, email *string, code *string) (*Otp, error)
}

type Service interface {
	RequestOTP(c *gin.Context, req ReqOtp) (*Otp, error)
	VerifyOTP(c *gin.Context, req ReqOtp) (*user.User, error)
}

type Handler interface {
	RequestOTP(c *gin.Context)
	VerifyOTP(c *gin.Context)
}
