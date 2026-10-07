package otp

import (
	"errors"
	"testing"
	"time"
)

func TestNormalizeEmail(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"  Budi@Example.COM ", "budi@example.com"},
		{"budi@example.com", "budi@example.com"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := NormalizeEmail(tt.in); got != tt.want {
			t.Errorf("NormalizeEmail(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidateOTP(t *testing.T) {
	now := time.Date(2026, time.October, 2, 12, 0, 0, 0, time.UTC)
	used := now.Add(-time.Minute)

	base := func() *Otp {
		return &Otp{
			Code:      "123456",
			ExpiresAt: now.Add(10 * time.Minute),
			CreatedAt: now.Add(-time.Minute),
		}
	}

	tests := []struct {
		name        string
		otp         *Otp
		code        string
		maxAttempts int
		wantErr     error
	}{
		{name: "kode benar", otp: base(), code: "123456", maxAttempts: MaxAttempts},
		{name: "otp tidak ada", otp: nil, code: "123456", maxAttempts: MaxAttempts, wantErr: ErrOTPNotFound},
		{
			name:        "kode sudah dipakai tidak bisa diulang",
			otp:         func() *Otp { o := base(); o.UsedAt = &used; return o }(),
			code:        "123456",
			maxAttempts: MaxAttempts,
			wantErr:     ErrOTPUsed,
		},
		{
			name:        "percobaan habis",
			otp:         func() *Otp { o := base(); o.Attempts = MaxAttempts; return o }(),
			code:        "123456",
			maxAttempts: MaxAttempts,
			wantErr:     ErrOTPAttempts,
		},
		{
			name:        "kode kedaluwarsa",
			otp:         func() *Otp { o := base(); o.ExpiresAt = now.Add(-time.Second); return o }(),
			code:        "123456",
			maxAttempts: MaxAttempts,
			wantErr:     ErrOTPExpired,
		},
		{
			name:        "kode salah",
			otp:         base(),
			code:        "000000",
			maxAttempts: MaxAttempts,
			wantErr:     ErrOTPMismatch,
		},
		{
			name:        "kode dipakai lewat batas percobaan dilaporkan sebagai percobaan habis",
			otp:         func() *Otp { o := base(); o.Attempts = MaxAttempts + 3; return o }(),
			code:        "123456",
			maxAttempts: MaxAttempts,
			wantErr:     ErrOTPAttempts,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateOTP(tt.otp, tt.code, now, tt.maxAttempts)
			if tt.wantErr == nil {
				if err != nil {
					t.Fatalf("tidak boleh error, dapat: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateOTPExpiryCheckedBeforeCode(t *testing.T) {
	now := time.Date(2026, time.October, 2, 12, 0, 0, 0, time.UTC)
	o := &Otp{
		Code:      "123456",
		ExpiresAt: now.Add(-time.Minute),
		CreatedAt: now.Add(-time.Hour),
	}

	// Kode yang sudah kedaluwarsa harus dilaporkan kedaluwarsa walaupun kode
	// yang dikirim juga salah: jangan membocorkan bahwa kodenya "hampir benar".
	err := ValidateOTP(o, "000000", now, MaxAttempts)
	if !errors.Is(err, ErrOTPExpired) {
		t.Errorf("error = %v, want %v", err, ErrOTPExpired)
	}
}

func TestCanResend(t *testing.T) {
	now := time.Date(2026, time.October, 2, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		otp  *Otp
		want bool
	}{
		{name: "belum ada kode boleh kirim", otp: nil, want: true},
		{
			name: "baru dikirim belum boleh",
			otp:  &Otp{CreatedAt: now.Add(-10 * time.Second)},
			want: false,
		},
		{
			name: "lewat cooldown boleh",
			otp:  &Otp{CreatedAt: now.Add(-2 * time.Minute)},
			want: true,
		},
		{
			name: "last_sent_at dipakai bila ada",
			otp: func() *Otp {
				last := now.Add(-10 * time.Second)
				return &Otp{CreatedAt: now.Add(-time.Hour), LastSentAt: &last}
			}(),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanResend(tt.otp, now, ResendCooldown); got != tt.want {
				t.Errorf("CanResend = %v, want %v", got, tt.want)
			}
		})
	}
}
