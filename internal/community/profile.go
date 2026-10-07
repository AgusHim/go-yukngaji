package community

import (
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
)

const (
	AliasMinLen = 3
	AliasMaxLen = 40
)

// NewPublicID menghasilkan identifier publik acak 16 karakter heksadesimal.
// Nilainya tidak dapat dipakai untuk menebak id akun internal.
func NewPublicID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
}

// ValidateAlias memeriksa alias komunitas. Alias kosong berarti "hapus alias"
// dan diizinkan; selain itu panjang dan karakternya dibatasi supaya aman
// ditampilkan dan tidak menyamar sebagai pengguna lain.
func ValidateAlias(alias string) error {
	alias = strings.TrimSpace(alias)
	if alias == "" {
		return nil
	}
	if n := utf8.RuneCountInString(alias); n < AliasMinLen || n > AliasMaxLen {
		return invalid("alias harus %d-%d karakter", AliasMinLen, AliasMaxLen)
	}
	for _, r := range alias {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == ' ', r == '.', r == '_', r == '-':
		default:
			return invalid("alias hanya boleh berisi huruf, angka, spasi, titik, garis bawah, dan tanda hubung")
		}
	}
	return nil
}

// ValidateVisibility memeriksa nilai visibilitas profil yang diizinkan.
func ValidateVisibility(v string) error {
	switch v {
	case VisibilityPublic, VisibilityMembers, VisibilityPrivate:
		return nil
	default:
		return invalid("visibilitas profil tidak dikenal: %s", v)
	}
}

// ProfileCompletion memuat field yang dinilai untuk reward "profil lengkap".
//
// Daftar field ini adalah keputusan produk yang masih dapat berubah — ubah di
// sini bila definisi "profil lengkap" berubah. Rewardnya sekali per akun, jadi
// melonggarkan definisi tidak memberi XP tambahan kepada yang sudah menerima.
type ProfileCompletion struct {
	Name            string
	Phone           string
	BirthDate       *time.Time
	ProvinceCode    string
	DistrictCode    string
	SubDistrictCode string
}

// IsProfileComplete melaporkan apakah profil sudah cukup lengkap.
func IsProfileComplete(p ProfileCompletion) bool {
	return strings.TrimSpace(p.Name) != "" &&
		strings.TrimSpace(p.Phone) != "" &&
		p.BirthDate != nil && !p.BirthDate.IsZero() &&
		strings.TrimSpace(p.ProvinceCode) != "" &&
		strings.TrimSpace(p.DistrictCode) != "" &&
		strings.TrimSpace(p.SubDistrictCode) != ""
}

// SanitizeAlias merapikan spasi berlebih pada alias.
func SanitizeAlias(alias string) string {
	return strings.Join(strings.Fields(alias), " ")
}
