package agenda

import (
	"errors"
	"mainyuk/internal/user"
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
func (s *service) Create(c *gin.Context, req *CreateAgenda) (*Agenda, error) {
	ranger := &Agenda{}
	ranger.ID = uuid.NewString()
	ranger.Name = req.Name
	ranger.Type = strings.ToLower(req.Type)
	ranger.Location = req.Location

	startAt, errParsed := time.Parse("2006-01-02T15:04", req.StartAt)
	if errParsed != nil {
		return nil, errParsed
	}
	ranger.Start_At = startAt

	ranger.DivisiID = req.DivisiID

	u, exists := c.Get("currentUser")
	if !exists {
		return nil, errors.New("NotAuthrized")
	}

	currentUser, ok := u.(user.User)

	if !ok {
		return nil, errors.New("FailedParsing: current user")
	}
	ranger.UserID = currentUser.ID

	ranger.CreatedAt = time.Now()
	ranger.UpdatedAt = time.Now()

	ranger, err := s.Repository.Create(c, ranger)
	if err != nil {
		return nil, err
	}
	return ranger, nil
}

func (s *service) Show(c *gin.Context, id string) (*Agenda, error) {
	event, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (s *service) Index(c *gin.Context) ([]*Agenda, error) {
	event, err := s.Repository.Index(c)
	if err != nil {
		return nil, err
	}
	return event, nil
}
