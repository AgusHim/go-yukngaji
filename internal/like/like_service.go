package like

import (
	"errors"
	"mainyuk/internal/apperr"
	"mainyuk/internal/authz"
	"mainyuk/internal/comment"
	"mainyuk/internal/ratelimit"
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
	// guard boleh nil; nil berarti tidak ada pemeriksaan blokir, seperti
	// sebelum Fase 3.
	guard AuthorGuard
}

// SetAuthorGuard memasang pemeriksa blokir akun setelah service identitas
// tersedia.
func (s *service) SetAuthorGuard(guard AuthorGuard) {
	s.guard = guard
}

// blocked melaporkan apakah penyuka sedang dibatasi. Kegagalan membaca profil
// diperlakukan sebagai "tidak diblokir", supaya masalah pada tabel profil
// tidak mematikan like untuk semua orang.
func (s *service) blocked(c *gin.Context, userID string) bool {
	if s.guard == nil {
		return false
	}
	profile, err := s.guard.EnsureProfile(c, userID)
	if err != nil {
		return false
	}
	return profile != nil && profile.IsBlocked
}

func NewService(repository Repository, us user.Service, cs comment.Service, hub *ws.Hub) Service {
	return &service{
		LikeRepository: repository,
		UserService:    us,
		CommentService: cs,
		Hub:            hub,
	}
}

// Create mencatat like dari pengguna yang sedang login. Identitas diambil
// dari konteks, bukan dari body permintaan.
func (s *service) Create(c *gin.Context, req *CreateLike) (*Like, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}
	if !ratelimit.Like.Allow(ratelimit.Key(c, currentUser.ID)) {
		return nil, ratelimit.ErrTooManyRequests
	}
	if s.blocked(c, currentUser.ID) {
		return nil, apperr.ErrForbidden
	}

	comment, errComment := s.CommentService.Show(c, req.CommentID)
	if errComment != nil {
		return nil, errors.New("CommentNotFound")
	}

	liker, errUser := s.UserService.Show(c, currentUser.ID)
	if errUser != nil {
		return nil, errors.New("UserNotFound")
	}

	like := &Like{}
	like.ID = uuid.NewString()
	like.CommentID = comment.ID
	like.EventID = comment.EventID
	like.UserID = liker.ID
	like.CreatedAt = time.Now()
	like.UpdatedAt = time.Now()

	like, err := s.LikeRepository.Create(c, like)
	if err != nil {
		return nil, err
	}

	if errCount := s.CommentService.AdjustLike(c, comment.ID, 1); errCount != nil {
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

// Delete membatalkan like. Hanya pemilik like (atau moderator) yang boleh.
func (s *service) Delete(c *gin.Context, id string) error {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return apperr.ErrUnauthorized
	}

	like, errLike := s.LikeRepository.Show(c, id)
	if errLike != nil {
		return errors.New("LikeNotFound")
	}

	if like.UserID != currentUser.ID && !authz.Can(currentUser.Role, authz.PermissionModerate) {
		return apperr.ErrUnauthorized
	}

	comment, errComment := s.CommentService.Show(c, like.CommentID)
	if errComment != nil {
		return errors.New("CommentNotFound")
	}
	err := s.LikeRepository.Delete(c, id)
	if err != nil {
		return err
	}

	if errCount := s.CommentService.AdjustLike(c, comment.ID, -1); errCount != nil {
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
