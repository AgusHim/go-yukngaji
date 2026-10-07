package otp

import (
	"errors"
	"strings"
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

func (r *repository) Create(c *gin.Context, otp *Otp) (*Otp, error) {
	err := r.db.Create(&otp).Error
	if err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *repository) ShowActive(c *gin.Context, email string) (*Otp, error) {
	otp := &Otp{}
	err := r.db.
		Where("lower(email) = ?", strings.ToLower(strings.TrimSpace(email))).
		Where("used_at IS NULL").
		Order("created_at DESC").
		First(&otp).Error
	if err != nil {
		return nil, err
	}
	return otp, nil
}

func (r *repository) MarkUsed(c *gin.Context, id string) (bool, error) {
	now := time.Now()
	res := r.db.Model(&Otp{}).
		Where("id = ? AND used_at IS NULL", id).
		Update("used_at", now)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *repository) IncrementAttempts(c *gin.Context, id string) error {
	return r.db.Model(&Otp{}).
		Where("id = ?", id).
		UpdateColumn("attempts", gorm.Expr("attempts + 1")).Error
}

func (r *repository) InvalidateActive(c *gin.Context, email string) error {
	now := time.Now()
	return r.db.Model(&Otp{}).
		Where("lower(email) = ? AND used_at IS NULL", strings.ToLower(strings.TrimSpace(email))).
		Update("used_at", now).Error
}

func (r *repository) TouchLastSent(c *gin.Context, id string) error {
	now := time.Now()
	res := r.db.Model(&Otp{}).Where("id = ?", id).Update("last_sent_at", now)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("otp not found")
	}
	return nil
}
