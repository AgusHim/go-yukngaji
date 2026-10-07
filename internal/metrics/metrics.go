// Package metrics menyusun satu laporan angka untuk pengurus.
//
// Seluruh isinya adalah pembacaan agregat atas tabel yang sudah ada: tidak ada
// tabel baru, tidak ada migrasi, dan tidak ada penulisan apa pun. Laporan ini
// juga tidak pernah memuat identitas — hanya angka dan rentang tanggal.
//
// Yang dimaksud "akun aktif" di sini perlu dibaca dengan hati-hati. Skema ini
// tidak menyimpan jejak sesi sama sekali, jadi keaktifan diturunkan dari aksi
// bermakna: mendaftar event, berdonasi, mengklaim misi, berbelanja, menulis,
// atau menerima XP. Akun yang hanya membaca tidak akan pernah terhitung. Itu
// keterbatasan skema, bukan kekeliruan perhitungan.
package metrics

import (
	"time"

	"mainyuk/internal/period"

	"github.com/gin-gonic/gin"
)

// Window adalah satu rentang waktu [From, To) beserta angka di dalamnya.
//
// To bersifat eksklusif. Untuk jendela bulan kalender ia boleh berada di masa
// depan; yang terhitung tetap hanya baris yang sudah ada.
type Window struct {
	From  time.Time `json:"from"`
	To    time.Time `json:"to"`
	Value int64     `json:"value"`
}

// Windows adalah himpunan rentang yang dipakai laporan ini.
//
// Dipisah dari service supaya perhitungannya dapat diuji tanpa database.
type Windows struct {
	// Now adalah titik acuan; jendela bergulir dihitung mundur darinya.
	Now time.Time
	// MonthFrom dan MonthTo membatasi bulan kalender Asia/Jakarta berjalan.
	MonthFrom time.Time
	MonthTo   time.Time
	// Day7From dan Day30From adalah awal jendela bergulir 7 dan 30 hari.
	Day7From  time.Time
	Day30From time.Time
}

// WindowsFor menghitung rentang laporan untuk waktu now.
//
// MAU memakai bulan kalender berjalan di Asia/Jakarta, sedangkan D7 dan D30
// adalah jendela bergulir ke belakang dari now — bukan minggu/bulan kalender.
// loc nil berarti Jakarta.
func WindowsFor(now time.Time, loc *time.Location) Windows {
	if loc == nil {
		loc = period.JakartaLocation
	}
	monthFrom, monthTo := period.MonthRange(now, loc)
	return Windows{
		Now:       now,
		MonthFrom: monthFrom,
		MonthTo:   monthTo,
		Day7From:  now.AddDate(0, 0, -7),
		Day30From: now.AddDate(0, 0, -30),
	}
}

// ActiveUsers adalah jumlah akun yang melakukan aksi bermakna.
type ActiveUsers struct {
	// CalendarMonth adalah bulan kalender berjalan — inilah MAU.
	CalendarMonth Window `json:"calendar_month"`
	Last7Days     Window `json:"last_7_days"`
	Last30Days    Window `json:"last_30_days"`
}

// MissionMetrics merangkum klaim misi yang disetujui.
type MissionMetrics struct {
	ApprovedTotal   int64  `json:"approved_total"`
	ApprovedInMonth Window `json:"approved_in_month"`
}

// DonationMetrics merangkum donasi yang pembayarannya terkonfirmasi.
type DonationMetrics struct {
	PaidTotal   int64  `json:"paid_total"`
	PaidInMonth Window `json:"paid_in_month"`
}

// OrderMetrics merangkum pesanan merchandise.
//
// Kedua peta dihitung dari seluruh riwayat, bukan hanya jendela berjalan:
// gunanya memantau berapa banyak pesanan yang masih menggantung, dan itu baru
// terlihat dari totalnya.
type OrderMetrics struct {
	ByPaymentStatus     map[string]int64 `json:"by_payment_status"`
	ByFulfillmentStatus map[string]int64 `json:"by_fulfillment_status"`
	CreatedInMonth      Window           `json:"created_in_month"`
}

// CommunityMetrics merangkum aktivitas dan laporan di ruang komunitas.
type CommunityMetrics struct {
	ThreadsInMonth   Window           `json:"threads_in_month"`
	CommentsInMonth  Window           `json:"comments_in_month"`
	ReactionsInMonth Window           `json:"reactions_in_month"`
	ReportsInMonth   Window           `json:"reports_in_month"`
	ReportsByStatus  map[string]int64 `json:"reports_by_status"`
}

// View adalah bentuk respons endpoint ini.
//
// Tidak ada satu pun bidang yang memuat id akun, nama, atau email. Angka
// agregat inilah satu-satunya yang boleh keluar dari endpoint ini.
type View struct {
	GeneratedAt time.Time        `json:"generated_at"`
	ActiveUsers ActiveUsers      `json:"active_users"`
	Missions    MissionMetrics   `json:"missions"`
	Donations   DonationMetrics  `json:"donations"`
	Orders      OrderMetrics     `json:"orders"`
	Community   CommunityMetrics `json:"community"`
}

// Repository membaca seluruh angka laporan dari database.
//
// Rentang waktu selalu [from, to) dan selalu ditentukan pemanggil, sehingga
// perhitungan tanggalnya dapat diuji terpisah dari SQL-nya.
type Repository interface {
	// ActiveAccounts menghitung akun unik yang melakukan aksi bermakna dalam
	// rentang, dari gabungan seluruh modul.
	ActiveAccounts(ctx *gin.Context, from, to time.Time) (int64, error)

	ApprovedClaims(ctx *gin.Context, from, to time.Time) (int64, error)
	ApprovedClaimsTotal(ctx *gin.Context) (int64, error)

	ConfirmedDonations(ctx *gin.Context, from, to time.Time) (int64, error)
	ConfirmedDonationsTotal(ctx *gin.Context) (int64, error)

	ShopOrders(ctx *gin.Context, from, to time.Time) (int64, error)
	ShopOrdersByPaymentStatus(ctx *gin.Context) (map[string]int64, error)
	ShopOrdersByFulfillmentStatus(ctx *gin.Context) (map[string]int64, error)

	Threads(ctx *gin.Context, from, to time.Time) (int64, error)
	ThreadComments(ctx *gin.Context, from, to time.Time) (int64, error)
	ThreadReactions(ctx *gin.Context, from, to time.Time) (int64, error)
	ThreadReports(ctx *gin.Context, from, to time.Time) (int64, error)
	ThreadReportsByStatus(ctx *gin.Context) (map[string]int64, error)
}

type Service interface {
	// Show menyusun laporan untuk saat ini.
	Show(ctx *gin.Context) (*View, error)
}

type Handler interface {
	Show(ctx *gin.Context)
}
