package community

import (
	"errors"
	"fmt"

	"mainyuk/internal/apperr"
)

// ErrProfileNotFound dipakai baik untuk profil yang benar-benar tidak ada
// maupun untuk profil privat/diblokir. Sengaja disamakan supaya keberadaan
// profil privat tidak bisa disimpulkan dari perbedaan status HTTP.
var ErrProfileNotFound = fmt.Errorf("%w: profil komunitas tidak ditemukan", apperr.ErrNotFound)

// ValidationError menandai kesalahan input pemanggil (alias tidak valid,
// visibilitas tidak dikenal). Handler memetakannya ke HTTP 400 lewat
// apperr.IsValidation, tanpa httperr perlu mengimpor paket ini.
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

// IsValidationError melaporkan apakah err adalah kegagalan validasi.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}
