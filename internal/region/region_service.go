package region

import (
	"github.com/gin-gonic/gin"
)

type service struct {
	Repository
}

func NewService(repository Repository) Service {
	return &service{
		repository,
	}
}

func (s *service) Index(c *gin.Context) ([]*Region, error) {
	region, err := s.Repository.Index(c)
	if err != nil {
		return nil, err
	}
	return region, nil
}
