package poll

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

func (r *repository) Create(c *gin.Context, poll *Poll) (*Poll, error) {
	err := r.db.Create(poll).Error
	if err != nil {
		return nil, err
	}
	return poll, nil
}

func (r *repository) IndexByEventID(c *gin.Context, eventID string) ([]*Poll, error) {
	var polls []*Poll
	err := r.db.Where("event_id = ? AND deleted_at IS NULL", eventID).
		Preload("Options", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL").Order("order_index ASC")
		}).
		Order("order_index ASC, created_at ASC").
		Find(&polls).Error
	if err != nil {
		return nil, err
	}
	return polls, nil
}

func (r *repository) Show(c *gin.Context, id string) (*Poll, error) {
	poll := &Poll{}
	err := r.db.Where("id = ? AND deleted_at IS NULL", id).
		Preload("Options", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL").Order("order_index ASC")
		}).
		First(poll).Error
	if err != nil {
		return nil, err
	}
	return poll, nil
}

func (r *repository) Update(c *gin.Context, id string, poll *Poll) (*Poll, error) {
	err := r.db.Where("id = ?", id).Updates(poll).Error
	if err != nil {
		return nil, err
	}
	return r.Show(c, id)
}

func (r *repository) UpdateStatus(c *gin.Context, id string, status string) error {
	return r.db.Model(&Poll{}).Where("id = ?", id).Update("status", status).Error
}

func (r *repository) Delete(c *gin.Context, id string) error {
	return r.db.Where("id = ?", id).Delete(&Poll{}).Error
}

func (r *repository) SaveResponse(c *gin.Context, response *PollResponse) (*PollResponse, error) {
	err := r.db.Create(response).Error
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (r *repository) GetResponses(c *gin.Context, pollID string) ([]PollResponse, error) {
	var responses []PollResponse
	err := r.db.Where("poll_id = ?", pollID).Order("created_at ASC").Find(&responses).Error
	if err != nil {
		return nil, err
	}
	return responses, nil
}

func (r *repository) DeleteOptionsByPollID(c *gin.Context, pollID string) error {
	return r.db.Where("poll_id = ?", pollID).Delete(&PollOption{}).Error
}

func (r *repository) ActiveByEventID(c *gin.Context, eventID string) (*Poll, error) {
	poll := &Poll{}
	err := r.db.Where("event_id = ? AND status = ? AND deleted_at IS NULL", eventID, "active").
		Preload("Options", func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL").Order("order_index ASC")
		}).
		First(poll).Error
	if err != nil {
		return nil, err
	}
	return poll, nil
}
