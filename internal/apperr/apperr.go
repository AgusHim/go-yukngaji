// Package apperr memuat error sentinel yang dipakai lintas modul.
//
// Tujuannya agar handler bisa memetakan error ke status HTTP dengan
// errors.Is, bukan dengan membandingkan teks pesan.
package apperr

import "errors"

var (
	// ErrUnauthorized dipakai saat aksi butuh identitas terverifikasi tetapi
	// konteks permintaan tidak punya user.
	ErrUnauthorized = errors.New("NotAuthrized")
	// ErrNotFound dipakai saat data yang dirujuk tidak ada.
	ErrNotFound = errors.New("NotFound")
	// ErrInvalidRequest dipakai saat permintaan tidak lolos validasi.
	ErrInvalidRequest = errors.New("InvalidRequest")
	// ErrForbidden dipakai saat identitas sudah terverifikasi tetapi tidak
	// berhak atas aksi tersebut (berbeda dari ErrUnauthorized: belum login).
	ErrForbidden = errors.New("Forbidden")
)

// Validation adalah penanda untuk error yang disebabkan input pemanggil.
//
// Dipakai supaya paket pemeta HTTP (httperr) bisa mengenali kesalahan
// validasi tanpa harus mengimpor paket domainnya — itu akan membentuk
// impor melingkar, karena paket domain memakai httperr di handler-nya.
type Validation interface {
	IsValidation() bool
}

// IsValidation melaporkan apakah err menandai dirinya sebagai kesalahan input.
func IsValidation(err error) bool {
	var v Validation
	return errors.As(err, &v) && v.IsValidation()
}
