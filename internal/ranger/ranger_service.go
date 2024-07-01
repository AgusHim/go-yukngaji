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

	var user *user.User
	var errUser error

	if req.UserID != nil {
		r, _ := s.Repository.ShowByUserID(c, *req.UserID)
		if r != nil {
			return nil, errors.New("UserAlreadyRanger")
		}

		user, errUser = s.UserService.Show(c, *req.UserID)
		if errUser != nil {
			return nil, errUser
		}
	}

	if req.User != nil && req.UserID == nil {
		user, errUser = s.UserService.CreateRanger(c, req.User)
		if errUser != nil {
			return nil, errUser
		}
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

// Update Rangers
func (s *service) Update(c *gin.Context, id string, req *CreateRanger) (*Ranger, error) {
	// Check rangers and get userID
	ranger, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}

	// Update user data
	var user user.CreateUser
	user.Name = req.User.Name
	user.Gender = req.User.Gender
	user.Age = req.User.Age
	user.Phone = req.User.Phone
	user.Email = req.User.Email
	user.Username = req.User.Username
	user.Address = req.User.Address
	user.Activity = req.User.Activity

	if req.User.Password != nil || *req.User.Password != "" {
		user.Password = req.User.Password
	}

	_, errUser := s.UserService.UpdateByAdmin(c, ranger.UserID, &user)
	if errUser != nil {
		return nil, errUser
	}

	ranger, err = s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return ranger, nil
}

func (s *service) Delete(c *gin.Context, id string) error {
	ranger, err := s.Repository.Show(c, id)
	if err != nil {
		return err
	}
	err = s.Repository.Delete(c, ranger.ID)
	if err != nil {
		return err
	}
	return nil
}
