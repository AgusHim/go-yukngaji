package feedback

import (
	"time"

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

func (r *repository) Create(c *gin.Context, feedback *Feedback) (*Feedback, error) {
	err := r.db.Create(feedback).Error
	if err != nil {
		return nil, err
	}
	return feedback, nil
}

func (r *repository) Index(c *gin.Context) ([]*Feedback, error) {
	var feedback []*Feedback

	tx := r.db
	query := tx.Model(&Feedback{})
	eventID := c.Query("event_id")

	if eventID != "" {
		query.Where("event_id = ?", eventID)
	}

	startAt := c.Query("start_at")
	endAt := c.Query("end_at")
	if startAt != "" && endAt != "" {
		start, errParsed := time.Parse("02-01-2006", startAt)
		if errParsed != nil {
			return nil, errParsed
		}
		end, errParsed := time.Parse("02-01-2006", endAt)
		if errParsed != nil {
			return nil, errParsed
		}
		query.Where("created_at BETWEEN ? AND ?", start, end)
	}

	err := query.Preload("User").Preload("Event").Where("deleted_at is NULL").Order("created_at ASC").Find(&feedback).Error
	if err != nil {
		return nil, err
	}
	return feedback, nil
}
