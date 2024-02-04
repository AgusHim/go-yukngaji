package comment

import (
	"errors"
	"mainyuk/internal/event"
	"mainyuk/internal/user"
	"mainyuk/internal/ws"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type service struct {
	CommentRepository Repository
	UserService       user.Service
	EventService      event.Service
	Hub               *ws.Hub
}

func NewService(repository Repository, us user.Service, es event.Service, hub *ws.Hub) Service {
	return &service{
		CommentRepository: repository,
		UserService:       us,
		EventService:      es,
		Hub:               hub,
	}
}

// Register implements Service
func (s *service) Create(c *gin.Context, req *CreateComment) (*Comment, error) {
	event, errEvent := s.EventService.Show(c, req.EventID)
	if errEvent != nil {
		return nil, errors.New("EventNotFound")
	}
	user, errUser := s.UserService.Show(c, req.UserID)
	if errUser != nil {
		return nil, errors.New("UserNotFound")
	}
	comment := &Comment{}
	comment.ID = uuid.NewString()
	comment.UserID = user.ID
	comment.EventID = event.ID
	comment.Comment = req.Comment
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()

	comment, err := s.CommentRepository.Create(c, comment)
	if err != nil {
		return nil, err
	}

	comment.User = &User{
		ID:       user.ID,
		Username: user.Username,
		Gender:   user.Gender,
	}

	msg := &ws.Message{
		RoomID:   event.ID,
		Username: "Server",
		Message: map[string]interface{}{
			"type": "comment.add",
			"data": comment,
		},
	}

	s.Hub.Broadcast <- msg

	return comment, nil
}

func (s *service) Show(c *gin.Context, id string) (*Comment, error) {
	event, err := s.CommentRepository.Show(c, id)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (s *service) Index(c *gin.Context) ([]*Comment, error) {
	event, err := s.CommentRepository.Index(c)
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (s *service) Update(c *gin.Context, comment *Comment) (*Comment, error) {
	comment.UpdatedAt = time.Now()
	comment, err := s.CommentRepository.Update(c, comment)
	if err != nil {
		return nil, err
	}
	return comment, nil
}
