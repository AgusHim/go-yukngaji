package event

import (
	"mainyuk/internal/divisi"
	"time"

	"github.com/gin-gonic/gin"
)

type Event struct {
	ID          string         `json:"id"`
	Slug        string         `json:"slug" `
	Code        string         `json:"code" `
	Title       string         `json:"title" binding:"required"`
	Desc        string         `json:"desc" binding:"required"`
	ImageUrl    string         `json:"image_url" binding:"required"`
	Speaker     string         `json:"speaker" binding:"required"`
	DivisiID    string         `json:"-" binding:"required"`
	Divisi      *divisi.Divisi `json:"divisi"`
	StartAt     time.Time      `json:"start_at" binding:"required"`
	EndAt       time.Time      `json:"end_at" binding:"required"`
	Participant int            `json:"participant"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"-"`
	DeletedAt   *time.Time     `json:"-"`
}

type CreateEvent struct {
	Title    string `json:"title" binding:"required"`
	Desc     string `json:"desc" binding:"required"`
	ImageUrl string `json:"image_url" binding:"required"`
	Speaker  string `json:"speaker" binding:"required"`
	DivisiID string `json:"divisi_id" binding:"required"`
	StartAt  string `json:"start_at" binding:"required"`
	EndAt    string `json:"end_at" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, event *Event) (*Event, error)
	Show(ctx *gin.Context, slug string) (*Event, error)
	ShowByCode(ctx *gin.Context, code string) (*Event, error)
	Index(ctx *gin.Context) ([]*Event, error)
	Update(ctx *gin.Context, event *Event) (*Event, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateEvent) (*Event, error)
	Show(ctx *gin.Context, slug string) (*Event, error)
	ShowByCode(ctx *gin.Context, code string) (*Event, error)
	Index(ctx *gin.Context) ([]*Event, error)
	Update(ctx *gin.Context, event *Event) (*Event, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	Show(ctx *gin.Context)
	ShowByCode(ctx *gin.Context)
	Index(ctx *gin.Context)
}
