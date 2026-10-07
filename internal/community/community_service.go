package community

import (
	"errors"

	"mainyuk/internal/apperr"
	"mainyuk/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

type service struct {
	Repository
	UserService user.Service
}

func NewService(repository Repository, userService user.Service) Service {
	return &service{
		Repository:  repository,
		UserService: userService,
	}
}

func (s *service) EnsureProfile(c *gin.Context, userID string) (*Profile, error) {
	if userID == "" {
		return nil, apperr.ErrInvalidRequest
	}
	return s.Repository.Ensure(c, userID)
}

func (s *service) Me(c *gin.Context) (*Profile, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}
	return s.Repository.Ensure(c, currentUser.ID)
}

func (s *service) Update(c *gin.Context, req *UpdateProfile) (*Profile, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	fields := map[string]interface{}{}

	if req.Alias != nil {
		alias := SanitizeAlias(*req.Alias)
		if err := ValidateAlias(alias); err != nil {
			return nil, err
		}
		// Alias kosong berarti menghapus alias, bukan menyimpan string kosong
		// yang akan menabrak index unik parsial.
		if alias == "" {
			fields["alias"] = nil
		} else {
			fields["alias"] = alias
		}
	}
	if req.Bio != nil {
		fields["bio"] = *req.Bio
	}
	if req.AvatarURL != nil {
		fields["avatar_url"] = *req.AvatarURL
	}
	if req.ProfileVisibility != nil {
		if err := ValidateVisibility(*req.ProfileVisibility); err != nil {
			return nil, err
		}
		fields["profile_visibility"] = *req.ProfileVisibility
	}
	if req.LeaderboardOptOut != nil {
		fields["leaderboard_opt_out"] = *req.LeaderboardOptOut
	}
	if req.ShowBadges != nil {
		fields["show_badges"] = *req.ShowBadges
	}

	// Pastikan barisnya ada dulu; kalau tidak, UPDATE akan diam-diam no-op.
	if _, err := s.Repository.Ensure(c, currentUser.ID); err != nil {
		return nil, err
	}

	if err := s.Repository.Update(c, currentUser.ID, fields); err != nil {
		if isUniqueViolation(err) {
			return nil, invalid("alias sudah dipakai akun lain")
		}
		return nil, err
	}
	return s.Repository.FindByUserID(c, currentUser.ID)
}

func (s *service) ShowPublic(c *gin.Context, publicID string) (*PublicProfile, error) {
	profile, err := s.Repository.FindByPublicID(c, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}

	// Profil privat dan akun terblokir sengaja mengembalikan error yang sama
	// dengan "tidak ada", supaya keberadaannya tidak bisa disimpulkan.
	if profile.IsBlocked || profile.ProfileVisibility == VisibilityPrivate {
		return nil, ErrProfileNotFound
	}

	fallback := ""
	if u, err := s.UserService.Show(c, profile.UserID); err == nil && u != nil {
		fallback = u.Username
		if fallback == "" {
			fallback = u.Name
		}
	}
	return ToPublicProfile(profile, fallback), nil
}

// AdminIdentity mengembalikan identitas publik pemilik akun untuk moderasi.
//
// Berbeda dari ShowPublic, profil privat dan akun terblokir tetap terlihat:
// tanpa itu pengurus tidak dapat mengenali pemilik klaim yang harus diputuskan.
// Data akun internal tetap tidak ikut.
func (s *service) AdminIdentity(c *gin.Context, userID string) (*PublicProfile, error) {
	if userID == "" {
		return nil, ErrProfileNotFound
	}

	profile, err := s.Repository.Ensure(c, userID)
	if err != nil {
		return nil, err
	}
	return s.identityOf(c, profile), nil
}

// identityOf menyelesaikan nama tampil sebuah profil tanpa memastikan
// barisnya ada. Dipakai jalur baca (daftar akun dibatasi) supaya tidak
// melakukan INSERT hanya untuk menampilkan daftar.
//
// Nama akun dipakai sebagai fallback di sini, dan hanya di sini: pemanggilnya
// adalah antrean moderasi yang memang berhak mengenali pemilik akun. Seluruh
// permukaan publik memakai aturan anonimitasnya sendiri.
func (s *service) identityOf(c *gin.Context, profile *Profile) *PublicProfile {
	fallback := ""
	if u, err := s.UserService.Show(c, profile.UserID); err == nil && u != nil {
		fallback = u.Username
		if fallback == "" {
			fallback = u.Name
		}
	}
	return ToPublicProfile(profile, fallback)
}

func (s *service) ProfilesOf(c *gin.Context, userIDs []string) (map[string]*Profile, error) {
	return s.Repository.ProfilesOf(c, userIDs)
}

// SetBlockedByPublicID memakai public_id sebagai pegangan, sehingga pemanggil
// dari HTTP tidak perlu pernah memegang id akun internal.
//
// Mengembalikan apakah statusnya benar-benar berubah: pemanggil memakai itu
// untuk memutuskan ada tidaknya yang perlu dicatat ke audit.
func (s *service) SetBlockedByPublicID(c *gin.Context, publicID string, blocked bool) (*PublicProfile, bool, error) {
	if publicID == "" {
		return nil, false, ErrProfileNotFound
	}

	profile, err := s.Repository.FindByPublicID(c, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, false, ErrProfileNotFound
		}
		return nil, false, err
	}

	changed, err := s.Repository.SetBlocked(c, profile.UserID, blocked)
	if err != nil {
		return nil, false, err
	}
	return s.identityOf(c, profile), changed, nil
}

func (s *service) ListBlocked(c *gin.Context, page, perPage int) ([]*PublicProfile, bool, error) {
	if perPage <= 0 {
		return nil, false, nil
	}

	rows, err := s.Repository.ListBlocked(c, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > perPage
	if hasMore {
		rows = rows[:perPage]
	}

	out := make([]*PublicProfile, 0, len(rows))
	for _, profile := range rows {
		out = append(out, s.identityOf(c, profile))
	}
	return out, hasMore, nil
}

// isUniqueViolation melaporkan apakah err berasal dari pelanggaran constraint
// unik PostgreSQL (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
