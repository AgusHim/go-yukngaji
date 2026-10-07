package community

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

// Ensure membuat baris profil bila belum ada. ON CONFLICT DO NOTHING membuat
// pemanggilan paralel aman: yang kalah balapan tidak menimpa data dan tidak
// menghasilkan error, lalu tetap membaca baris hasil pemenang.
func (r *repository) Ensure(c *gin.Context, userID string) (*Profile, error) {
	err := r.db.Exec(
		`INSERT INTO community_profiles (user_id, public_id) VALUES (?, ?)
		 ON CONFLICT (user_id) DO NOTHING`,
		userID, NewPublicID(),
	).Error
	if err != nil {
		return nil, err
	}
	return r.FindByUserID(c, userID)
}

func (r *repository) FindByUserID(c *gin.Context, userID string) (*Profile, error) {
	profile := &Profile{}
	err := r.db.Where("user_id = ?", userID).First(profile).Error
	if err != nil {
		return nil, err
	}
	return profile, nil
}

func (r *repository) FindByPublicID(c *gin.Context, publicID string) (*Profile, error) {
	profile := &Profile{}
	err := r.db.Where("public_id = ?", publicID).First(profile).Error
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// Update memakai map kolom eksplisit supaya hanya field yang dikirim client
// yang berubah, dan supaya UpdatedAt ikut terisi tanpa menyentuh kolom lain.
func (r *repository) Update(c *gin.Context, userID string, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	fields["updated_at"] = time.Now()
	return r.db.Model(&Profile{}).
		Where("user_id = ?", userID).
		Updates(fields).Error
}

// ProfilesOf membaca banyak profil dalam satu query. Tidak ada Ensure di sini
// dengan sengaja: jalur baca tidak boleh menulis, dan memanggil Ensure per
// penulis pada satu halaman feed berarti satu INSERT per baris.
func (r *repository) ProfilesOf(c *gin.Context, userIDs []string) (map[string]*Profile, error) {
	profiles := make(map[string]*Profile, len(userIDs))
	if len(userIDs) == 0 {
		return profiles, nil
	}

	rows := []*Profile{}
	if err := r.db.Where("user_id IN ?", userIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, p := range rows {
		profiles[p.UserID] = p
	}
	return profiles, nil
}

// SetBlocked memakai syarat is_blocked = !blocked supaya permintaan yang tidak
// mengubah apa pun tidak menghasilkan baris terpengaruh. Pemanggil memakai
// hasilnya untuk memutuskan apakah ada yang perlu dicatat ke audit.
func (r *repository) SetBlocked(c *gin.Context, userID string, blocked bool) (bool, error) {
	res := r.db.Model(&Profile{}).
		Where("user_id = ?", userID).
		Where("is_blocked = ?", !blocked).
		Updates(map[string]interface{}{
			"is_blocked": blocked,
			"updated_at": time.Now(),
		})
	return res.RowsAffected > 0, res.Error
}

func (r *repository) ListBlocked(c *gin.Context, limit, offset int) ([]*Profile, error) {
	rows := []*Profile{}
	err := r.db.
		Where("is_blocked = ?", true).
		Order("updated_at DESC, user_id DESC").
		Limit(limit).
		Offset(offset).
		Find(&rows).Error
	return rows, err
}
