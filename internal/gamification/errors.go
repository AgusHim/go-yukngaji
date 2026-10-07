package gamification

import (
	"fmt"

	"mainyuk/internal/apperr"
)

// ErrRuleNotFound dipakai saat sumber XP yang diminta tidak punya aturan.
var ErrRuleNotFound = fmt.Errorf("%w: aturan XP tidak ditemukan", apperr.ErrNotFound)

// ErrUserNotFound dipakai saat admin mengoreksi XP untuk akun yang tidak ada.
var ErrUserNotFound = fmt.Errorf("%w: akun tidak ditemukan", apperr.ErrNotFound)

// ValidationError menandai kesalahan input pemanggil (periode tidak dikenal,
// delta nol, alasan kosong). Dipetakan ke HTTP 400 lewat apperr.IsValidation.
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
