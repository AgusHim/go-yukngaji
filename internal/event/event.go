package event

import (
	"mainyuk/internal/divisi"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
)

type Event struct {
	ID               string         `json:"id"`
	Slug             string         `json:"slug" `
	Code             string         `json:"code" `
	Title            string         `json:"title" binding:"required"`
	Desc             string         `json:"desc" binding:"required"`
	ImageUrl         string         `json:"image_url" binding:"required"`
	Speaker          string         `json:"speaker" binding:"required"`
	DivisiID         string         `json:"-" binding:"required"`
	Divisi           *divisi.Divisi `json:"divisi"`
	StartAt          time.Time      `json:"start_at" binding:"required"`
	EndAt            time.Time      `json:"end_at" binding:"required"`
	CloseAt          *time.Time     `json:"close_at"`
	Participant      int            `json:"participant"`
	IsPublished      bool           `json:"isPublished"`
	IsWhitelistOnly  bool           `json:"isWhitelistOnly"`
	AllowedGender    string         `json:"allowed_gender" binding:"required"`
	IsAllowedToOrder bool           `json:"isAllowedToOrder"`
	LocationTypes    pq.StringArray `json:"location_types" gorm:"type:text[]"`
	LocationDesc     pq.StringArray `json:"location_desc" gorm:"type:text[]"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"-"`
	DeletedAt        *time.Time     `json:"-"`
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

func CreateEventToEvent(value CreateEvent) (res *Event, err error) {
	event := Event{}
	event.Title = value.Title
	event.Desc = value.Desc
	event.ImageUrl = value.ImageUrl
	event.Speaker = value.Speaker
	event.DivisiID = value.DivisiID
	startAt, errParsed := time.Parse("2006-01-02T15:04", value.StartAt)
	if errParsed != nil {
		return nil, errParsed
	}
	endAt, errParsed := time.Parse("2006-01-02T15:04", value.EndAt)
	if errParsed != nil {
		return nil, errParsed
	}
	event.StartAt = startAt
	event.EndAt = endAt

	return &event, nil
}

type Repository interface {
	Create(ctx *gin.Context, event *Event) (*Event, error)
	Show(ctx *gin.Context, slug string) (*Event, error)
	ShowByCode(ctx *gin.Context, code string) (*Event, error)
	Index(ctx *gin.Context) ([]*Event, error)
	Update(ctx *gin.Context, id string, event *Event) (*Event, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateEvent) (*Event, error)
	Show(ctx *gin.Context, slug string) (*Event, error)
	ShowByCode(ctx *gin.Context, code string) (*Event, error)
	Index(ctx *gin.Context) ([]*Event, error)
	Update(ctx *gin.Context, id string, event *Event) (*Event, error)
}

type Handler interface {
	Update(ctx *gin.Context)
	Create(ctx *gin.Context)
	Show(ctx *gin.Context)
	ShowByCode(ctx *gin.Context)
	Index(ctx *gin.Context)
}
