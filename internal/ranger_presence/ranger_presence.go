package ranger_presence

import (
	"mainyuk/internal/agenda"
	"mainyuk/internal/divisi"
	"mainyuk/internal/ranger"
	"time"

	"github.com/gin-gonic/gin"
)

type RangerPresence struct {
	ID        string         `json:"id" binding:"required"`
	RangerID  string         `json:"-" binding:"required"`
	Ranger    *ranger.Ranger `json:"ranger"`
	AgendaID  string         `json:"-" binding:"required"`
	Agenda    *agenda.Agenda `json:"agenda"`
	DivisiID  string         `json:"-" binding:"required"`
	Divisi    *divisi.Divisi `json:"divisi"`
	CreatedAt time.Time      `json:"created_at" binding:"required"`
	UpdatedAt time.Time      `json:"-" binding:"required"`
	DeletedAt *time.Time     `json:"-" binding:"required"`
}

type CreatePresence struct {
	RangerID string `json:"ranger_id" binding:"required"`
	AgendaID string `json:"agenda_id" binding:"required"`
	DivisiID string `json:"divisi_id" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, ranger *RangerPresence) (*RangerPresence, error)
	Show(ctx *gin.Context, id string) (*RangerPresence, error)
	Index(ctx *gin.Context) ([]*RangerPresence, error)
	IndexByUserID(ctx *gin.Context, rangerID string) ([]*RangerPresence, error)
	CheckAlreadyPresence(ctx *gin.Context, rangerID string, agendaID string) (*RangerPresence, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreatePresence) (*RangerPresence, error)
	Show(ctx *gin.Context, id string) (*RangerPresence, error)
	Index(ctx *gin.Context) ([]*RangerPresence, error)
	CheckAlreadyPresence(ctx *gin.Context, rangerID string, agendaID string) (*RangerPresence, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	Show(ctx *gin.Context)
	Index(ctx *gin.Context)
}
