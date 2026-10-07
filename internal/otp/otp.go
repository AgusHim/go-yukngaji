package otp

import (
	"mainyuk/internal/user"
	"time"

	"github.com/gin-gonic/gin"
)

type Otp struct {
	ID         string `gorm:"primaryKey"`
	Email      string
	Code       string
	ExpiresAt  time.Time
	Attempts   int        `gorm:"default:0"`
	UsedAt     *time.Time `gorm:"column:used_at"`
	LastSentAt *time.Time `gorm:"column:last_sent_at"`
	CreatedAt  time.Time
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
	// ShowActive mengambil OTP terbaru yang belum dipakai untuk email.
	ShowActive(c *gin.Context, email string) (*Otp, error)
	// MarkUsed menandai kode terpakai. Mengembalikan false bila baris sudah
	// terpakai lebih dulu (replay request paralel).
	MarkUsed(c *gin.Context, id string) (bool, error)
	// IncrementAttempts menaikkan penghitung percobaan salah secara atomik.
	IncrementAttempts(c *gin.Context, id string) error
	// InvalidateActive menandai semua kode aktif email sebagai terpakai.
	InvalidateActive(c *gin.Context, email string) error
	// TouchLastSent memperbarui waktu kirim terakhir.
	TouchLastSent(c *gin.Context, id string) error
}

type Service interface {
	RequestOTP(c *gin.Context, req ReqOtp) (*Otp, error)
	VerifyOTP(c *gin.Context, req ReqOtp) (*user.User, error)
}

type Handler interface {
	RequestOTP(c *gin.Context)
	VerifyOTP(c *gin.Context)
}
