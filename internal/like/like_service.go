package like

import (
	"errors"
	"mainyuk/internal/comment"
	"mainyuk/internal/user"
	"mainyuk/internal/ws"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type service struct {
	LikeRepository Repository
	UserService    user.Service
	CommentService comment.Service
	Hub            *ws.Hub
}

func NewService(repository Repository, us user.Service, cs comment.Service, hub *ws.Hub) Service {
	return &service{
		LikeRepository: repository,
		UserService:    us,
		CommentService: cs,
		Hub:            hub,
	}
}

// Register implements Service
func (s *service) Create(c *gin.Context, req *CreateLike) (*Like, error) {
	comment, errComment := s.CommentService.Show(c, req.CommentID)
	if errComment != nil {
		return nil, errors.New("CommentNotFound")
	}

	user, errUser := s.UserService.Show(c, req.UserID)
	if errUser != nil {
		return nil, errors.New("UserNotFound")
	}

	like := &Like{}
	like.ID = uuid.NewString()
	like.CommentID = comment.ID
	like.EventID = comment.EventID
	like.UserID = user.ID
	like.CreatedAt = time.Now()
	like.UpdatedAt = time.Now()

	like, err := s.LikeRepository.Create(c, like)
	if err != nil {
		return nil, err
	}

	comment.Like = comment.Like + 1
	_, errCount := s.CommentService.Update(c, comment)
	if errCount != nil {
		return nil, errCount
	}

	msg := &ws.Message{
		RoomID:   comment.EventID,
		Username: "Server",
		Message: map[string]interface{}{
			"type": "like.add",
			"data": like,
		},
	}

	s.Hub.Broadcast <- msg

	return like, nil
}

func (s *service) Delete(c *gin.Context, id string) error {
	like, errLike := s.LikeRepository.Show(c, id)
	if errLike != nil {
		return errors.New("LikeNotFound")
	}
	comment, errComment := s.CommentService.Show(c, like.CommentID)
	if errComment != nil {
		return errors.New("CommentNotFound")
	}
	err := s.LikeRepository.Delete(c, id)
	if err != nil {
		return err
	}

	comment.Like = comment.Like - 1
	_, errCount := s.CommentService.Update(c, comment)

	if errCount != nil {
		return errCount
	}

	msg := &ws.Message{
		RoomID:   comment.EventID,
		Username: "Server",
		Message: map[string]interface{}{
			"type": "like.delete",
			"data": like,
		},
	}

	s.Hub.Broadcast <- msg

	return nil
}

func (s *service) Index(c *gin.Context) ([]*Like, error) {
	event, err := s.LikeRepository.Index(c)
	if err != nil {
		return nil, err
	}
	return event, nil
}
