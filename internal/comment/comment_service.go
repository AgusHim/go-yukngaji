package comment

import (
	"errors"
	"mainyuk/internal/apperr"
	"mainyuk/internal/event"
	"mainyuk/internal/ratelimit"
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
	// guard boleh nil. Sebelum Fase 3 tidak ada pemeriksaan blokir di sini;
	// nil berarti perilakunya tetap seperti itu.
	guard AuthorGuard
}

// SetAuthorGuard memasang pemeriksa blokir akun setelah service identitas
// tersedia. Dipanggil dari cmd/main.go, bukan dari konstruktor, karena
// community.Service dibangun setelah paket ini.
func (s *service) SetAuthorGuard(guard AuthorGuard) {
	s.guard = guard
}

// blocked melaporkan apakah penulis sedang dibatasi. Kegagalan membaca profil
// diperlakukan sebagai "tidak diblokir": QnA event tidak boleh ikut mati hanya
// karena tabel profil bermasalah.
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

func NewService(repository Repository, us user.Service, es event.Service, hub *ws.Hub) Service {
	return &service{
		CommentRepository: repository,
		UserService:       us,
		EventService:      es,
		Hub:               hub,
	}
}

// Create menyimpan komentar baru. Penulis diambil dari identitas yang sudah
// diverifikasi middleware, sehingga `user_id` yang dikirim client diabaikan.
func (s *service) Create(c *gin.Context, req *CreateComment) (*Comment, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}
	if !ratelimit.Comment.Allow(ratelimit.Key(c, currentUser.ID)) {
		return nil, ratelimit.ErrTooManyRequests
	}
	// Diperiksa setelah pembatas laju dan sebelum apa pun ditulis, sama
	// seperti di feed komunitas: akun terblokir tidak boleh menitipkan satu
	// baris pun, dan tidak boleh memakai penolakan blokir untuk mengukur apa pun.
	if s.blocked(c, currentUser.ID) {
		return nil, apperr.ErrForbidden
	}

	event, errEvent := s.EventService.Show(c, req.EventID)
	if errEvent != nil {
		return nil, errors.New("EventNotFound")
	}
	author, errUser := s.UserService.Show(c, currentUser.ID)
	if errUser != nil {
		return nil, errors.New("UserNotFound")
	}
	comment := &Comment{}
	comment.ID = uuid.NewString()
	comment.UserID = author.ID
	comment.EventID = event.ID
	comment.Comment = req.Comment
	comment.CreatedAt = time.Now()
	comment.UpdatedAt = time.Now()

	comment, err := s.CommentRepository.Create(c, comment)
	if err != nil {
		return nil, err
	}

	comment.User = &User{
		ID:       author.ID,
		Username: author.Username,
		Gender:   author.Gender,
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

func (s *service) AdjustLike(c *gin.Context, id string, delta int) error {
	return s.CommentRepository.AdjustLike(c, id, delta)
}
