package ticket

import (
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type service struct {
	Repository
}

func NewService(repository Repository) Service {
	return &service{
		repository,
	}
}

// Register implements Service
func (s *service) Create(c *gin.Context, req *CreateTicket) (*Ticket, error) {
	ticket := &Ticket{}
	ticket.ID = uuid.NewString()
	ticket.Name = req.Name
	ticket.Description = req.Description
	ticket.Price = req.Price
	ticket.EventID = req.EventID
	ticket.PaxMultiplier = req.PaxMultiplier
	ticket.MinOrderPax = req.MinOrderPax
	ticket.MaxOrderPax = req.MaxOrderPax
	ticket.MaxPax = req.MaxPax

	startAt, errParsed := time.Parse("2006-01-02T15:04", req.StartAt)
	if errParsed != nil {
		return nil, errParsed
	}
	endAt, errParsed := time.Parse("2006-01-02T15:04", req.EndAt)
	if errParsed != nil {
		return nil, errParsed
	}
	ticket.StartAt = startAt
	ticket.EndAt = endAt

	ticket.Visibility = "DRAFT"
	if req.Visibility != nil {
		ticket.Visibility = strings.ToUpper(*req.Visibility)
	}

	ticket.CreatedAt = time.Now()
	ticket.UpdatedAt = time.Now()

	ticket, err := s.Repository.Create(c, ticket)
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

func (s *service) Update(c *gin.Context, id string, req *CreateTicket) (*Ticket, error) {
	ticket, err := s.Show(c, id)
	if err != nil {
		return nil, err
	}

	ticket.Name = req.Name
	ticket.Description = req.Description
	ticket.Price = req.Price
	ticket.EventID = req.EventID
	ticket.PaxMultiplier = req.PaxMultiplier
	ticket.MinOrderPax = req.MinOrderPax
	ticket.MaxOrderPax = req.MaxOrderPax
	ticket.MaxPax = req.MaxPax

	startAt, errParsed := time.Parse("2006-01-02T15:04", req.StartAt)
	if errParsed != nil {
		return nil, errParsed
	}
	endAt, errParsed := time.Parse("2006-01-02T15:04", req.EndAt)
	if errParsed != nil {
		return nil, errParsed
	}
	ticket.StartAt = startAt
	ticket.EndAt = endAt

	if req.Visibility != nil {
		ticket.Visibility = strings.ToUpper(*req.Visibility)
	}

	ticket.UpdatedAt = time.Now()

	updatedTicket, err := s.Repository.Update(c, ticket.ID, ticket)
	if err != nil {
		return nil, err
	}
	return updatedTicket, nil
}

func (s *service) Show(c *gin.Context, id string) (*Ticket, error) {
	divisi, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return divisi, nil
}

func (s *service) Index(c *gin.Context) ([]*Ticket, error) {
	divisi, err := s.Repository.Index(c)
	if err != nil {
		return nil, err
	}
	return divisi, nil
}
