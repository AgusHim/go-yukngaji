package otp

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
func (r *repository) Create(c *gin.Context, otp *Otp) (*Otp, error) {
	err := r.db.Create(&otp).Error
	if err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *repository) Show(c *gin.Context, email *string, code *string) (*Otp, error) {
	otp := &Otp{}
	query := r.db
	if email != nil {
		query.Where("email = ?", email)
	}
	if code != nil {
		query.Where("code = ?", code)
	}
	err := query.Order("created_at DESC").First(&otp).Error
	if err != nil {
		return nil, err
	}
	return otp, nil
}
