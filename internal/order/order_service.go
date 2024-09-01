package order

import (
	"errors"
	"mainyuk/internal/ticket"
	"mainyuk/internal/user"
	"mainyuk/internal/user_ticket"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type service struct {
	Repository
	TicketService     ticket.Service
	UserTicketService user_ticket.Service
}

func NewService(repository Repository, ticketService ticket.Service, userTicketService user_ticket.Service) Service {
	return &service{
		repository,
		ticketService,
		userTicketService,
	}
}

// Register implements Service
func (s *service) Create(c *gin.Context, req *CreateOrder) (*Order, error) {
	order := &Order{}
	order.ID = uuid.NewString()
	order.PublicID = uuid.NewString()[:8]

	userID, errUserID := s.GetUserIDAuth(c)
	if errUserID != nil {
		return nil, errUserID
	}
	order.UserID = userID

	var totalAmount int = 0
	for _, userTicket := range req.UserTickets {
		t, err := s.TicketService.Show(c, userTicket.TicketID)
		if err != nil {
			return nil, err
		}
		totalAmount += t.Price
	}

	order.Amount = totalAmount

	order.Donation = 0
	if req.Donation != nil {
		order.Donation = *req.Donation
	}

	// Payment gateway fee or unique code
	order.AdminFee = 0
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	order, err := s.Repository.Create(c, order)
	if err != nil {
		return nil, err
	}

	// Create userTickets
	for _, t := range req.UserTickets {
		ticket := user_ticket.CreateUserTicket{}
		ticket.UserName = t.UserName
		ticket.UserEmail = t.UserEmail
		ticket.UserGender = t.UserGender
		ticket.TicketID = t.TicketID
		ticket.UserID = ticket.OrderID
		ticket.OrderID = order.ID

		_, err := s.UserTicketService.Create(c, &ticket)
		if err != nil {
			return nil, err
		}
	}

	return order, nil
}

func (s *service) Show(c *gin.Context, id string) (*Order, error) {
	order, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *service) ShowByPublicID(c *gin.Context, id string) (*Order, error) {
	order, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *service) Index(c *gin.Context) ([]*Order, error) {
	order, err := s.Repository.Index(c)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *service) GetUserIDAuth(c *gin.Context) (string, error) {
	u, exists := c.Get("currentUser")
	if !exists {
		return "", errors.New("NotAuthrized")
	}

	currentUser, ok := u.(user.User)

	if !ok {
		return "", errors.New("FailedParsing: current user")
	}
	return currentUser.ID, nil
}
