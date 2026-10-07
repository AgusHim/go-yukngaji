// Package ratelimit menyediakan pembatas laju in-memory sederhana.
//
// Ini bukan pengganti pembatas terdistribusi: hitungannya per proses, jadi
// hanya cocok untuk deployment satu instance (lihat docs/deployment.md).
// Tujuannya meredam percobaan berulang pada endpoint auth dan aksi tulis.
package ratelimit

import (
	"errors"
	"net"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ErrTooManyRequests dikembalikan service saat pembatas laju menolak aksi.
// Handler memetakannya ke HTTP 429.
var ErrTooManyRequests = errors.New("TooManyRequests")

// IsLimited melaporkan apakah err berasal dari penolakan pembatas laju.
func IsLimited(err error) bool {
	return errors.Is(err, ErrTooManyRequests)
}

// Limiter membatasi jumlah kejadian per kunci dalam jendela waktu bergulir.
type Limiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
	now    func() time.Time
}

// New membuat limiter yang mengizinkan paling banyak limit kejadian per
// kunci dalam window.
func New(limit int, window time.Duration) *Limiter {
	return &Limiter{
		hits:   make(map[string][]time.Time),
		limit:  limit,
		window: window,
		now:    time.Now,
	}
}

// Allow mencatat satu kejadian untuk key dan melaporkan apakah masih dalam
// batas. Kejadian di luar jendela dibuang lebih dulu.
func (l *Limiter) Allow(key string) bool {
	if l == nil || l.limit <= 0 {
		return true
	}

	now := l.now()
	cutoff := now.Add(-l.window)

	l.mu.Lock()
	defer l.mu.Unlock()

	kept := l.hits[key][:0]
	for _, t := range l.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}

	if len(kept) >= l.limit {
		l.hits[key] = kept
		return false
	}

	l.hits[key] = append(kept, now)
	return true
}

// Reset menghapus riwayat untuk key, misalnya setelah aksi berhasil.
func (l *Limiter) Reset(key string) {
	if l == nil {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.hits, key)
}

// Limiters yang dipakai aplikasi. Semuanya per proses.
var (
	// Login membatasi percobaan login per alamat IP.
	Login = New(10, time.Minute)
	// OTPRequest membatasi permintaan OTP per email.
	OTPRequest = New(5, time.Hour)
	// OTPVerify membatasi percobaan verifikasi OTP per email.
	OTPVerify = New(10, time.Minute)
	// Comment membatasi penulisan komentar per identitas.
	Comment = New(20, time.Minute)
	// Like membatasi penulisan like per identitas.
	Like = New(60, time.Minute)
	// PollResponse membatasi kiriman jawaban poll per identitas.
	PollResponse = New(30, time.Minute)
	// Presence membatasi pendaftaran kehadiran per identitas.
	Presence = New(20, time.Minute)
	// Feedback membatasi kiriman masukan per identitas.
	Feedback = New(10, time.Minute)
	// MissionClaim membatasi pembuatan klaim misi per identitas. Batas ini
	// menahan percobaan klaim beruntun atas misi berbatas klaim besar.
	MissionClaim = New(20, time.Minute)
	// XpAdjust membatasi koreksi XP manual oleh pengurus per identitas.
	XpAdjust = New(30, time.Minute)
	// Donation membatasi pembuatan donasi per identitas. Selain meredam
	// kiriman beruntun, batas ini menahan percobaan memanen XP donasi dengan
	// membuat banyak donasi kecil.
	Donation = New(10, time.Minute)
	// DonationReview membatasi keputusan verifikasi/refund donasi oleh
	// pengurus per identitas.
	DonationReview = New(60, time.Minute)

	// ShopOrder membatasi pembuatan pesanan merchandise per identitas.
	// Checkout menahan stok fisik; tanpa batas, satu akun dapat mengunci
	// seluruh persediaan dengan pesanan yang tidak pernah dibayar.
	ShopOrder = New(10, time.Hour)
	// ShopProof membatasi penyerahan bukti transfer merchandise per identitas.
	ShopProof = New(10, time.Hour)
	// ShopReview membatasi keputusan verifikasi/pemenuhan pesanan merchandise
	// oleh pengurus per identitas, sepadan DonationReview.
	ShopReview = New(60, time.Minute)

	// Thread membatasi pembuatan thread per identitas. Batasnya jauh lebih
	// ketat daripada komentar: kiriman adalah entri baru di feed, sedangkan
	// komentar menempel pada kiriman yang sudah ada.
	Thread = New(5, time.Hour)
	// ThreadComment membatasi penulisan komentar feed per identitas.
	ThreadComment = New(20, time.Minute)
	// ThreadReaction membatasi penulisan reaksi feed per identitas.
	ThreadReaction = New(60, time.Minute)
	// ThreadReport membatasi pengiriman laporan per identitas. Ini yang
	// menahan penyalahgunaan tombol lapor sebagai alat penekan.
	ThreadReport = New(10, time.Hour)
)

// Key menyusun kunci pembatas dari identitas yang sudah diverifikasi server.
// identity kosong (tamu) jatuh ke alamat IP klien.
func Key(c *gin.Context, identity string) string {
	if identity != "" {
		return "user:" + identity
	}
	return "ip:" + ClientIP(c)
}

// ClientIP mengembalikan alamat IP klien tanpa port.
func ClientIP(c *gin.Context) string {
	if c == nil {
		return "unknown"
	}
	ip := c.ClientIP()
	if host, _, err := net.SplitHostPort(ip); err == nil {
		return host
	}
	return ip
}
