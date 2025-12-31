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
	err := r.db.
		Model(&Ticket{}).
		Where("id = ?", id).
		Omit("sold_pax").
		Updates(ticket).Error

	if err != nil {
		return nil, err
	}

	return ticket, nil
}

func (r *repository) Show(c *gin.Context, id string) (*Ticket, error) {
	ticket := &Ticket{}
	err := r.db.Model(&Ticket{}).
		Select("tickets.*, (SELECT COUNT(ut.id) FROM user_tickets ut JOIN orders o ON o.id = ut.order_id WHERE ut.ticket_id = tickets.id AND o.status IN ('pending', 'paid') AND o.deleted_at IS NULL AND ut.deleted_at IS NULL) as sold_pax").
		Where("id = ?", id).Where("deleted_at IS NULL").First(&ticket).Error
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

	query.Select("tickets.*, (SELECT COUNT(ut.id) FROM user_tickets ut JOIN orders o ON o.id = ut.order_id WHERE ut.ticket_id = tickets.id AND o.status IN ('pending', 'paid') AND o.deleted_at IS NULL AND ut.deleted_at IS NULL) as sold_pax")

	err := query.Where("deleted_at IS NULL").Find(&tickets).Error
	if err != nil {
		return nil, err
	}
	return tickets, nil
}

func (r *repository) Delete(c *gin.Context, id string) error {
	ticket := &Ticket{}
	err := r.db.Where("id = ?", id).Delete(&ticket).Error
	if err != nil {
		return err
	}
	return nil
}
