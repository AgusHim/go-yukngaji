package presence

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

func (r *repository) Create(c *gin.Context, presence *Presence) (*Presence, error) {
	err := r.db.Preload("User").Preload("Event").Create(presence).Error
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (r *repository) Show(c *gin.Context, id string) (*Presence, error) {
	presence := &Presence{}
	err := r.db.Preload("User").Preload("Event").Where("id = ?", id).Where("deleted_at IS NULL").First(&presence).Error
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (r *repository) FindByUserID(c *gin.Context, id string, eventID string) (*Presence, error) {
	presence := &Presence{}
	err := r.db.Preload("User").Preload("Event").Where("user_id = ?", id).Where("event_id = ?",eventID).Where("deleted_at IS NULL").First(&presence).Error
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (r *repository) Index(c *gin.Context) ([]*Presence, error) {
	var presences []*Presence
	tx := r.db
	query := tx.Model(&presences)
	eventID := c.Query("event_id")

	if eventID != "" {
		query.Where("event_id = ?", eventID)
	}
	err := query.Preload("User").Preload("Event").Where("deleted_at IS NULL").Find(&presences).Error
	if err != nil {
		return nil, err
	}

	return presences, nil
}
