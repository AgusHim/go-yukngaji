package community

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Tingkat visibilitas profil.
const (
	VisibilityPublic  = "public"
	VisibilityMembers = "members"
	VisibilityPrivate = "private"
)

// Profile adalah profil komunitas publik milik satu akun.
//
// public_id adalah satu-satunya identifier yang boleh dikirim ke response
// publik. UserID (id akun internal) tidak pernah disertakan pada DTO publik.
type Profile struct {
	UserID            string    `json:"user_id" gorm:"column:user_id;primaryKey"`
	PublicID          string    `json:"public_id" gorm:"column:public_id"`
	Alias             *string   `json:"alias" gorm:"column:alias"`
	Bio               *string   `json:"bio" gorm:"column:bio"`
	AvatarURL         *string   `json:"avatar_url" gorm:"column:avatar_url"`
	ProfileVisibility string    `json:"profile_visibility" gorm:"column:profile_visibility"`
	LeaderboardOptOut bool      `json:"leaderboard_opt_out" gorm:"column:leaderboard_opt_out"`
	IsBlocked         bool      `json:"is_blocked" gorm:"column:is_blocked"`
	ShowBadges        bool      `json:"show_badges" gorm:"column:show_badges"`
	CreatedAt         time.Time `json:"-" gorm:"column:created_at"`
	UpdatedAt         time.Time `json:"-" gorm:"column:updated_at"`
}

func (Profile) TableName() string {
	return "community_profiles"
}

// UpdateProfile memakai pointer agar field yang tidak dikirim client tidak
// ikut ditimpa. Semua opsional — hanya yang non-nil yang diubah.
type UpdateProfile struct {
	Alias             *string `json:"alias"`
	Bio               *string `json:"bio"`
	AvatarURL         *string `json:"avatar_url"`
	ProfileVisibility *string `json:"profile_visibility"`
	LeaderboardOptOut *bool   `json:"leaderboard_opt_out"`
	ShowBadges        *bool   `json:"show_badges"`
}

// DisplayAlias mengembalikan nama yang aman ditampilkan: alias bila sudah
// diisi, kalau tidak nama pengguna akun.
func (p *Profile) DisplayAlias(fallback string) string {
	if p != nil && p.Alias != nil && *p.Alias != "" {
		return *p.Alias
	}
	if fallback == "" {
		return "Anonim"
	}
	return fallback
}

// PublicProfile adalah bentuk profil yang aman dikirim ke publik. Tidak
// memuat id akun internal, status blokir, maupun preferensi privat.
type PublicProfile struct {
	PublicID   string  `json:"public_id"`
	Alias      string  `json:"alias"`
	Bio        *string `json:"bio"`
	AvatarURL  *string `json:"avatar_url"`
	ShowBadges bool    `json:"show_badges"`
}

// ToPublicProfile mengubah profil internal menjadi DTO publik.
func ToPublicProfile(p *Profile, fallbackName string) *PublicProfile {
	if p == nil {
		return nil
	}
	return &PublicProfile{
		PublicID:   p.PublicID,
		Alias:      p.DisplayAlias(fallbackName),
		Bio:        p.Bio,
		AvatarURL:  p.AvatarURL,
		ShowBadges: p.ShowBadges,
	}
}

// OwnProfile adalah bentuk profil untuk pemiliknya sendiri. Menggantikan
// alias mentah supaya penanda internal seperti is_blocked tidak ikut terkirim.
type OwnProfile struct {
	PublicID          string  `json:"public_id"`
	Alias             *string `json:"alias"`
	Bio               *string `json:"bio"`
	AvatarURL         *string `json:"avatar_url"`
	ProfileVisibility string  `json:"profile_visibility"`
	LeaderboardOptOut bool    `json:"leaderboard_opt_out"`
	ShowBadges        bool    `json:"show_badges"`
}

// ToOwnProfile mengubah profil internal menjadi DTO pemilik.
func ToOwnProfile(p *Profile) *OwnProfile {
	if p == nil {
		return nil
	}
	return &OwnProfile{
		PublicID:          p.PublicID,
		Alias:             p.Alias,
		Bio:               p.Bio,
		AvatarURL:         p.AvatarURL,
		ProfileVisibility: p.ProfileVisibility,
		LeaderboardOptOut: p.LeaderboardOptOut,
		ShowBadges:        p.ShowBadges,
	}
}

