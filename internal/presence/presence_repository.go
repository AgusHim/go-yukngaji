package presence

import (
	"errors"
	"mainyuk/internal/user"
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

func (r *repository) Create(c *gin.Context, presence *Presence) (*Presence, error) {
	err := r.db.Preload("User").Preload("Event").Create(&presence).Error
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (r *repository) Show(c *gin.Context, id string) (*Presence, error) {
	presence := &Presence{}
	err := r.db.Preload("User").Preload("Event").Where("id = ?", id).Where("deleted_at IS NULL").First(&presence).Error
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (r *repository) FindByUserID(c *gin.Context, id string, eventID string) (*Presence, error) {
	presence := &Presence{}
	err := r.db.Preload("User").Preload("Event").Where("user_id = ?", id).Where("event_id = ?", eventID).Where("deleted_at IS NULL").First(&presence).Error
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (r *repository) FindByUserTicketID(c *gin.Context, id string, eventID string) (*Presence, error) {
	presence := &Presence{}
	err := r.db.Preload("User").Preload("Event").Where("user_ticket_id = ?", id).Where("event_id = ?", eventID).Where("deleted_at IS NULL").First(&presence).Error
	if err != nil {
		return nil, err
	}
	return presence, nil
}

// FindByUserTicketAndDate mencari baris check-in tiket pada satu tanggal.
// Ini jalur cepat idempotensi; unique index tetap penjamin akhirnya.
func (r *repository) FindByUserTicketAndDate(c *gin.Context, userTicketID string, checkInDate time.Time) (*Presence, error) {
	presence := &Presence{}
	err := r.db.Preload("User").Preload("Event").
		Where("user_ticket_id = ?", userTicketID).
		Where("check_in_date = ?", checkInDate).
		Where("deleted_at IS NULL").
		First(&presence).Error
	if err != nil {
		return nil, err
	}
	return presence, nil
}

// CountByUserTicket menghitung semua check-in tiket, dipakai untuk menentukan
// apakah ini check-in pertama (baru menaikkan event.participant).
func (r *repository) CountByUserTicket(c *gin.Context, userTicketID string) (int64, error) {
	var count int64
	err := r.db.Model(&Presence{}).
		Where("user_ticket_id = ?", userTicketID).
		Where("deleted_at IS NULL").
		Count(&count).Error
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (r *repository) Index(c *gin.Context) ([]*Presence, error) {
	var presences []*Presence
	tx := r.db
	query := tx.Model(&presences)
	eventID := c.Query("event_id")

	if eventID != "" {
		query.Where("event_id = ?", eventID)
	}

	if strings.Contains(c.FullPath(), "user_api/presence") {
		currentUser, ok := user.FromContext(c)
		if !ok {
			return nil, errors.New("NotAuthrized")
		}

		query.Where("user_id = ?", currentUser.ID)
	}

	err := query.Preload("User").Preload("User.Province").Preload("User.District").Preload("User.SubDistrict").Preload("Event").Preload("UserTicket").Preload("UserTicket.Ticket").Where("deleted_at IS NULL").Find(&presences).Error
	if err != nil {
		return nil, err
	}

	return presences, nil
}

func (r *repository) IndexByUserTicket(c *gin.Context, user_ticket_id string) ([]*Presence, error) {
	var presences []*Presence
	tx := r.db
	query := tx.Model(&presences)
	query.Where("user_ticket_id = ?", user_ticket_id)
	err := query.Preload("User").Preload("Event").Preload("UserTicket").Preload("UserTicket.Ticket").Where("deleted_at IS NULL").Find(&presences).Error
	if err != nil {
		return nil, err
	}

	return presences, nil
}
