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

// NewUserTicket membangun UserTicket lengkap dengan ID dan public ID-nya.
//
// Dipakai bersama oleh service ini dan order service, sehingga pembuatan
// tiket lewat checkout dan lewat endpoint lain tidak menyimpang.
func NewUserTicket(req CreateUserTicket) *UserTicket {
	userTicket := &UserTicket{}
	userTicket.ID = uuid.NewString()

	publicID := strings.Split(uuid.NewString(), "-")[0]

	userTicket.PublicID = strings.ToUpper(publicID)
	userTicket.UserName = req.UserName
	userTicket.UserEmail = req.UserEmail
	userTicket.UserGender = req.UserGender
	userTicket.UserID = req.UserID
	userTicket.ParticipantUserID = req.ParticipantUserID
	userTicket.OrderID = req.OrderID
	userTicket.TicketID = req.TicketID
	userTicket.EventID = req.EventID

	userTicket.CreatedAt = time.Now()
	userTicket.UpdatedAt = time.Now()
	return userTicket
}

// Register implements Service
func (s *service) Create(c *gin.Context, req *CreateUserTicket) (*UserTicket, error) {
	userTicket := NewUserTicket(*req)

	userTicket, err := s.Repository.Create(c, userTicket)

	if err != nil {
		return nil, err
	}
	return userTicket, nil
}

func (s *service) CountByTicketID(c *gin.Context, ticket_id string) (int64, error) {
	return s.Repository.CountByTicketID(c, ticket_id)
}

func (s *service) ClaimTicketsByEmail(c *gin.Context, user_id string, email string) (int64, error) {
	return s.Repository.ClaimByEmail(c, user_id, email)
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
	ticket, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

func (s *service) ShowByPublicID(c *gin.Context, public_id string) (*UserTicket, error) {
	ticket, err := s.Repository.ShowByPublicID(c, public_id)
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

func (s *service) Index(c *gin.Context) ([]*UserTicket, error) {
	tickets, err := s.Repository.Index(c)
	if err != nil {
		return nil, err
	}
	return tickets, nil
}

func (s *service) IndexByOrderID(c *gin.Context, order_id string) ([]*UserTicket, error) {
	tickets, err := s.Repository.IndexByOrderID(c, order_id)
	if err != nil {
		return nil, err
	}
	return tickets, nil
}
