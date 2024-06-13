package ranger

import (
	"errors"
	"mainyuk/internal/divisi"
	"mainyuk/internal/user"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type service struct {
	Repository    Repository
	UserService   user.Service
	DivisiService divisi.Service
}

func NewService(repository Repository, userService user.Service, divisiService divisi.Service) Service {
	return &service{
		Repository:    repository,
		UserService:   userService,
		DivisiService: divisiService,
	}
}

// Register implements Service
func (s *service) Create(c *gin.Context, req *CreateRanger) (*Ranger, error) {
	ranger := &Ranger{}
	ranger.ID = uuid.NewString()

	r, _ := s.Repository.ShowByUserID(c, req.UserID)
	if r != nil {
		return nil, errors.New("UserAlreadyRanger")
	}
	user, errUser := s.UserService.Show(c, req.UserID)
	if errUser != nil {
		return nil, errUser
	}

	ranger.UserID = user.ID
	divisi, errDivisi := s.DivisiService.Show(c, req.DivisiID)
	if errDivisi != nil {
		return nil, errDivisi
	}
	ranger.DivisiID = divisi.ID
	ranger.CreatedAt = time.Now()
	ranger.UpdatedAt = time.Now()

	ranger, err := s.Repository.Create(c, ranger)
	if err != nil {
		return nil, err
	}
	return ranger, nil
}

func (s *service) Show(c *gin.Context, id string) (*Ranger, error) {
	ranger, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return ranger, nil
}

func (s *service) ShowByUserID(c *gin.Context, userID string) (*Ranger, error) {
	ranger, err := s.Repository.ShowByUserID(c, userID)
	if err != nil {
		return nil, err
	}
	return ranger, nil
}

func (s *service) Index(c *gin.Context) ([]*Ranger, error) {
	rangers, err := s.Repository.Index(c)
	if err != nil {
		return nil, err
	}
	return rangers, nil
}
