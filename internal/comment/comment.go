package comment

import (
	"time"

	"mainyuk/internal/community"

	"github.com/gin-gonic/gin"
)

type Comment struct {
	ID        string     `json:"id"`
	EventID   string     `json:"event_id"`
	UserID    string     `json:"-"`
	User      *User      `json:"user"`
	Comment   string     `json:"comment"`
	Like      int        `json:"like"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"-"`
	DeletedAt *time.Time `json:"-"`
}

// CreateComment sengaja tidak punya field user_id: identitas penulis selalu
// diambil dari token oleh server, bukan dari body permintaan.
type CreateComment struct {
	EventID string `json:"event_id" binding:"required"`
	Comment string `json:"comment" binding:"required"`
}

type User struct {
	ID       string `json:"id" `
	Username string `json:"username" binding:"required"`
	Gender   string `json:"gender" binding:"required"`
}

type Repository interface {
	Create(ctx *gin.Context, comment *Comment) (*Comment, error)
	Show(ctx *gin.Context, id string) (*Comment, error)
	Index(ctx *gin.Context) ([]*Comment, error)
	Update(ctx *gin.Context, comment *Comment) (*Comment, error)
	// AdjustLike menambah/mengurangi kolom like secara atomik di database.
	AdjustLike(ctx *gin.Context, id string, delta int) error
}

// AuthorGuard adalah seam sempit ke modul identitas komunitas, dipenuhi oleh
// community.Service. Tugasnya satu: memberi tahu apakah akun ini sedang
// dibatasi, sehingga akun yang ditangguhkan tidak bisa menulis di QnA event.
//
// Bentuknya di sisi pemakai, seperti user.ProfileRewarder dan
// presence.CheckInRewarder, supaya paket ini tidak perlu tahu apa pun tentang
// tabel profil maupun aturan visibilitasnya.
type AuthorGuard interface {
	EnsureProfile(ctx *gin.Context, userID string) (*community.Profile, error)
}

type Service interface {
	Create(ctx *gin.Context, req *CreateComment) (*Comment, error)
	Show(ctx *gin.Context, id string) (*Comment, error)
	Index(ctx *gin.Context) ([]*Comment, error)
	Update(ctx *gin.Context, comment *Comment) (*Comment, error)
	// AdjustLike menambah/mengurangi jumlah like tanpa baca-ubah-tulis.
	AdjustLike(ctx *gin.Context, id string, delta int) error
	// SetAuthorGuard memasang pemeriksa blokir akun. Boleh tidak dipasang:
	// tanpa penjaga, perilakunya sama seperti sebelum Fase 3.
	SetAuthorGuard(guard AuthorGuard)
}

type Handler interface {
	Create(ctx *gin.Context)
	Index(ctx *gin.Context)
}
