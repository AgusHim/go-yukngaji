package comment

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

func (r *repository) Create(c *gin.Context, comment *Comment) (*Comment, error) {
	err := r.db.Create(comment).Error
	if err != nil {
		return nil, err
	}
	return comment, nil
}

func (r *repository) Show(c *gin.Context, id string) (*Comment, error) {
	comment := &Comment{}
	err := r.db.Where("id = ?", id).First(&comment).Error
	if err != nil {
		return nil, err
	}
	return comment, nil
}

func (r *repository) Index(c *gin.Context) ([]*Comment, error) {
	var comments []*Comment
	tx := r.db
	query := tx.Model(&Comment{})
	eventID := c.Query("event_id")

	if eventID != "" {
		query.Where("event_id = ?", eventID)
	}
	err := query.Preload("User").Where("deleted_at is NULL").Order("created_at ASC").Find(&comments).Error
	if err != nil {
		return nil, err
	}
	return comments, nil
}

func (r *repository) Update(c *gin.Context, comment *Comment) (*Comment, error) {
	err := r.db.Save(&comment).Error
	if err != nil {
		return nil, err
	}
	return comment, nil
}
