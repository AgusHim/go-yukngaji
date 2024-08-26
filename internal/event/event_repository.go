package event

import (
	"strconv"
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
	tx := r.db
	query := tx.Model(&Event{})

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

	var errConvert error
	limitStr := c.Query("limit")
	var limit int = 10

	if limitStr != "" {
		limit, errConvert = strconv.Atoi(limitStr)
		if errConvert != nil {
			return nil, errConvert
		}
		query.Limit(limit)
	}

	pageStr := c.Query("page")
	if pageStr != "" {
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			return nil, err
		}
		query.Offset((page - 1) * limit)
	}

	err := query.Where("deleted_at IS NULL").Preload("Divisi").Order("created_at DESC").Find(&events).Error
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
