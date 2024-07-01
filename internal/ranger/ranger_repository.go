package ranger

import (
	"mainyuk/internal/agenda"

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
	err := r.db.Preload("User").Preload("Divisi").Where("user_id = ?", userID).First(&ranger).Error
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

	queryAgenda := tx.Model(&agenda.Agenda{})
	if divisiID != "" {
		queryAgenda.Where("divisi_id = ?", divisiID)
	}
	var totalAgenda int64
	queryAgenda.Count(&totalAgenda)

	for _, ranger := range rangers {
		queryPresence := tx.Model(&RangerPresence{})
		var countPresent int64
		queryPresence.Where("ranger_id = ?", ranger.ID)

		if divisiID != "" {
			queryPresence.Where("divisi_id = ?", divisiID)
		}
		queryPresence.Count(&countPresent)
		count := int(countPresent)
		ranger.Present = &count

		absent := int(totalAgenda) - int(countPresent)
		ranger.Absent = &absent
	}

	return rangers, nil
}

func (r *repository) Update(c *gin.Context, id string, ranger *Ranger) (*Ranger, error) {
	err := r.db.Where("id = ?", id).Updates(ranger).Error
	if err != nil {
		return nil, err
	}
	return ranger, nil
}

func (r *repository) Delete(c *gin.Context, id string) error {
	ranger := &Ranger{}
	err := r.db.Where("id = ?", id).Updates(&ranger).Error
	if err != nil {
		return err
	}
	return nil
}
