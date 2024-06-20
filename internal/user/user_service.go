package user

import (
	"errors"
	"mainyuk/utils"
	"strconv"
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
func (s *service) Register(c *gin.Context, req *CreateUser) (*User, error) {
	u, _ := s.GetUserByEmail(c, req.Email)
	if u != nil {
		return nil, errors.New("EmailRegistered")
	}
	user := &User{}
	user.ID = uuid.NewString()
	user.Name = req.Name
	user.Username = req.Username
	user.Gender = req.Gender

	age, errAge := strconv.Atoi(req.Age)
	if errAge != nil {
		return nil, errAge
	}

	user.Age = age
	user.Phone = req.Phone
	user.Email = req.Email
	user.Address = req.Address
	user.Role = "user"

	activity := strings.ToLower(req.Activity)
	user.Activity = &activity

	hash, errHash := utils.HashPassword(req.Password)
	if errHash != nil {
		return nil, errHash
	}
	user.Password = hash
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	user, err := s.Repository.CreateUser(c, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

// Register implements Service
func (s *service) Login(c *gin.Context, req *Login) (*User, error) {
	user, err := s.Repository.GetUserByEmail(c, req.Email)
	if err != nil {
		return nil, errors.New("EmailNotFound")
	}

	if err := utils.CheckPassword(req.Password, user.Password); err != nil {
		return nil, errors.New("PasswordNotMatch")
	}

	return user, nil
}

func (s *service) Presence(c *gin.Context, req *CreateUser) (*User, error) {
	user := &User{}
	user.ID = uuid.NewString()
	user.Name = req.Name

	if req.Username == "" {
		user.Username = "anonim"
	} else {
		user.Username = req.Username
	}
	user.Gender = req.Gender

	age, errAge := strconv.Atoi(req.Age)
	if errAge != nil {
		return nil, errAge
	}

	user.Age = age
	user.Phone = req.Phone
	user.Email = req.Email
	user.Address = req.Address
	user.Role = "jamaah"

	activity := strings.ToLower(req.Activity)
	user.Activity = &activity

	hash, errHash := utils.HashPassword("taatbahagia")
	if errHash != nil {
		return nil, errHash
	}
	user.Password = hash
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	user, err := s.Repository.CreateUser(c, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) DeleteByID(c *gin.Context, id string) error {
	err := s.Repository.DeleteByID(c, id)
	if err != nil {
		return err
	}
	return nil
}

func (s *service) Show(c *gin.Context, id string) (*User, error) {
	event, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (s *service) CreateRanger(c *gin.Context, req *CreateUser) (*User, error) {
	u, _ := s.GetUserByEmail(c, req.Email)
	if u != nil {
		return nil, errors.New("EmailRegistered")
	}
	user := &User{}
	user.ID = uuid.NewString()
	user.Name = req.Name
	user.Username = req.Username
	user.Gender = req.Gender

	age, errAge := strconv.Atoi(req.Age)
	if errAge != nil {
		return nil, errAge
	}
	user.Age = age

	user.Phone = req.Phone
	user.Email = req.Email
	user.Address = req.Address
	user.Role = "ranger"

	activity := strings.ToLower(req.Activity)
	user.Activity = &activity

	hash, errHash := utils.HashPassword(req.Password)
	if errHash != nil {
		return nil, errHash
	}
	user.Password = hash
	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	user, err := s.Repository.CreateUser(c, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}
