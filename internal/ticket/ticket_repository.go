package ticket

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

func (r *repository) Create(c *gin.Context, ticket *Ticket) (*Ticket, error) {
	err := r.db.Create(ticket).Error
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

func (r *repository) Update(c *gin.Context, id string, ticket *Ticket) (*Ticket, error) {
	err := r.db.Create(ticket).Error
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

func (r *repository) Show(c *gin.Context, id string) (*Ticket, error) {
	ticket := &Ticket{}
	err := r.db.Where("id = ?", id).First(&ticket).Error
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

func (r *repository) Index(c *gin.Context) ([]*Ticket, error) {
	var tickets []*Ticket
	tx := r.db
	query := tx.Model(&Ticket{})
	eventID := c.Query("event_id")
	if eventID != "" {
		query.Where("event_id = ?", eventID)
	}

	err := query.Where("deleted_at IS NULL").Find(&tickets).Error
	if err != nil {
		return nil, err
	}
	return tickets, nil
}
