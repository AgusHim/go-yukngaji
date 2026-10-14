package instagram

import (
	"errors"
	"log"
	"strings"
	"time"

	"mainyuk/internal/apperr"
	"mainyuk/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type service struct {
	Repository
	// client boleh nil. Nil berarti kredensial Graph API belum diisi, dan
	// sinkronisasi dijawab dengan ErrSyncDisabled.
	client Client
}

// NewService merakit service. client boleh nil.
func NewService(repository Repository, client Client) Service {
	return &service{
		Repository: repository,
		client:     client,
	}
}

func (s *service) ListPublished(c *gin.Context) ([]*Post, error) {
	return s.Repository.ListPublished(c)
}

func (s *service) ListAll(c *gin.Context) ([]*Post, error) {
	return s.Repository.ListAll(c)
}

func (s *service) Create(c *gin.Context, req *CreatePost) (*Post, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	username, err := normalizeUsername(req.Username)
	if err != nil {
		return nil, err
	}
	mediaURL := strings.TrimSpace(req.MediaURL)
	if mediaURL == "" {
		return nil, invalid("tautan gambar wajib diisi")
	}
	if err := validateMediaType(req.MediaType); err != nil {
		return nil, err
	}
	if err := validateCounts(req.LikeCount, req.CommentsCount); err != nil {
		return nil, err
	}
	postedAt, err := parseOptionalTime(req.PostedAt)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	post := &Post{
		ID:            uuid.NewString(),
		Source:        SourceManual,
		Username:      username,
		Caption:       trimPtr(req.Caption),
		MediaType:     req.MediaType,
		MediaURL:      &mediaURL,
		ThumbnailURL:  trimPtr(req.ThumbnailURL),
		Permalink:     trimPtr(req.Permalink),
		LikeCount:     req.LikeCount,
		CommentsCount: req.CommentsCount,
		PostedAt:      postedAt,
		IsPublished:   req.IsPublished,
		SortOrder:     req.SortOrder,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if currentUser, ok := user.FromContext(c); ok {
		post.CreatedBy = &currentUser.ID
	}

	if err := s.Repository.Create(c, post); err != nil {
		return nil, err
	}
	return post, nil
}

func (s *service) Update(c *gin.Context, id string, req *UpdatePost) (*Post, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	if _, err := s.Repository.FindByID(c, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPostNotFound
		}
		return nil, err
	}

	fields := map[string]interface{}{}

	if req.Username != nil {
		username, err := normalizeUsername(*req.Username)
		if err != nil {
			return nil, err
		}
		fields["username"] = username
	}
	if req.MediaURL != nil {
		mediaURL := strings.TrimSpace(*req.MediaURL)
		if mediaURL == "" {
			return nil, invalid("tautan gambar wajib diisi")
		}
		fields["media_url"] = mediaURL
	}
	if req.MediaType != nil {
		if err := validateMediaType(req.MediaType); err != nil {
			return nil, err
		}
		fields["media_type"] = trimPtr(req.MediaType)
	}
	if req.LikeCount != nil || req.CommentsCount != nil {
		if err := validateCounts(req.LikeCount, req.CommentsCount); err != nil {
			return nil, err
		}
	}
	// Angka 0 adalah nilai yang sah ("nol suka"), jadi kehadiran field ditentukan
	// oleh pointer-nya, bukan oleh nilainya.
	if req.LikeCount != nil {
		fields["like_count"] = *req.LikeCount
	}
	if req.CommentsCount != nil {
		fields["comments_count"] = *req.CommentsCount
	}
	if req.Caption != nil {
		fields["caption"] = trimPtr(req.Caption)
	}
	if req.ThumbnailURL != nil {
		fields["thumbnail_url"] = trimPtr(req.ThumbnailURL)
	}
	if req.Permalink != nil {
		fields["permalink"] = trimPtr(req.Permalink)
	}
	if req.PostedAt != nil {
		postedAt, err := parseOptionalTime(req.PostedAt)
		if err != nil {
			return nil, err
		}
		fields["posted_at"] = postedAt
	}
	if req.IsPublished != nil {
		fields["is_published"] = *req.IsPublished
	}
	if req.SortOrder != nil {
		fields["sort_order"] = *req.SortOrder
	}

	if len(fields) > 0 {
		if err := s.Repository.Update(c, id, fields); err != nil {
			return nil, err
		}
	}
	return s.Repository.FindByID(c, id)
}

func (s *service) Delete(c *gin.Context, id string) error {
	if _, err := s.Repository.FindByID(c, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPostNotFound
		}
		return err
	}
	return s.Repository.SoftDelete(c, id)
}