type Repository interface {
	// Ensure membuat baris profil bila belum ada, tanpa menimpa data yang
	// sudah diisi. Aman dipanggil berkali-kali dan dari request paralel.
	Ensure(ctx *gin.Context, userID string) (*Profile, error)
	FindByUserID(ctx *gin.Context, userID string) (*Profile, error)
	FindByPublicID(ctx *gin.Context, publicID string) (*Profile, error)
	Update(ctx *gin.Context, userID string, fields map[string]interface{}) error

	// ProfilesOf mengambil profil beberapa akun sekaligus, tanpa membuat baris
	// baru. Akun yang belum punya profil tidak muncul di peta hasil.
	//
	// Ada supaya jalur baca tidak perlu memanggil Ensure — yang berarti menulis
	// — satu kali per penulis yang ditampilkan.
	ProfilesOf(ctx *gin.Context, userIDs []string) (map[string]*Profile, error)

	// SetBlocked menangguhkan (true) atau memulihkan (false) sebuah akun.
	// Mengembalikan false bila statusnya sudah seperti yang diminta: itu bukan
	// kegagalan, hanya tidak ada perubahan yang perlu diaudit.
	SetBlocked(ctx *gin.Context, userID string, blocked bool) (bool, error)

	// ListBlocked mengembalikan akun yang sedang dibatasi, terbaru diubah dulu.
	ListBlocked(ctx *gin.Context, limit, offset int) ([]*Profile, error)
}

type Service interface {
	EnsureProfile(ctx *gin.Context, userID string) (*Profile, error)
	Me(ctx *gin.Context) (*Profile, error)
	Update(ctx *gin.Context, req *UpdateProfile) (*Profile, error)
	// ShowPublic mengembalikan profil publik. Profil privat atau diblokir
	// menghasilkan ErrProfileNotFound supaya keberadaannya tidak terungkap.
	ShowPublic(ctx *gin.Context, publicID string) (*PublicProfile, error)
	// AdminIdentity mengembalikan identitas publik pemilik akun untuk keperluan
	// moderasi. Berbeda dari ShowPublic, visibilitas privat dan status blokir
	// tidak disembunyikan: pengurus harus tetap dapat mengenali pemilik klaim
	// yang sedang diperiksa. Yang tetap tidak dibuka hanya data akun internal
	// (id akun, email, telepon, alamat).
	AdminIdentity(ctx *gin.Context, userID string) (*PublicProfile, error)
	// ProfilesOf mengambil profil beberapa akun sekaligus untuk keperluan
	// tampil, tanpa membuat baris baru. Bentuk mentahnya dipakai modul yang
	// perlu menegakkan aturan anonimitasnya sendiri (internal/thread);
	// hasilnya tidak pernah diserialisasi langsung ke HTTP.
	ProfilesOf(ctx *gin.Context, userIDs []string) (map[string]*Profile, error)
	// SetBlockedByPublicID menangguhkan atau memulihkan akun berdasarkan
	// public_id, sehingga id akun internal tidak perlu melintasi kabel.
	// Mengembalikan identitas publik akun dan apakah statusnya benar-benar
	// berubah — hanya perubahan yang perlu dicatat ke audit.
	SetBlockedByPublicID(ctx *gin.Context, publicID string, blocked bool) (*PublicProfile, bool, error)
	// ListBlocked menampilkan akun yang sedang dibatasi. Identitasnya
	// diselesaikan seperti AdminIdentity: petugas berizin boleh mengenali
	// pemiliknya, tetapi id akun, email, dan telepon tetap tidak ikut.
	//
	// page dan perPage dianggap sudah dirapikan pemanggil.
	ListBlocked(ctx *gin.Context, page, perPage int) ([]*PublicProfile, bool, error)
}

type Handler interface {
	Me(ctx *gin.Context)
	Update(ctx *gin.Context)
	ShowPublic(ctx *gin.Context)
}
