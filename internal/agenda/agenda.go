package agenda

import (
	"mainyuk/internal/divisi"
	"time"

	"github.com/gin-gonic/gin"
)

type Agenda struct {
	ID        string         `json:"id" binding:"required"`
	Name      string         `json:"name" binding:"required"`
	Type      string         `json:"type" binding:"required"`
	Location  string         `json:"location" binding:"required"`
	Start_At  time.Time      `json:"start_at" binding:"required"`
	DivisiID  string         `json:"-" binding:"required"`
	Divisi    *divisi.Divisi `json:"divisi"`
	UserID    string         `json:"-" binding:"required"`
	User      *User          `json:"leader"`
	CreatedAt time.Time      `json:"created_at" binding:"required"`
	UpdatedAt time.Time      `json:"-" binding:"required"`
	DeletedAt *time.Time     `json:"-" binding:"required"`
}

func (Agenda) TableName() string {
	return "agenda"
}

type User struct {
	ID   string `json:"id" `
	Name string `json:"name" `
	Role string `json:"role"`
	Age  int    `json:"age"`
}

type CreateAgenda struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Location string `json:"location" binding:"required"`
	StartAt  string `json:"start_at" binding:"required"`
	DivisiID string `json:"divisi_id" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, agenda *Agenda) (*Agenda, error)
	Show(ctx *gin.Context, id string) (*Agenda, error)
	Index(ctx *gin.Context) ([]*Agenda, error)
	Update(ctx *gin.Context, id string, agenda *Agenda) (*Agenda, error)
	Delete(ctx *gin.Context, id string) error
}

type Service interface {
	Create(ctx *gin.Context, agenda *CreateAgenda) (*Agenda, error)
	Show(ctx *gin.Context, id string) (*Agenda, error)
	Index(ctx *gin.Context) ([]*Agenda, error)
	Update(ctx *gin.Context, id string, agenda *CreateAgenda) (*Agenda, error)
	Delete(ctx *gin.Context, id string) error
}

type Handler interface {
	Create(ctx *gin.Context)
	Show(ctx *gin.Context)
	Index(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
}
