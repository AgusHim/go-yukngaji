package thread

import (
	"errors"
	"fmt"

	"mainyuk/internal/apperr"
)

// ErrThreadNotFound dipakai untuk thread yang tidak ada, sudah dihapus, atau
// tidak terlihat oleh pemanggil. Disamakan supaya status konten yang
// disembunyikan tidak bisa disimpulkan dari perbedaan status HTTP.
var ErrThreadNotFound = fmt.Errorf("%w: thread tidak ditemukan", apperr.ErrNotFound)

// ErrCommentNotFound dipakai untuk komentar yang tidak ada atau sudah dihapus.
var ErrCommentNotFound = fmt.Errorf("%w: komentar tidak ditemukan", apperr.ErrNotFound)

// ErrReportNotFound dipakai untuk laporan yang tidak ada.
var ErrReportNotFound = fmt.Errorf("%w: laporan tidak ditemukan", apperr.ErrNotFound)

// ErrAccountRestricted dipakai saat akun yang dibatasi mencoba menulis.
// Dipetakan ke HTTP 403, bukan 400: ini soal kewenangan, bukan input.
var ErrAccountRestricted = fmt.Errorf("%w: akun ini sedang dibatasi", apperr.ErrForbidden)

// ErrAccountNotFound dipakai saat public_id akun yang hendak dibatasi tidak
// ada. Dipisahkan dari ErrAccountRestricted supaya "tidak ada" (404) tidak
// pernah tertukar dengan "tidak boleh" (403).
var ErrAccountNotFound = fmt.Errorf("%w: akun tidak ditemukan", apperr.ErrNotFound)

// ValidationError menandai kesalahan input pemanggil (panjang judul, kata
// terlarang, alasan kosong). Dipetakan ke HTTP 400 lewat apperr.IsValidation,
// tanpa httperr perlu mengimpor paket ini.
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

// IsValidationError melaporkan apakah err adalah kegagalan validasi paket ini.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}
