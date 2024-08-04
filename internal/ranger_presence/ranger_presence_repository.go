package ranger_presence

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

func (r *repository) Create(c *gin.Context, presence *RangerPresence) (*RangerPresence, error) {
	err := r.db.Create(presence).Error
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (r *repository) Show(c *gin.Context, id string) (*RangerPresence, error) {
	presence := &RangerPresence{}
	err := r.db.Preload("Ranger").Preload("Agenda").Preload("Divisi").Where("id = ?", id).First(&presence).Error
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (r *repository) CheckAlreadyPresence(c *gin.Context, rangerID string, agendaID string) (*RangerPresence, error) {
	presence := &RangerPresence{}
	err := r.db.Preload("Ranger").Preload("Divisi").Where("ranger_id = ?", rangerID).Where("agenda_id = ?", agendaID).First(&presence).Error
	if err != nil {
		return nil, err
	}
	return presence, nil
}

func (r *repository) Index(c *gin.Context) ([]*RangerPresence, error) {
	var rangers []*RangerPresence

	tx := r.db
	query := tx.Model(&RangerPresence{})
	divisiID := c.Query("divisi_id")

	if divisiID != "" {
		query.Where("divisi_id = ?", divisiID)
	}

	agendaID := c.Query("agenda_id")

	if agendaID != "" {
		query.Where("agenda_id = ?", agendaID)
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

	if divisiID != "" {
		query.Where("divisi_id = ?", divisiID)
	}
	query.Where("deleted_at is NULL").Order("created_at ASC")

	err := query.Preload("Ranger").Preload("Ranger.User").Preload("Ranger.Divisi").Preload("Agenda").Preload("Divisi").Find(&rangers).Error
	if err != nil {
		return nil, err
	}
	return rangers, nil
}

func (r *repository) IndexByUserID(c *gin.Context, rangerID string) ([]*RangerPresence, error) {
	var presence []*RangerPresence

	tx := r.db
	query := tx.Model(&RangerPresence{})

	if rangerID != "" {
		query.Where("ranger_id = ?", rangerID)
	}
	query.Where("deleted_at is NULL").Order("created_at ASC")

	err := query.Preload("Ranger").Preload("Ranger.User").Preload("Ranger.Divisi").Preload("Agenda").Preload("Divisi").Find(&presence).Error
	if err != nil {
		return nil, err
	}
	return presence, nil
}
