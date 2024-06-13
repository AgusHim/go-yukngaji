package ranger

import (
	"time"

	"github.com/gin-gonic/gin"
)

type Ranger struct {
	ID        string     `json:"id" binding:"required"`
	UserID    string     `json:"-" binding:"required"`
	User      *User      `json:"user"`
	DivisiID  string     `json:"-" binding:"required"`
	Divisi    *Divisi    `json:"divisi"`
	Present   *int       `json:"present" gorm:"-"`
	Absent    *int       `json:"absent" gorm:"-"`
	CreatedAt time.Time  `json:"created_at" binding:"required"`
	UpdatedAt time.Time  `json:"-" binding:"required"`
	DeletedAt *time.Time `json:"-" binding:"required"`
}

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Age      int    `json:"age"`
	Gender   string `json:"gender"`
	Role     string `json:"role"`
	Address  string `json:"address"`
	Activity string `json:"activity"`
}

type Divisi struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Regional string `json:"regional"`
}

func (Divisi) TableName() string {
	return "divisi"
}

type RangerPresence struct {
	ID       string `json:"id"`
	RangerID string `json:"ranger_id"`
	DivisiID string `json:"divisi_id"`
}

type Agenda struct {
	ID string `json:"id"`
}

type CreateRanger struct {
	UserID   string `json:"user_id" binding:"required"`
	DivisiID string `json:"divisi_id" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, event *Ranger) (*Ranger, error)
	Show(ctx *gin.Context, id string) (*Ranger, error)
	ShowByUserID(ctx *gin.Context, userID string) (*Ranger, error)
	Index(ctx *gin.Context) ([]*Ranger, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateRanger) (*Ranger, error)
	Show(ctx *gin.Context, id string) (*Ranger, error)
	ShowByUserID(ctx *gin.Context, userID string) (*Ranger, error)
	Index(ctx *gin.Context) ([]*Ranger, error)
}

type Handler interface {
	Create(ctx *gin.Context)
	Show(ctx *gin.Context)
	Index(ctx *gin.Context)
}
