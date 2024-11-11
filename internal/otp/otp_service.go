package otp

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"html/template"
	"log"
	"mainyuk/internal/user"
	"mainyuk/utils"
	"math/big"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type service struct {
	Repository     Repository
	UserRepository user.Repository
}

func NewService(repository Repository, userRepository user.Repository) Service {
	return &service{
		Repository:     repository,
		UserRepository: userRepository,
	}
}

func GenerateOTP(length int) (string, error) {
	const charset = "1234567890"
	otp := make([]byte, length)
	for i := range otp {
		randomInt, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}
		otp[i] = charset[randomInt.Int64()]
	}
	return string(otp), nil
}

func (s *service) RequestOTP(c *gin.Context, req ReqOtp) (*Otp, error) {
	otpActive, _ := s.Repository.Show(c, &req.Email, nil)
	if otpActive != nil {
		now := time.Now()
		current := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), 0, time.UTC)
		expires := otpActive.ExpiresAt
		if current.Before(expires) {
			return otpActive, nil
		}
	}

	otp := &Otp{}
	otp.ID = uuid.NewString()
	otp.Email = req.Email
	code, err := GenerateOTP(6)
	if err != nil {
		return nil, err
	}
	otp.Code = code
	now := time.Now()
	otp.CreatedAt = now
	otp.ExpiresAt = now.Add(15 * time.Minute)
	otp, err = s.Repository.Create(c, otp)
	if err != nil {
		return nil, err
	}

	// Send OTP in a goroutine, so it doesn’t block the request
	go func() {
		// Load HTML template
		tmpl, err := template.ParseFiles("template/otp_template.tmpl")
		if err != nil {
			log.Printf("Failed read template %s", err)
		}

		var body bytes.Buffer
		if err := tmpl.Execute(&body, otp); err != nil {
			log.Printf("Failed parse template %s", err)
		}

		if err := utils.SendEmail(otp.Email, "no-reply@ynsolo.com", "Kode OTP untuk login di ynsolo.com", body.String()); err != nil {
			fmt.Printf("Failed to send OTP %s: %s", otp.Email, err)
			// Handle logging or any follow-up for failure if needed
		}
	}()
	return otp, nil
}

func (s *service) VerifyOTP(c *gin.Context, req ReqOtp) (*user.User, error) {
	otp, err := s.Repository.Show(c, &req.Email, nil)
	if err != nil {
		return nil, err
	}

	if otp.Code != req.Code {
		return nil, errors.New("code OTP not match")
	}

	// Check Expires
	now := time.Now()
	current := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute(), now.Second(), 0, time.UTC)
	expires := otp.ExpiresAt
	if current.After(expires) {
		return nil, errors.New("code OTP expired")
	}

	u, _ := s.UserRepository.GetUserByEmail(c, req.Email)
	if u != nil {
		return u, nil
	}
	user := &user.User{}
	user.ID = uuid.NewString()
	user.Name = ""
	user.Username = "anonim"
	user.Gender = "male"

	user.Age = 0
	user.Phone = ""
	user.Email = &otp.Email
	user.Address = ""
	user.Role = "user"

	user.Activity = nil

	user.CreatedAt = time.Now()
	user.UpdatedAt = time.Now()

	user, err = s.UserRepository.CreateUser(c, user)
	if err != nil {
		return u, nil
	}

	return user, nil
}

func (s *service) GetUserIDAuth(c *gin.Context) (*user.User, error) {
	u, exists := c.Get("currentUser")
	if !exists {
		return nil, errors.New("not authrized")
	}

	currentUser, ok := u.(user.User)

	if !ok {
		return nil, errors.New("FailedParsing: current user")
	}
	return &currentUser, nil
}
