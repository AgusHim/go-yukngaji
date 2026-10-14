package instagram

import (
	"fmt"

	"mainyuk/internal/apperr"
)

// ErrPostNotFound dipakai untuk postingan yang tidak ada atau sudah dihapus.
var ErrPostNotFound = fmt.Errorf("%w: postingan tidak ditemukan", apperr.ErrNotFound)

// ErrSyncDisabled dipakai saat sinkronisasi diminta tetapi kredensial Graph API
// belum diisi. Ini bukan kegagalan sistem: dinding tetap jalan dengan postingan
// kurasi admin, jadi pesannya sengaja menjelaskan cara menyalakannya.
var ErrSyncDisabled = fmt.Errorf(
	"%w: sinkronisasi Instagram belum dikonfigurasi (isi INSTAGRAM_ACCESS_TOKEN dan INSTAGRAM_IG_USER_ID)",
	apperr.ErrInvalidRequest,
)

// ErrSyncFailed dipakai saat Graph API menolak permintaan. Pesannya membawa
// keterangan dari Meta supaya pengurus bisa menindaklanjuti tanpa membuka log.
var ErrSyncFailed = fmt.Errorf("%w: sinkronisasi Instagram gagal", apperr.ErrInvalidRequest)

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
