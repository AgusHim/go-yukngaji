package order

import (
	"strings"
	"time"

	"mainyuk/internal/user_ticket"

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

// CreateWithTickets menyimpan order dan tiket-tiketnya secara atomik.
// Bila salah satu tiket gagal disimpan, order pun dibatalkan sehingga tidak
// ada order yang tersisa tanpa tiket.
func (r *repository) CreateWithTickets(c *gin.Context, order *Order, tickets []*user_ticket.UserTicket) (*Order, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		for _, t := range tickets {
			if err := tx.Create(t).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *repository) Show(c *gin.Context, id string) (*Order, error) {
	order := &Order{}
	tx := r.db
	query := tx.Model(&order)
	err := query.Preload("User").Preload("Event").Preload("Event.Divisi").Preload("PaymentMethod").Where("id = ?", id).First(&order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *repository) ShowByPublicID(c *gin.Context, public_id string, user_id *string) (*Order, error) {
	order := &Order{}
	tx := r.db
	query := tx.Model(&order)
	if user_id != nil {
		query.Where("user_id = ?", user_id)
	}
	err := query.Preload("User").Preload("Event").Preload("Event.Divisi").Preload("PaymentMethod").Where("public_id = ?", public_id).First(&order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *repository) Index(c *gin.Context, user_id *string) ([]*Order, error) {
	var order []*Order
	tx := r.db
	query := tx.Model(&Order{})
	if user_id != nil && *user_id != "" {
		query.Where("user_id = ?", user_id)
	}
	// Filter status id
	status := c.Query("status")
	if status != "" {
		query.Where("status = ?", strings.ToLower(status))
	}
	// Filter event_id
	event_id := c.Query("event_id")
	if event_id != "" {
		query.Where("event_id = ?", event_id)
	}

	err := query.Preload("PaymentMethod").Preload("UserTickets").Preload("User").Preload("User.Province").Preload("User.District").Preload("User.SubDistrict").Preload("Event").Preload("Event.Divisi").Preload("PaymentMethod").Order("created_at DESC").Find(&order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

// Update hanya menulis kolom status. Sebelumnya dipakai Save yang menulis
// seluruh baris dan ikut menyentuh asosiasi (user/event) yang di-preload,
// sehingga berisiko menimpa data yang tidak dimaksud.
func (r *repository) Update(c *gin.Context, order *Order) (*Order, error) {
	err := r.db.Model(&Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"status":     order.Status,
		"updated_at": time.Now(),
	}).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

// TransitionStatus memindahkan status hanya bila status saat ini masih `from`.
//
// Syarat itu ada di klausa WHERE, bukan di kode Go, sehingga dua permintaan
// verifikasi yang datang bersamaan tidak bisa sama-sama lolos: yang kedua
// tidak menemukan baris dan melaporkan false.
func (r *repository) TransitionStatus(c *gin.Context, id string, from string, to string) (bool, error) {
	res := r.db.Model(&Order{}).
		Where("id = ?", id).
		Where("status = ?", from).
		Updates(map[string]interface{}{
			"status":     to,
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *repository) Participants(c *gin.Context, event_id string) ([]*Order, error) {
	var order []*Order
	tx := r.db
	query := tx.Model(&Order{})
	if event_id != "" {
		query.Where("event_id = ?", event_id)
	}
	err := query.Preload("UserTickets").Preload("User").Preload("User.Province").Preload("User.District").Preload("User.SubDistrict").Preload("Event").Preload("PaymentMethod").Where("status = ?", "paid").Order("created_at DESC").Find(&order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}
