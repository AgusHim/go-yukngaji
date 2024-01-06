package event

import (
	"errors"
	"fmt"
	"mainyuk/utils"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/gosimple/slug"
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
func (s *service) Create(c *gin.Context, req *CreateEvent) (*Event, error) {
	event := &Event{}
	event.ID = uuid.NewString()
	rand := uuid.New().String()[:5]
	slug := slug.Make(req.Title)
	combine := fmt.Sprintf("%s-%s", slug, rand)
	event.Slug = combine
	event.Code = utils.RandomToString(6)

	event.Title = req.Title
	event.Desc = req.Desc
	event.ImageUrl = req.ImageUrl
	event.Speaker = req.Speaker
	event.DivisiID = req.DivisiID
	event.Participant = 0
	event.StartAt = req.StartAt
	event.EndAt = req.EndAt

	event.CreatedAt = time.Now()
	event.UpdatedAt = time.Now()

	event, err := s.Repository.Create(c, event)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (s *service) Show(c *gin.Context, slug string) (*Event, error) {
	event, err := s.Repository.Show(c, slug)
	if err != nil {
		return nil, errors.New("EventNotFound")
	}
	return event, nil
}

func (s *service) ShowByCode(c *gin.Context, code string) (*Event, error) {
	event, err := s.Repository.ShowByCode(c, code)
	if err != nil {
		return nil, errors.New("EventNotFound")
	}
	return event, nil
}

func (s *service) Index(c *gin.Context) ([]*Event, error) {
	event, err := s.Repository.Index(c)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (s *service) Update(c *gin.Context, event *Event) (*Event, error) {
	event.UpdatedAt = time.Now()
	event, err := s.Repository.Update(c, event)
	if err != nil {
		return nil, err
	}
	return event, nil
}
