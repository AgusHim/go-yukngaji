package otp

import (
	"errors"
	"strings"
	"time"
)

// Aturan OTP. Nilai-nilai ini dipakai service dan diuji tanpa database.
const (
	// MaxAttempts adalah jumlah percobaan kode salah sebelum kode ditolak.
	MaxAttempts = 5
	// CodeTTL adalah masa berlaku satu kode OTP.
	CodeTTL = 15 * time.Minute
	// ResendCooldown adalah jeda minimum sebelum kode boleh dikirim ulang.
	ResendCooldown = 60 * time.Second
)

var (
	ErrOTPNotFound  = errors.New("code OTP tidak ditemukan")
	ErrOTPUsed      = errors.New("code OTP sudah dipakai")
	ErrOTPExpired   = errors.New("code OTP expired")
	ErrOTPMismatch  = errors.New("code OTP not match")
	ErrOTPAttempts  = errors.New("terlalu banyak percobaan kode OTP")
	ErrRateLimited  = errors.New("terlalu banyak permintaan, coba lagi nanti")
	ErrEmailInvalid = errors.New("email tidak valid")
)

// NormalizeEmail menyeragamkan email untuk penulisan dan pencarian.
func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidateOTP memeriksa kode terhadap baris OTP aktif.
// Fungsi murni: tidak menyentuh database, sehingga bisa diuji langsung.
func ValidateOTP(o *Otp, code string, now time.Time, maxAttempts int) error {
	if o == nil {
		return ErrOTPNotFound
	}
	if o.UsedAt != nil {
		return ErrOTPUsed
	}
	if o.Attempts >= maxAttempts {
		return ErrOTPAttempts
	}
	if now.After(o.ExpiresAt) {
		return ErrOTPExpired
	}
	if o.Code != code {
		return ErrOTPMismatch
	}
	return nil
}

// CanResend melaporkan apakah kode boleh dikirim ulang sekarang, yaitu bila
// jeda sejak kiriman terakhir sudah melewati cooldown.
func CanResend(o *Otp, now time.Time, cooldown time.Duration) bool {
	if o == nil {
		return true
	}
	last := o.CreatedAt
	if o.LastSentAt != nil {
		last = *o.LastSentAt
	}
	return now.Sub(last) >= cooldown
}
