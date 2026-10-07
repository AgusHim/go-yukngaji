package shop

import (
	"errors"
	"fmt"

	"mainyuk/internal/apperr"
)

// ErrProductNotFound dipakai untuk produk yang tidak ada, sudah dihapus, atau
// belum diterbitkan (bagi publik).
var ErrProductNotFound = fmt.Errorf("%w: produk tidak ditemukan", apperr.ErrNotFound)

// ErrVariantNotFound dipakai untuk varian yang tidak ada atau sudah dihapus.
var ErrVariantNotFound = fmt.Errorf("%w: varian tidak ditemukan", apperr.ErrNotFound)

// ErrOrderNotFound dipakai untuk pesanan yang tidak ada, sudah dihapus, atau
// bukan milik pemanggil.
var ErrOrderNotFound = fmt.Errorf("%w: pesanan tidak ditemukan", apperr.ErrNotFound)

// ErrOrderAlreadyDecided dipakai saat pesanan yang sudah diputuskan dicoba
// diputuskan lagi dengan keputusan yang berbeda.
var ErrOrderAlreadyDecided = fmt.Errorf("%w: pesanan sudah diputuskan", apperr.ErrInvalidRequest)

// ErrInvalidTransition dipakai saat perpindahan status yang diminta tidak sah.
var ErrInvalidTransition = fmt.Errorf("%w: perpindahan status tidak sah", apperr.ErrInvalidRequest)

// ErrStockShortage dipakai saat stok tidak lagi mencukupi.
//
// Dipisah dari ErrInvalidTransition karena ia punya arti tersendiri bagi
// pengurus: pesanan yang reservasinya sudah kedaluwarsa dan stoknya diambil
// pembeli lain harus ditolak, bukan dipaksa jalan.
var ErrStockShortage = fmt.Errorf("%w: stok tidak lagi tersedia", apperr.ErrInvalidRequest)

// errStockShortage adalah sentinel internal yang dikembalikan repository dari
// dalam transaksi supaya transaksinya menggelinding mundur dengan bersih —
// pola yang sama dengan errLimitReached di internal/mission. Service
// menerjemahkannya menjadi ErrStockShortage yang menyebut variannya.
var errStockShortage = fmt.Errorf("stok tidak mencukupi")

// errVariantGone adalah sentinel internal untuk varian yang hilang atau
// dihapus tepat saat checkout berlangsung. Dipisah dari ErrVariantNotFound
// supaya service dapat menyebut varian mana yang bermasalah tanpa menebak dari
// teks galat.
var errVariantGone = fmt.Errorf("varian hilang saat checkout")

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

// IsValidationError melaporkan apakah err adalah kegagalan validasi paket ini.
func IsValidationError(err error) bool {
	var ve *ValidationError
	return errors.As(err, &ve)
}
