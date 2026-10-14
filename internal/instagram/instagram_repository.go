package instagram

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// isUniqueViolation melaporkan apakah err berasal dari pelanggaran constraint
// unik PostgreSQL (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

// ListPublished mengembalikan dinding publik: hanya yang terbit, urut kurasi
// admin lebih dulu, lalu yang paling baru.
func (r *repository) ListPublished(c *gin.Context) ([]*Post, error) {
	posts := []*Post{}
	err := r.db.
		Where("deleted_at IS NULL").
		Where("is_published = true").
		Order("sort_order ASC, posted_at DESC NULLS LAST, created_at DESC").
		Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}

// ListAll adalah daftar untuk pengurus: termasuk yang belum terbit.
func (r *repository) ListAll(c *gin.Context) ([]*Post, error) {
	posts := []*Post{}
	err := r.db.
		Where("deleted_at IS NULL").
		Order("sort_order ASC, posted_at DESC NULLS LAST, created_at DESC").
		Find(&posts).Error
	if err != nil {
		return nil, err
	}
	return posts, nil
}

func (r *repository) FindByID(c *gin.Context, id string) (*Post, error) {
	post := &Post{}
	err := r.db.Where("id = ?", id).Where("deleted_at IS NULL").First(post).Error
	if err != nil {
		return nil, err
	}
	return post, nil
}

func (r *repository) Create(c *gin.Context, p *Post) error {
	return r.db.Create(p).Error
}

func (r *repository) Update(c *gin.Context, id string, fields map[string]interface{}) error {
	fields["updated_at"] = time.Now()
	return r.db.Model(&Post{}).Where("id = ?", id).Updates(fields).Error
}

func (r *repository) SoftDelete(c *gin.Context, id string) error {
	now := time.Now()
	return r.db.Model(&Post{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"updated_at": now,
		}).Error
}

// syncedFields adalah kolom yang boleh ditimpa ulang oleh sinkronisasi.
//
// is_published, sort_order, dan created_by sengaja TIDAK ada di sini: keputusan
// pengurus untuk menyembunyikan atau menggeser satu postingan tidak boleh
// dibatalkan oleh sync berikutnya.
func syncedFields(p *Post) map[string]interface{} {
	return map[string]interface{}{
		"caption":        p.Caption,
		"media_type":     p.MediaType,
		"media_url":      p.MediaURL,
		"thumbnail_url":  p.ThumbnailURL,
		"permalink":      p.Permalink,
		"like_count":     p.LikeCount,
		"comments_count": p.CommentsCount,
		"posted_at":      p.PostedAt,
		"username":       p.Username,
		"updated_at":     time.Now(),
	}
}

// UpsertSynced menyimpan satu hasil sinkronisasi.
//
// Deduplikasinya bertumpu pada ig_media_id, dengan dua lapis penjagaan:
//
//  1. Kunci advisory transaksional per media menyerialkan hitung-lalu-tulis,
//     sehingga dua sync bersamaan tidak sama-sama memutuskan "belum ada".
//  2. Index unik parsial uniq_instagram_posts_media_aktif adalah jaminan
//     kerasnya, berlaku walaupun kedua permintaan datang dari proses berbeda.
//
// Baris baru selalu dibuat dengan is_published = false: sinkronisasi tidak
// pernah menerbitkan apa pun yang belum ditinjau pengurus.
func (r *repository) UpsertSynced(c *gin.Context, p *Post) (bool, error) {
	if p.IGMediaID == nil || *p.IGMediaID == "" {
		return false, invalid("hasil sinkronisasi tanpa ig_media_id")
	}

	created := false

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			"SELECT pg_advisory_xact_lock(hashtext(?))",
			"ig:"+*p.IGMediaID,
		).Error; err != nil {
			return err
		}

		existing := &Post{}
		err := tx.
			Where("ig_media_id = ?", *p.IGMediaID).
			Where("deleted_at IS NULL").
			First(existing).Error

		switch {
		case err == nil:
			return tx.Model(&Post{}).
				Where("id = ?", existing.ID).
				Updates(syncedFields(p)).Error

		case errors.Is(err, gorm.ErrRecordNotFound):
			p.Source = SourceSync
			p.IsPublished = false
			if err := tx.Create(p).Error; err != nil {
				// Balapan dengan sync lain: barisnya sudah ada, jadi ini
				// dilewati, bukan digagalkan.
				if isUniqueViolation(err) {
					return nil
				}
				return err
			}
			created = true
			return nil

		default:
			return err
		}
	})
	if err != nil {
		return false, err
	}
	return created, nil
}
