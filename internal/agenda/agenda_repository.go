package agenda

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

func (r *repository) Create(c *gin.Context, agenda *Agenda) (*Agenda, error) {
	err := r.db.Create(agenda).Error
	if err != nil {
		return nil, err
	}
	return agenda, nil
}

func (r *repository) Update(c *gin.Context, id string, agenda *Agenda) (*Agenda, error) {
	err := r.db.Where("id = ?", id).Updates(agenda).Error
	if err != nil {
		return nil, err
	}
	return agenda, nil
}

func (r *repository) Show(c *gin.Context, id string) (*Agenda, error) {
	agenda := &Agenda{}
	err := r.db.Preload("User").Preload("Divisi").Where("id = ?", id).First(&agenda).Error
	if err != nil {
		return nil, err
	}
	return agenda, nil
}

func (r *repository) Index(c *gin.Context) ([]*Agenda, error) {
	var agendas []*Agenda
	tx := r.db
	query := tx.Model(&Agenda{})
	eventID := c.Query("divisi_id")

	if eventID != "" {
		query.Where("divisi_id = ?", eventID)
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

	err := query.Preload("User").Preload("Divisi").Where("deleted_at is NULL").Order("created_at DESC").Find(&agendas).Error
	if err != nil {
		return nil, err
	}
	return agendas, nil
}

func (r *repository) Delete(c *gin.Context, id string) error {
	agenda := &Agenda{}
	err := r.db.Where("id = ?", id).Delete(&agenda).Error
	if err != nil {
		return err
	}
	return nil
}