// Sync menyalin postingan terbaru dari Graph API.
//
// Tanpa kredensial, permintaannya ditolak dengan pesan yang bisa dibaca
// pengurus dan tidak ada satu baris pun yang ditulis.
func (s *service) Sync(c *gin.Context) (*SyncResult, error) {
	if s.client == nil {
		return nil, ErrSyncDisabled
	}

	result := &SyncResult{}

	// Postingan akun sendiri membawa angka like/komentar yang asli.
	own, err := s.client.Media(c)
	if err != nil {
		return nil, err
	}
	// Postingan anggota yang menandai akun. Kegagalannya tidak menggagalkan
	// seluruh sync: dinding akun resmi tetap terisi, dan alasannya dicatat.
	tagged, err := s.client.Tagged(c)
	if err != nil {
		log.Printf("[instagram] gagal mengambil postingan bertanda: %v", err)
		tagged = nil
	}

	for _, m := range append(own, tagged...) {
		result.Fetched++
		post, err := postFromMedia(m)
		if err != nil {
			result.Skipped++
			continue
		}
		created, err := s.Repository.UpsertSynced(c, post)
		if err != nil {
			return nil, err
		}
		if created {
			result.Created++
		} else {
			result.Updated++
		}
	}

	return result, nil
}

// postFromMedia mengubah satu item Graph API menjadi baris tabel.
//
// Item tanpa id dilewati: tanpa ig_media_id, deduplikasi tidak mungkin.
func postFromMedia(m *Media) (*Post, error) {
	if m == nil || strings.TrimSpace(m.ID) == "" {
		return nil, invalid("item tanpa id")
	}

	mediaID := strings.TrimSpace(m.ID)
	username := "yukngajisolo"
	if m.Username != nil && strings.TrimSpace(*m.Username) != "" {
		username = strings.TrimSpace(*m.Username)
	}

	return &Post{
		ID:            uuid.NewString(),
		IGMediaID:     &mediaID,
		Source:        SourceSync,
		Username:      username,
		Caption:       trimPtr(m.Caption),
		MediaType:     m.MediaType,
		MediaURL:      trimPtr(m.MediaURL),
		ThumbnailURL:  trimPtr(m.ThumbnailURL),
		Permalink:     trimPtr(m.Permalink),
		LikeCount:     m.LikeCount,
		CommentsCount: m.CommentsCount,
		PostedAt:      parseGraphTime(m.Timestamp),
		IsPublished:   false,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}, nil
}

// normalizeUsername menerima nama akun dengan atau tanpa '@' dan mengembalikan
// bentuk kanoniknya tanpa '@'.
func normalizeUsername(value string) (string, error) {
	username := strings.TrimPrefix(strings.TrimSpace(value), "@")
	if username == "" {
		return "", invalid("nama akun wajib diisi")
	}
	return username, nil
}

func validateMediaType(value *string) error {
	if value == nil {
		return nil
	}
	switch strings.TrimSpace(*value) {
	case "", MediaTypeImage, MediaTypeVideo, MediaTypeCarousel:
		return nil
	default:
		return invalid("jenis media harus salah satu dari IMAGE, VIDEO, CAROUSEL_ALBUM")
	}
}

func validateCounts(like, comments *int) error {
	if like != nil && *like < 0 {
		return invalid("jumlah suka tidak boleh negatif")
	}
	if comments != nil && *comments < 0 {
		return invalid("jumlah komentar tidak boleh negatif")
	}
	return nil
}

// parseOptionalTime mengurai tanggal RFC3339. String kosong berarti "tanggal
// tidak diketahui" dan menghasilkan nil, bukan waktu nol.
func parseOptionalTime(value *string) (*time.Time, error) {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(*value))
	if err != nil {
		return nil, invalid("format tanggal tidak valid, gunakan RFC3339: %s", *value)
	}
	return &parsed, nil
}

// parseGraphTime mengurai field `timestamp` Graph API.
//
// Graph API mengirim offset tanpa titik dua ("+0000"), yang tidak diterima
// layout RFC3339 Go, jadi bentuk itu dicoba terpisah. Nilai yang tidak terbaca
// menghasilkan nil — kartu tetap tampil, hanya urutannya jatuh ke created_at.
func parseGraphTime(value *string) *time.Time {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	raw := strings.TrimSpace(*value)
	for _, layout := range []string{time.RFC3339, "2006-01-02T15:04:05-0700"} {
		if parsed, err := time.Parse(layout, raw); err == nil {
			return &parsed
		}
	}
	return nil
}

// trimPtr merapikan string opsional; nilai kosong menjadi nil supaya tidak
// tersimpan sebagai string kosong yang sulit dibedakan dari "tidak diisi".
func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
