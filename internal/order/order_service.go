package order

import (
	"errors"
	"mainyuk/internal/event"
	"mainyuk/internal/payment_method"
	"mainyuk/internal/ticket"
	"mainyuk/internal/user"
	"mainyuk/internal/user_ticket"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type service struct {
	Repository
	TicketService     ticket.Service
	UserTicketService user_ticket.Service
	EventService      event.Service
	PaymentMethod     payment_method.Service
}

func NewService(repository Repository, ticketService ticket.Service, userTicketService user_ticket.Service, eventService event.Service, pmService payment_method.Service) Service {
	return &service{
		repository,
		ticketService,
		userTicketService,
		eventService,
		pmService,
	}
}

// Register implements Service
func (s *service) Create(c *gin.Context, req *CreateOrder) (*Order, error) {
	order := &Order{}
	event, err := s.EventService.Show(c, req.EventID)
	if err != nil {
		return nil, errors.New("event not found")
	}
	order.EventID = event.ID
	order.Event = event
	if req.PaymentMethodID != "" {
		paymentMethod, err := s.PaymentMethod.Show(c, req.PaymentMethodID)
		if err != nil {
			return nil, errors.New("payment method not found")
		}
		order.PaymentMethodID = &paymentMethod.ID
		order.PaymentMethod = paymentMethod
	}

	order.ID = uuid.NewString()
	publicID := strings.Split(uuid.NewString(), "-")[0]

	order.PublicID = strings.ToUpper(publicID)

	user, errUserID := s.GetUserIDAuth(c)
	if errUserID != nil {
		return nil, errUserID
	}
	order.UserID = user.ID
	order.User = user

	var totalAmount int = 0
	for _, userTicket := range req.UserTickets {
		t, err := s.TicketService.Show(c, userTicket.TicketID)
		if err != nil {
			return nil, err
		}
		totalAmount += t.Price
	}
	order.Donation = 0
	if req.Donation != nil && (*req.Donation >= 0) {
		order.Donation = *req.Donation
	}

	// Payment gateway fee or unique code
	order.AdminFee = 0
	if req.AdminFee != nil && (*req.AdminFee >= 0) {
		order.AdminFee = *req.AdminFee
	}
	order.Amount = totalAmount
	order.Status = "pending"
	if order.Amount == 0 {
		order.Status = "paid"
	}

	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	order, err = s.Repository.Create(c, order)
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
		ticket.EventID = t.EventID
		ticket.UserID = order.UserID
		ticket.OrderID = order.ID

		nt, err := s.UserTicketService.Create(c, &ticket)
		if err != nil {
			return nil, err
		}
		order.UserTickets = append(order.UserTickets, nt)
	}

	return order, nil
}

func (s *service) Show(c *gin.Context, id string) (*Order, error) {
	order, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	tickets, err := s.UserTicketService.IndexByOrderID(c, order.ID)
	if err != nil {
		return nil, err
	}
	order.UserTickets = tickets
	return order, nil
}

func (s *service) ShowByPublicID(c *gin.Context, public_id string) (*Order, error) {
	user, errUserID := s.GetUserIDAuth(c)
	if errUserID != nil {
		return nil, errUserID
	}
	order, err := s.Repository.ShowByPublicID(c, public_id, &user.ID)
	if err != nil {
		return nil, err
	}
	tickets, err := s.UserTicketService.IndexByOrderID(c, order.ID)
	if err != nil {
		return nil, err
	}
	order.UserTickets = tickets
	return order, nil
}

func (s *service) Index(c *gin.Context) ([]*Order, error) {
	user, errUserID := s.GetUserIDAuth(c)
	if errUserID != nil {
		return nil, errUserID
	}
	order, err := s.Repository.Index(c, &user.ID)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *service) IndexAdmin(c *gin.Context) ([]*Order, error) {
	order, err := s.Repository.Index(c, nil)
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (s *service) GetUserIDAuth(c *gin.Context) (*user.User, error) {
	u, exists := c.Get("currentUser")
	if !exists {
		return nil, errors.New("not authrized")
	}

	currentUser, ok := u.(user.User)

	if !ok {
		return nil, errors.New("FailedParsing: current user")
	}
	return &currentUser, nil
}

func (s *service) VerifyOrder(c *gin.Context, id string, status string) (*Order, error) {
	order, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	order.Status = strings.ToLower(status)
	order.UpdatedAt = time.Now()

	order, err = s.Repository.Update(c, order)
	if err != nil {
		return nil, err
	}
	return order, nil
}
