package mission

import (
	"fmt"

	"mainyuk/internal/apperr"
)

// ErrMissionNotFound dipakai untuk misi yang tidak ada, sudah dihapus, atau
// belum dipublikasikan (bagi anggota).
var ErrMissionNotFound = fmt.Errorf("%w: misi tidak ditemukan", apperr.ErrNotFound)

// ErrClaimNotFound dipakai untuk klaim yang tidak ada atau sudah dihapus.
var ErrClaimNotFound = fmt.Errorf("%w: klaim tidak ditemukan", apperr.ErrNotFound)

// ErrClaimWindowClosed dipakai saat misi belum mulai, sudah berakhir, atau
// belum dipublikasikan.
var ErrClaimWindowClosed = fmt.Errorf("%w: jendela klaim misi sedang tertutup", apperr.ErrInvalidRequest)

// ErrClaimLimitReached dipakai saat kuota klaim anggota untuk periode ini habis.
var ErrClaimLimitReached = fmt.Errorf("%w: batas klaim untuk periode ini sudah tercapai", apperr.ErrInvalidRequest)

// ErrClaimAlreadyDecided dipakai saat klaim yang sudah diputuskan dicoba
// diputuskan lagi dengan keputusan yang berbeda.
var ErrClaimAlreadyDecided = fmt.Errorf("%w: klaim sudah diputuskan", apperr.ErrInvalidRequest)

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
