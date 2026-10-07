package user_ticket

import (
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

func (r *repository) Create(c *gin.Context, userTicket *UserTicket) (*UserTicket, error) {
	err := r.db.Preload("User").Preload("Ticket").Preload("Event").Create(userTicket).Error
	if err != nil {
		return nil, err
	}
	return userTicket, nil
}

// Update hanya menulis kolom yang boleh diubah. Sebelumnya memakai Create
// sehingga "update" justru mencoba menyisipkan baris baru.
func (r *repository) Update(c *gin.Context, id string, userTicket *UserTicket) (*UserTicket, error) {
	err := r.db.Model(&UserTicket{}).Where("id = ?", id).Updates(map[string]interface{}{
		"user_name":   userTicket.UserName,
		"user_email":  userTicket.UserEmail,
		"user_gender": userTicket.UserGender,
		"updated_at":  time.Now(),
	}).Error
	if err != nil {
		return nil, err
	}
	return r.Show(c, id)
}

func (r *repository) Show(c *gin.Context, id string) (*UserTicket, error) {
	userTicket := &UserTicket{}
	err := r.db.Preload("Ticket").Preload("Event").Preload("User").Preload("Participant").Where("id = ?", id).First(&userTicket).Error
	if err != nil {
		return nil, err
	}
	return userTicket, nil
}

func (r *repository) ShowByPublicID(c *gin.Context, id string) (*UserTicket, error) {
	userTicket := &UserTicket{}
	// Order ikut dimuat: check-in memakai status pembayaran sebagai syarat.
	err := r.db.Preload("Ticket").Preload("Event").Preload("User").Preload("Participant").Preload("Order").Where("public_id = ?", id).First(&userTicket).Error
	if err != nil {
		return nil, err
	}
	return userTicket, nil
}

func (r *repository) CountByTicketID(c *gin.Context, ticket_id string) (int64, error) {
	var count int64
	err := r.db.Model(&UserTicket{}).
		Where("ticket_id = ?", ticket_id).
		Where("deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ClaimByEmail mengaitkan tiket yang belum punya peserta ke akun dengan
// email yang sama. Hanya baris tanpa participant_user_id yang disentuh.
func (r *repository) ClaimByEmail(c *gin.Context, user_id string, email string) (int64, error) {
	res := r.db.Model(&UserTicket{}).
		Where("lower(user_email) = ?", strings.ToLower(strings.TrimSpace(email))).
		Where("participant_user_id IS NULL").
		Where("deleted_at IS NULL").
		Updates(map[string]interface{}{
			"participant_user_id": user_id,
			"updated_at":          time.Now(),
		})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

func (r *repository) Index(c *gin.Context) ([]*UserTicket, error) {
	var userTickets []*UserTicket
	tx := r.db
	query := tx.Model(&UserTicket{})

	event_id := c.Query("event_id")
	if event_id != "" {
		query.Joins("JOIN events ON events.id = user_tickets.event_id").Where("events.id = ?", event_id)
	}

	orderStatus := c.Query("order[status]")
	if orderStatus != "" {
		query.Joins("JOIN orders ON orders.id = user_tickets.order_id").Where("orders.status = ?", orderStatus)
	}

	err := query.Preload("Ticket").Preload("Event").Preload("Order").Preload("User").Preload("User.Province").Preload("User.District").Preload("User.SubDistrict").Find(&userTickets).Error
	if err != nil {
		return nil, err
	}
	return userTickets, nil
}

func (r *repository) IndexByOrderID(c *gin.Context, order_id string) ([]*UserTicket, error) {
	var userTickets []*UserTicket
	err := r.db.Where("order_id = ?", order_id).Preload("Ticket").Preload("Event").Preload("User").Find(&userTickets).Error
	if err != nil {
		return nil, err
	}
	return userTickets, nil
}
