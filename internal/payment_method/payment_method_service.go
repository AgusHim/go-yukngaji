package payment_method

import (
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
func (s *service) Create(c *gin.Context, req *CreatePaymentMethod) (*PaymentMethod, error) {
	method := &PaymentMethod{}
	method.ID = uuid.NewString()
	method.Name = req.Name
	method.Type = req.Type
	method.Code = req.Code
	method.AccountName = req.AccountName
	method.AccountNumber = req.AccountNumber
	method.CreatedAt = time.Now()
	method.UpdatedAt = time.Now()

	method, err := s.Repository.Create(c, method)
	if err != nil {
		return nil, err
	}
	return method, nil
}

func (s *service) Show(c *gin.Context, id string) (*PaymentMethod, error) {
	divisi, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return divisi, nil
}

func (s *service) Index(c *gin.Context) ([]*PaymentMethod, error) {
	divisi, err := s.Repository.Index(c)
	if err != nil {
		return nil, err
	}
	return divisi, nil
}
