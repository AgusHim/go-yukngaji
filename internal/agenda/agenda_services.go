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
func (s *service) Create(c *gin.Context, createAgenda *CreateAgenda) (*Agenda, error) {
	agenda := &Agenda{}
	agenda.ID = uuid.NewString()
	agenda.Name = createAgenda.Name
	agenda.Type = strings.ToLower(createAgenda.Type)
	agenda.Location = createAgenda.Location

	startAt, errParsed := time.Parse("2006-01-02T15:04", createAgenda.StartAt)
	if errParsed != nil {
		return nil, errParsed
	}
	agenda.Start_At = startAt

	agenda.DivisiID = createAgenda.DivisiID

	u, exists := c.Get("currentUser")
	if !exists {
		return nil, errors.New("NotAuthrized")
	}

	currentUser, ok := u.(user.User)

	if !ok {
		return nil, errors.New("FailedParsing: current user")
	}
	agenda.UserID = currentUser.ID

	agenda.CreatedAt = time.Now()
	agenda.UpdatedAt = time.Now()

	agenda, err := s.Repository.Create(c, agenda)
	if err != nil {
		return nil, err
	}
	return agenda, nil
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

func (s *service) Update(c *gin.Context, id string, createAgenda *CreateAgenda) (*Agenda, error) {
	_, errAgenda := s.Show(c, id)
	if errAgenda != nil {
		return nil, errAgenda
	}

	agenda := &Agenda{}
	agenda.Name = createAgenda.Name
	agenda.Type = strings.ToLower(createAgenda.Type)
	agenda.Location = createAgenda.Location

	startAt, errParsed := time.Parse("2006-01-02T15:04", createAgenda.StartAt)
	if errParsed != nil {
		return nil, errParsed
	}
	agenda.Start_At = startAt
	agenda.DivisiID = createAgenda.DivisiID

	u, exists := c.Get("currentUser")
	if !exists {
		return nil, errors.New("NotAuthrized")
	}

	currentUser, ok := u.(user.User)

	if !ok {
		return nil, errors.New("FailedParsing: current user")
	}
	agenda.UserID = currentUser.ID
	agenda.UpdatedAt = time.Now()

	agenda, err := s.Repository.Update(c, id, agenda)
	if err != nil {
		return nil, err
	}
	return agenda, nil
}

func (s *service) Delete(c *gin.Context, id string) error {
	agenda, err := s.Repository.Show(c, id)
	if err != nil {
		return err
	}
	now := time.Now()
	agenda.UpdatedAt = now
	agenda.DeletedAt = &now

	_, err = s.Repository.Update(c, agenda.ID, agenda)
	if err != nil {
		return err
	}
	return nil
}
