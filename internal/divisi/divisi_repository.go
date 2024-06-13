package divisi

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

func (r *repository) Create(c *gin.Context, divisi *Divisi) (*Divisi, error) {
	err := r.db.Create(divisi).Error
	if err != nil {
		return nil, err
	}
	return divisi, nil
}

func (r *repository) Show(c *gin.Context, id string) (*Divisi, error) {
	divisi := &Divisi{}
	err := r.db.Where("id = ?", id).First(&divisi).Error
	if err != nil {
		return nil, err
	}
	return divisi, nil
}

func (r *repository) Index(c *gin.Context) ([]*Divisi, error) {
	var divisi []*Divisi
	err := r.db.Find(&divisi).Error
	if err != nil {
		return nil, err
	}
	return divisi, nil
}
