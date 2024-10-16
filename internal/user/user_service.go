package user

import (
	"errors"
	"mainyuk/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	oauth2api "google.golang.org/api/oauth2/v2"
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
	u, _ := s.GetUserByEmail(c, *req.Email)
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

	if req.Password != nil {
		hash, errHash := utils.HashPassword(*req.Password)
		if errHash != nil {
			return nil, errHash
		}
		user.Password = &hash
	}

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

	if err := utils.CheckPassword(req.Password, *user.Password); err != nil {
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

	if req.Password != nil {
		hash, errHash := utils.HashPassword(*req.Password)
		if errHash != nil {
			return nil, errHash
		}
		user.Password = &hash
	}

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
	u, _ := s.GetUserByEmail(c, *req.Email)
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

	if req.Password != nil {
		hash, errHash := utils.HashPassword(*req.Password)
		if errHash != nil {
			return nil, errHash
		}
		user.Password = &hash
	}

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	user, err := s.Repository.CreateUser(c, user)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *service) Update(c *gin.Context, id string, u *CreateUser) (*User, error) {
	// Check User in Database
	user, err := s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}

	user.Name = u.Name
	user.Gender = u.Gender

	age, err := strconv.Atoi(u.Age)
	if err != nil {
		return nil, err
	}
	user.Age = age

	user.Phone = u.Phone
	user.Username = u.Username
	user.Address = u.Address
	user.ProvinceCode = u.ProvinceCode
	user.DistrictCode = u.DistrictCode
	user.SubDistrictCode = u.SubDistrictCode
	user.Activity = &u.Activity

	// if u.Email != nil {
	// 	user.Email = u.Email
	// }

	if u.Instagram != nil {
		user.Instagram = *u.Instagram
	}

	if u.Password != nil && *u.Password != "" {
		hash, errHash := utils.HashPassword(*u.Password)
		if errHash != nil {
			return nil, errHash
		}
		user.Password = &hash
	}

	if u.BirthDate != nil && *u.BirthDate != "" {
		birthDate, errParsed := time.Parse("2006-01-02T15:04", *u.BirthDate)
		if errParsed != nil {
			return nil, errParsed
		}
		user.BirthDate = birthDate
	}

	user.UpdatedAt = time.Now()

	_, err = s.Repository.Update(c, id, user)
	if err != nil {
		return nil, err
	}

	user, err = s.Repository.Show(c, id)
	if err != nil {
		return nil, err
	}

	return user, nil
}

func (s *service) AuthGoogleCallback(c *gin.Context, userInfo *oauth2api.Userinfo) (*User, error) {
	// Check google id in table
	u, _ := s.Repository.ShowByGoogleID(c, userInfo.Id)
	if u != nil {
		return u, nil
	}

	// Registered new user
	user := &User{}
	user.ID = uuid.NewString()
	user.Name = userInfo.Name
	user.GoogleID = &userInfo.Id
	user.ImageUrl = &userInfo.Picture
	user.Username = "anonim"
	user.Gender = strings.ToLower(userInfo.Gender)

	age := 0
	user.Age = age
	user.Phone = ""
	user.Email = &userInfo.Email
	user.Address = ""
	user.Role = "user"

	activity := ""
	user.Activity = &activity

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	createdUser, err := s.Repository.CreateUser(c, user)
	if err != nil {
		return nil, err
	}
	return createdUser, nil
}
