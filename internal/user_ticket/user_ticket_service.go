package user_ticket

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
func (s *service) Create(c *gin.Context, req *CreateUserTicket) (*UserTicket, error) {
	userTicket := &UserTicket{}
	userTicket.ID = uuid.NewString()
	userTicket.PublicID = strings.ToUpper(uuid.NewString()[:8])
	userTicket.UserName = req.UserName
	userTicket.UserEmail = req.UserEmail
	userTicket.UserGender = req.UserGender
	userTicket.UserID = req.UserID
	userTicket.OrderID = req.OrderID
	userTicket.TicketID = req.TicketID

	userTicket.CreatedAt = time.Now()
	userTicket.UpdatedAt = time.Now()

	userTicket, err := s.Repository.Create(c, userTicket)

	if err != nil {
		return nil, err
	}
	return userTicket, nil
}

func (s *service) Update(c *gin.Context, id string, req *CreateUserTicket) (*UserTicket, error) {
	userTicket, err := s.Show(c, id)
	if err != nil {
		return nil, err
	}

	userTicket.UserName = req.UserName
	userTicket.UserEmail = req.UserEmail
	userTicket.UserGender = req.UserGender

	userTicket.UpdatedAt = time.Now()

	updatedUserTicket, err := s.Repository.Update(c, userTicket.ID, userTicket)
	if err != nil {
		return nil, err
	}
	return updatedUserTicket, nil
}

func (s *service) Show(c *gin.Context, id string) (*UserTicket, error) {
	divisi, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return divisi, nil
}

func (s *service) Index(c *gin.Context) ([]*UserTicket, error) {
	divisi, err := s.Repository.Index(c)
	if err != nil {
		return nil, err
	}
	return divisi, nil
}
