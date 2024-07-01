package event

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) Create(c *gin.Context, event *Event) (*Event, error) {
	err := r.db.Preload("Divisi").Create(event).Error
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (r *repository) Show(c *gin.Context, slug string) (*Event, error) {
	event := &Event{}
	err := r.db.Preload("Divisi").Where("slug = ?", slug).Where("deleted_at IS NULL").First(&event).Error
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (r *repository) ShowByCode(c *gin.Context, code string) (*Event, error) {
	event := &Event{}
	err := r.db.Where("code = ?", code).Where("deleted_at IS NULL").Preload("Divisi").First(&event).Error
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (r *repository) Index(c *gin.Context) ([]*Event, error) {
	var events []*Event
	err := r.db.Where("deleted_at IS NULL").Preload("Divisi").Order("created_at DESC").Find(&events).Error
	if err != nil {
		return nil, err
	}
	return events, nil
}

func (r *repository) Update(c *gin.Context, id string, event *Event) (*Event, error) {
	err := r.db.Save(&event).Error
	if err != nil {
		return nil, err
	}
	return event, nil
}
