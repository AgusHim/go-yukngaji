package payment

import (
	"errors"
	"fmt"
)

// ErrWebhookUnsupported dikembalikan provider yang tidak punya webhook.
//
// Sengaja bukan galat validasi: pemanggil yang memanggil ParseWebhook pada
// provider manual sedang salah memakai provider, bukan mengirim input buruk.
var ErrWebhookUnsupported = errors.New("provider ini tidak mendukung webhook")

// ValidationError menandai kesalahan input pemanggil. Dipetakan ke HTTP 400
// lewat apperr.IsValidation.
//
// Setiap paket domain di repo ini punya tipe validasinya sendiri, dan paket
// ini mengikuti pola yang sama supaya tidak ada paket domain yang perlu
// mengimpor paket lain hanya untuk menyusun galat 400.
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
