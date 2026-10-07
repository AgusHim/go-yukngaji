package feedback

import (
	"errors"
	"mainyuk/internal/apperr"
	"mainyuk/internal/event"
	"mainyuk/internal/ratelimit"
	"mainyuk/internal/user"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type service struct {
	FeedbackRepository Repository
	UserService        user.Service
	EventService       event.Service
}

func NewService(repository Repository, us user.Service, es event.Service) Service {
	return &service{
		FeedbackRepository: repository,
		UserService:        us,
		EventService:       es,
	}
}

// Create menyimpan masukan dari pengguna yang sedang login.
func (s *service) Create(c *gin.Context, req *CreateFeedback) (*Feedback, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}
	if !ratelimit.Feedback.Allow(ratelimit.Key(c, currentUser.ID)) {
		return nil, ratelimit.ErrTooManyRequests
	}

	event, errEvent := s.EventService.Show(c, req.EventID)
	if errEvent != nil {
		return nil, errors.New("EventNotFound")
	}
	author, errUser := s.UserService.Show(c, currentUser.ID)
	if errUser != nil {
		return nil, errors.New("UserNotFound")
	}
	feedback := &Feedback{}
	feedback.ID = uuid.NewString()
	feedback.UserID = author.ID
	feedback.EventID = event.ID
	feedback.Message = req.Message
	feedback.CreatedAt = time.Now()
	feedback.UpdatedAt = time.Now()

	feedback, err := s.FeedbackRepository.Create(c, feedback)
	if err != nil {
		return nil, err
	}

	feedback.User = &User{
		ID:     author.ID,
		Name:   author.Name,
		Gender: author.Gender,
	}

	return feedback, nil
}

func (s *service) Index(c *gin.Context) ([]*Feedback, error) {
	event, err := s.FeedbackRepository.Index(c)
	if err != nil {
		return nil, err
	}
	return event, nil
}
