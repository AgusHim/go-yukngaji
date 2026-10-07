package order

import (
	"errors"
	"fmt"
)

// ValidationError menandai kegagalan order yang disebabkan input pemesan
// (tiket tidak valid, kuota habis, transisi status tidak sah), bukan
// kegagalan server. Handler memetakannya ke HTTP 400.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

// IsValidation menandai tipe ini untuk apperr.IsValidation, sehingga pemeta
// HTTP bisa mengenali kesalahan input tanpa mengimpor paket order.
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
