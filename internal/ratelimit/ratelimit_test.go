package ratelimit

import (
	"errors"
	"testing"
	"time"
)

func TestAllowWithinWindow(t *testing.T) {
	now := time.Date(2026, time.October, 2, 12, 0, 0, 0, time.UTC)
	l := New(3, time.Minute)
	l.now = func() time.Time { return now }

	for i := 0; i < 3; i++ {
		if !l.Allow("user:a") {
			t.Fatalf("panggilan ke-%d harus diizinkan", i+1)
		}
	}
	if l.Allow("user:a") {
		t.Error("panggilan ke-4 harus ditolak")
	}

	// Kunci lain tidak terpengaruh.
	if !l.Allow("user:b") {
		t.Error("kunci berbeda harus punya jatah sendiri")
	}
}

func TestAllowSlidingWindow(t *testing.T) {
	now := time.Date(2026, time.October, 2, 12, 0, 0, 0, time.UTC)
	l := New(2, time.Minute)
	l.now = func() time.Time { return now }

	if !l.Allow("k") || !l.Allow("k") {
		t.Fatal("dua panggilan pertama harus diizinkan")
	}
	if l.Allow("k") {
		t.Fatal("panggilan ketiga harus ditolak")
	}

	// Setelah jendela bergeser, jatah pulih.
	now = now.Add(61 * time.Second)
	if !l.Allow("k") {
		t.Error("setelah lewat jendela, permintaan harus diizinkan lagi")
	}
}

func TestReset(t *testing.T) {
	now := time.Date(2026, time.October, 2, 12, 0, 0, 0, time.UTC)
	l := New(1, time.Minute)
	l.now = func() time.Time { return now }

	if !l.Allow("k") {
		t.Fatal("panggilan pertama harus diizinkan")
	}
	if l.Allow("k") {
		t.Fatal("panggilan kedua harus ditolak")
	}

	l.Reset("k")
	if !l.Allow("k") {
		t.Error("setelah Reset, permintaan harus diizinkan lagi")
	}
}

func TestNilAndZeroLimiterAlwaysAllow(t *testing.T) {
	var nilLimiter *Limiter
	if !nilLimiter.Allow("k") {
		t.Error("limiter nil harus selalu mengizinkan")
	}
	zero := New(0, time.Minute)
	if !zero.Allow("k") {
		t.Error("limit 0 berarti tanpa pembatas")
	}
}

func TestIsLimited(t *testing.T) {
	if !IsLimited(ErrTooManyRequests) {
		t.Error("ErrTooManyRequests harus dikenali")
	}
	if IsLimited(errors.New("error lain")) {
		t.Error("error lain tidak boleh dianggap rate limit")
	}
	if IsLimited(nil) {
		t.Error("nil bukan rate limit")
	}
}
