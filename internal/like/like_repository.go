package like

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

func (r *repository) Create(c *gin.Context, like *Like) (*Like, error) {
	err := r.db.Create(like).Error
	if err != nil {
		return nil, err
	}
	return like, nil
}

func (r *repository) Show(c *gin.Context, id string) (*Like, error) {
	like := &Like{}
	err := r.db.Where("id = ?", id).First(like).Error
	if err != nil {
		return nil, err
	}
	return like, nil
}

func (r *repository) Delete(c *gin.Context, id string) error {
	tx := r.db
	query := tx.Model(&Like{})
	err := query.Where("id = ?", id).Delete(&Like{}).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) Index(c *gin.Context) ([]*Like, error) {
	var likes []*Like
	tx := r.db
	query := tx.Model(&Like{})
	eventID := c.Query("event_id")
	userID := c.Query("user_id")

	if eventID != "" && userID != "" {
		query.Where("event_id = ?", eventID).Where("user_id = ?", userID)
	}
	err := query.Find(&likes).Error
	if err != nil {
		return nil, err
	}
	return likes, nil
}
