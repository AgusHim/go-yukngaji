package user_ticket

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

func (r *repository) Create(c *gin.Context, userTicket *UserTicket) (*UserTicket, error) {
	err := r.db.Preload("User").Preload("Ticket").Preload("Event").Create(userTicket).Error
	if err != nil {
		return nil, err
	}
	return userTicket, nil
}

func (r *repository) Update(c *gin.Context, id string, userTicket *UserTicket) (*UserTicket, error) {
	err := r.db.Create(&userTicket).Error
	if err != nil {
		return nil, err
	}
	return userTicket, nil
}

func (r *repository) Show(c *gin.Context, id string) (*UserTicket, error) {
	userTicket := &UserTicket{}
	err := r.db.Preload("Ticket").Preload("Event").Preload("User").Where("id = ?", id).First(&userTicket).Error
	if err != nil {
		return nil, err
	}
	return userTicket, nil
}

func (r *repository) ShowByPublicID(c *gin.Context, id string) (*UserTicket, error) {
	userTicket := &UserTicket{}
	err := r.db.Preload("Ticket").Preload("Event").Preload("User").Where("public_id = ?", id).First(&userTicket).Error
	if err != nil {
		return nil, err
	}
	return userTicket, nil
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
