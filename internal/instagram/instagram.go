// Package instagram mengelola dinding komunitas Instagram yang tampil di
// landing page.
//
// Isinya dua jalur yang bertemu di satu tabel:
//
//   - kurasi admin lewat endpoint admin (source = 'manual'), dan
//   - sinkronisasi Graph API (source = 'sync'), yang menyalin postingan akun
//     resmi maupun postingan anggota yang menandai akun itu.
//
// Sinkronisasi bersifat opsional dan tidak pernah menghalangi jalur kurasi:
// tanpa kredensial, client-nya nil dan setiap permintaan sync dijawab dengan
// pesan "belum dikonfigurasi".
package instagram

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Asal baris.
const (
	// SourceManual adalah postingan yang dikurasi pengurus.
	SourceManual = "manual"
	// SourceSync adalah postingan hasil sinkronisasi Graph API.
	SourceSync = "sync"
)

// Jenis media, mengikuti nilai field media_type dari Graph API.
const (
	MediaTypeImage    = "IMAGE"
	MediaTypeVideo    = "VIDEO"
	MediaTypeCarousel = "CAROUSEL_ALBUM"
)

// Post adalah satu kartu di dinding komunitas.
//
// LikeCount/CommentsCount bertipe pointer karena NULL berarti "angkanya tidak
// diketahui" — kartu menyembunyikan baris like/komentar selama keduanya NULL,
// dan itu berbeda artinya dari angka nol.
type Post struct {
	ID            string     `json:"id" gorm:"column:id;primaryKey"`
	IGMediaID     *string    `json:"ig_media_id" gorm:"column:ig_media_id"`
	Source        string     `json:"source" gorm:"column:source"`
	Username      string     `json:"username" gorm:"column:username"`
	Caption       *string    `json:"caption" gorm:"column:caption"`
	MediaType     *string    `json:"media_type" gorm:"column:media_type"`
	MediaURL      *string    `json:"media_url" gorm:"column:media_url"`
	ThumbnailURL  *string    `json:"thumbnail_url" gorm:"column:thumbnail_url"`
	Permalink     *string    `json:"permalink" gorm:"column:permalink"`
	LikeCount     *int       `json:"like_count" gorm:"column:like_count"`
	CommentsCount *int       `json:"comments_count" gorm:"column:comments_count"`
	PostedAt      *time.Time `json:"posted_at" gorm:"column:posted_at"`
	IsPublished   bool       `json:"is_published" gorm:"column:is_published"`
	SortOrder     int        `json:"sort_order" gorm:"column:sort_order"`
	CreatedBy     *string    `json:"-" gorm:"column:created_by"`
	CreatedAt     time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt     *time.Time `json:"-" gorm:"column:deleted_at"`
}

func (Post) TableName() string {
	return "instagram_posts"
}

// CreatePost adalah payload admin untuk menambah postingan.
type CreatePost struct {
	Username      string  `json:"username" binding:"required"`
	Caption       *string `json:"caption"`
	MediaType     *string `json:"media_type"`
	MediaURL      string  `json:"media_url" binding:"required"`
	ThumbnailURL  *string `json:"thumbnail_url"`
	Permalink     *string `json:"permalink"`
	LikeCount     *int    `json:"like_count"`
	CommentsCount *int    `json:"comments_count"`
	// PostedAt memakai RFC3339. Boleh kosong untuk postingan yang tanggalnya
	// tidak diketahui.
	PostedAt    *string `json:"posted_at"`
	IsPublished bool    `json:"is_published"`
	SortOrder   int     `json:"sort_order"`
}

// UpdatePost adalah payload admin untuk mengubah postingan. Field yang tidak
// dikirim tidak diubah.
type UpdatePost struct {
	Username      *string `json:"username"`
	Caption       *string `json:"caption"`
	MediaType     *string `json:"media_type"`
	MediaURL      *string `json:"media_url"`
	ThumbnailURL  *string `json:"thumbnail_url"`
	Permalink     *string `json:"permalink"`
	LikeCount     *int    `json:"like_count"`
	CommentsCount *int    `json:"comments_count"`
	PostedAt      *string `json:"posted_at"`
	IsPublished   *bool   `json:"is_published"`
	SortOrder     *int    `json:"sort_order"`
}

// SyncResult merangkum satu kali sinkronisasi.
type SyncResult struct {
	Fetched int `json:"fetched"`
	Created int `json:"created"`
	Updated int `json:"updated"`
	Skipped int `json:"skipped"`
}

// Media adalah satu item dari Graph API. Angka like/komentar bertipe pointer
// supaya field yang tidak dikirim API tidak tertulis sebagai nol.
type Media struct {
	ID            string  `json:"id"`
	Username      *string `json:"username"`
	Caption       *string `json:"caption"`
	MediaType     *string `json:"media_type"`
	MediaURL      *string `json:"media_url"`
	ThumbnailURL  *string `json:"thumbnail_url"`
	Permalink     *string `json:"permalink"`
	Timestamp     *string `json:"timestamp"`
	LikeCount     *int    `json:"like_count"`
	CommentsCount *int    `json:"comments_count"`
}

// Client adalah seam ke Graph API. Dipenuhi oleh graphClient, dan boleh nil
// ketika kredensial belum diisi — semua pemanggil wajib memeriksanya dulu.
type Client interface {
	// Media mengambil postingan milik akun sendiri.
	Media(ctx *gin.Context) ([]*Media, error)
	// Tagged mengambil postingan orang lain yang menandai akun ini.
	Tagged(ctx *gin.Context) ([]*Media, error)
}

type Repository interface {
	ListPublished(ctx *gin.Context) ([]*Post, error)
	ListAll(ctx *gin.Context) ([]*Post, error)
	FindByID(ctx *gin.Context, id string) (*Post, error)
	Create(ctx *gin.Context, p *Post) error
	Update(ctx *gin.Context, id string, fields map[string]interface{}) error
	SoftDelete(ctx *gin.Context, id string) error
	// UpsertSynced menyimpan satu hasil sinkronisasi, dideduplikasi lewat
	// ig_media_id. Mengembalikan true bila barisnya baru dibuat.
	UpsertSynced(ctx *gin.Context, p *Post) (bool, error)
}

type Service interface {
	ListPublished(ctx *gin.Context) ([]*Post, error)
	ListAll(ctx *gin.Context) ([]*Post, error)
	Create(ctx *gin.Context, req *CreatePost) (*Post, error)
	Update(ctx *gin.Context, id string, req *UpdatePost) (*Post, error)
	Delete(ctx *gin.Context, id string) error
	Sync(ctx *gin.Context) (*SyncResult, error)
}

type Handler interface {
	ListPublished(ctx *gin.Context)
	ListAll(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	Sync(ctx *gin.Context)
}
