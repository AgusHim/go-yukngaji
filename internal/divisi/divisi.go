package divisi

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Divisi struct {
	ID        string     `json:"id" binding:"required"`
	Name      string     `json:"name" binding:"required"`
	Regional  string     `json:"regional" binding:"required"`
	CreatedAt time.Time  `json:"-" binding:"required"`
	UpdatedAt time.Time  `json:"-" binding:"required"`
	DeletedAt *time.Time `json:"-" binding:"required"`
}

func (Divisi) TableName() string {
	return "divisi"
}

type CreateDivisi struct {
	Name     string `json:"name" binding:"required"`
	Regional string `json:"regional" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, divisi *Divisi) (*Divisi, error)
	Show(ctx *gin.Context, id string) (*Divisi, error)
	Index(ctx *gin.Context) ([]*Divisi, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateDivisi) (*Divisi, error)
	Show(ctx *gin.Context, id string) (*Divisi, error)
	Index(ctx *gin.Context) ([]*Divisi, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	Show(ctx *gin.Context)
	Index(ctx *gin.Context)
}
