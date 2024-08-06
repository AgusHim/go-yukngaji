package ranger

import (
	"mainyuk/internal/agenda"
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

func (r *repository) Create(c *gin.Context, ranger *Ranger) (*Ranger, error) {
	err := r.db.Create(ranger).Error
	if err != nil {
		return nil, err
	}
	return ranger, nil
}

func (r *repository) Show(c *gin.Context, id string) (*Ranger, error) {
	ranger := &Ranger{}
	err := r.db.Preload("User").Preload("Divisi").Where("id = ?", id).Where("deleted_at is NULL").First(&ranger).Error
	if err != nil {
		return nil, err
	}
	return ranger, nil
}

func (r *repository) ShowByUserID(c *gin.Context, userID string) (*Ranger, error) {
	ranger := &Ranger{}
	err := r.db.Preload("User").Preload("Divisi").Where("user_id = ?", userID).Where("deleted_at is NULL").First(&ranger).Error
	if err != nil {
		return nil, err
	}
	return ranger, nil
}

func (r *repository) Index(c *gin.Context) ([]*Ranger, error) {
	var rangers []*Ranger

	tx := r.db
	query := tx.Model(&Ranger{})
	divisiID := c.Query("divisi_id")

	if divisiID != "" {
		query.Where("divisi_id = ?", divisiID)
	}

	err := query.Preload("User").Preload("Divisi").Where("deleted_at is NULL").Order("created_at ASC").Find(&rangers).Error
	if err != nil {
		return nil, err
	}

	for _, ranger := range rangers {

		ranger.Present, err = r.CountPresentNonDivisi(c, ranger)
		if err != nil {
			return nil, err
		}
		ranger.PresentDivisi, err = r.CountPresentOnDivisi(c, ranger)
		if err != nil {
			return nil, err
		}

		totalAgenda, err := r.CountAgenda(c, ranger)
		if err != nil {
			return nil, err
		}
		absent := 0
		if *totalAgenda >= *ranger.PresentDivisi {
			absent = *totalAgenda - *ranger.PresentDivisi
		}
		ranger.AbsentDivisi = &absent
	}

	return rangers, nil
}

func (r *repository) Update(c *gin.Context, id string, ranger *Ranger) (*Ranger, error) {
	err := r.db.Where("id = ?", id).Updates(&ranger).Error
	if err != nil {
		return nil, err
	}
	return ranger, nil
}

func (r *repository) Delete(c *gin.Context, id string, ranger *Ranger) error {
	err := r.db.Where("id = ?", id).Updates(&ranger).Error
	if err != nil {
		return err
	}
	return nil
}

func (r *repository) CountAgenda(c *gin.Context, ranger *Ranger) (*int, error) {
	tx := r.db
	query := tx.Model(&agenda.Agenda{})
	query.Where("divisi_id = ?", ranger.DivisiID)

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

	var totalAgenda int64
	query.Count(&totalAgenda)
	total := int(totalAgenda)
	return &total, nil
}

func (r *repository) CountPresentOnDivisi(c *gin.Context, ranger *Ranger) (*int, error) {
	tx := r.db
	query := tx.Model(&RangerPresence{})
	var countPresent int64

	query.Where("ranger_id = ?", ranger.ID)
	query.Where("divisi_id = ?", ranger.DivisiID)

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

	query.Count(&countPresent)
	count := int(countPresent)
	return &count, nil
}

func (r *repository) CountPresentNonDivisi(c *gin.Context, ranger *Ranger) (*int, error) {
	tx := r.db
	query := tx.Model(&RangerPresence{})
	var countPresent int64

	query.Where("ranger_id = ?", ranger.ID)
	query.Where("divisi_id != ?", ranger.DivisiID)

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

	query.Count(&countPresent)
	count := int(countPresent)
	return &count, nil
}
