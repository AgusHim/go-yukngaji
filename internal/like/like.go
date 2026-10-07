package like

import (
	"time"

	"mainyuk/internal/community"

	"github.com/gin-gonic/gin"
)

type Like struct {
	ID        string     `json:"id" binding:"required"`
	CommentID string     `json:"comment_id"`
	EventID   string     `json:"event_id"`
	UserID    string     `json:"user_id"`
	CreatedAt time.Time  `json:"created_at" binding:"required"`
	UpdatedAt time.Time  `json:"-" binding:"required"`
	DeletedAt *time.Time `json:"-" binding:"required"`
}

// CreateLike tidak menerima user_id dari client; identitas penyuka diambil
// server dari token.
type CreateLike struct {
	LikeID    string `json:"like_id"`
	CommentID string `json:"comment_id" binding:"required"`
}

type Repository interface {
	Show(ctx *gin.Context, id string) (*Like, error)
	Create(ctx *gin.Context, like *Like) (*Like, error)
	Delete(ctx *gin.Context, id string) error
	Index(ctx *gin.Context) ([]*Like, error)
}

// AuthorGuard adalah seam sempit ke modul identitas komunitas, dipenuhi oleh
// community.Service. Sama seperti di internal/comment: akun yang dibatasi
// tidak boleh menulis, termasuk menyukai komentar.
type AuthorGuard interface {
	EnsureProfile(ctx *gin.Context, userID string) (*community.Profile, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateLike) (*Like, error)
	Delete(ctx *gin.Context, id string) error
	Index(ctx *gin.Context) ([]*Like, error)
	// SetAuthorGuard memasang pemeriksa blokir akun. Boleh tidak dipasang.
	SetAuthorGuard(guard AuthorGuard)
}

type Handler interface {
	Create(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Index(ctx *gin.Context)
}
