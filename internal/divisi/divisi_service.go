package divisi

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
func (s *service) Create(c *gin.Context, req *CreateDivisi) (*Divisi, error) {
	divisi := &Divisi{}
	divisi.ID = uuid.NewString()
	divisi.Name = req.Name
	divisi.Regional = req.Regional
	divisi.CreatedAt = time.Now()
	divisi.UpdatedAt = time.Now()

	divisi, err := s.Repository.Create(c, divisi)
	if err != nil {
		return nil, err
	}
	return divisi, nil
}

func (s *service) Show(c *gin.Context, id string) (*Divisi, error) {
	divisi, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return divisi, nil
}

func (s *service) Index(c *gin.Context) ([]*Divisi, error) {
	divisi, err := s.Repository.Index(c)
	if err != nil {
		return nil, err
	}
	return divisi, nil
}
