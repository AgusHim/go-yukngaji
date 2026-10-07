package fundraising

import (
	"fmt"

	"mainyuk/internal/apperr"
)

// ErrCampaignNotFound dipakai untuk campaign yang tidak ada, sudah dihapus,
// atau belum diterbitkan (bagi publik).
var ErrCampaignNotFound = fmt.Errorf("%w: campaign tidak ditemukan", apperr.ErrNotFound)

// ErrDonationNotFound dipakai untuk donasi yang tidak ada, sudah dihapus, atau
// bukan milik pemanggil.
var ErrDonationNotFound = fmt.Errorf("%w: donasi tidak ditemukan", apperr.ErrNotFound)

// ErrUpdateNotFound dipakai untuk update campaign yang tidak ada.
var ErrUpdateNotFound = fmt.Errorf("%w: update campaign tidak ditemukan", apperr.ErrNotFound)

// ErrDonationWindowClosed dipakai saat campaign belum mulai, sudah berakhir,
// atau belum diterbitkan.
var ErrDonationWindowClosed = fmt.Errorf("%w: campaign ini sedang tidak menerima donasi", apperr.ErrInvalidRequest)

// ErrDonationAlreadyDecided dipakai saat donasi yang sudah diputuskan dicoba
// diputuskan lagi dengan keputusan yang berbeda.
var ErrDonationAlreadyDecided = fmt.Errorf("%w: donasi sudah diputuskan", apperr.ErrInvalidRequest)

// ErrInvalidTransition dipakai saat perpindahan status yang diminta tidak sah.
var ErrInvalidTransition = fmt.Errorf("%w: perpindahan status tidak sah", apperr.ErrInvalidRequest)

// ValidationError menandai kesalahan input pemanggil. Dipetakan ke HTTP 400
// lewat apperr.IsValidation.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

func (e *ValidationError) IsValidation() bool {
	return true
}

func invalid(format string, args ...any) error {
	return &ValidationError{Message: fmt.Sprintf(format, args...)}
}
