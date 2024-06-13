package agenda

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

func (r *repository) Create(c *gin.Context, agenda *Agenda) (*Agenda, error) {
	err := r.db.Create(agenda).Error
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

	err := query.Preload("User").Preload("Divisi").Where("deleted_at is NULL").Order("created_at ASC").Find(&agendas).Error
	if err != nil {
		return nil, err
	}
	return agendas, nil
}
