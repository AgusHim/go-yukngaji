package gamification

import (
	"time"

	"mainyuk/internal/audit"
	"mainyuk/internal/community"

	"github.com/gin-gonic/gin"
)

// Jenis sumber XP. Nilainya dipakai sebagai bagian dari kunci dedup, jadi
// jangan diubah tanpa migrasi data.
const (
	SourceProfileComplete = "profile_complete"
	SourceCheckIn         = "checkin"
	SourceMission         = "mission"
	SourceAdjustment      = "adjustment"
	// SourceDonation adalah reward untuk donasi yang pembayarannya sudah
	// terkonfirmasi. Besarannya tetap, bukan proporsional nominal, supaya XP
	// tidak bisa "dibeli" dengan memperbesar donasi.
	SourceDonation = "donation"
	// SourceDonationReversal membalik reward donasi saat donasi di-refund.
	// Dipisah dari SourceAdjustment supaya pembalikan otomatis dapat dibedakan
	// dari koreksi manual, dan supaya tidak ikut terhitung pada batas XP donasi.
	SourceDonationReversal = "donation_reversal"
	// SourceShopOrder adalah reward untuk pesanan merchandise yang
	// pembayarannya sudah terkonfirmasi. Sama seperti donasi, besarannya tetap
	// dan bukan proporsional nominal, supaya XP tidak bisa "dibeli".
	SourceShopOrder = "shop_order"
	// SourceShopOrderReversal membalik reward pesanan saat pesanannya
	// di-refund. Dipisah dari SourceAdjustment dengan alasan yang sama seperti
	// pembalikan donasi.
	SourceShopOrderReversal = "shop_order_reversal"
)

// Periode leaderboard.
const (
	PeriodWeekly  = "weekly"
	PeriodMonthly = "monthly"
	PeriodAllTime = "all_time"
)

// Jenis entitas pada jejak audit. Ketiganya adalah satu-satunya jenis yang
// ditulis modul ini, sehingga daftarnya sekaligus menjadi penyaring bacaannya
// di audit_logs — tabel itu dipakai bersama modul dana, moderasi, dan toko.
const (
	// entityXPAdjustment menunjuk satu baris xp_ledger hasil koreksi manual.
	entityXPAdjustment = "xp_adjustment"
	// entityXPRule memakai source_type sebagai id, karena aturan XP memang
	// dikenali per sumber.
	entityXPRule = "xp_rule"
	// entityLevelRule tidak punya id tunggal: kurva level selalu ditimpa
	// seluruhnya, jadi entity_id-nya kosong dan jumlah levelnya dicatat di
	// detail.
	entityLevelRule = "level_rule"
)

// auditEntityTypes adalah cakupan jejak audit milik modul ini.
var auditEntityTypes = []string{entityXPAdjustment, entityXPRule, entityLevelRule}

// LedgerEntry adalah satu baris ledger XP. Ledger bersifat append-only:
// koreksi berupa baris penyesuaian baru, bukan perubahan atau penghapusan.
type LedgerEntry struct {
	ID          string    `json:"id" gorm:"column:id;primaryKey"`
	UserID      string    `json:"-" gorm:"column:user_id"`
	Delta       int       `json:"delta" gorm:"column:delta"`
	SourceType  string    `json:"source_type" gorm:"column:source_type"`
	RefType     *string   `json:"ref_type" gorm:"column:ref_type"`
	RefID       *string   `json:"ref_id" gorm:"column:ref_id"`
	DedupKey    string    `json:"-" gorm:"column:dedup_key"`
	Note        *string   `json:"note" gorm:"column:note"`
	ActorUserID *string   `json:"-" gorm:"column:actor_user_id"`
	CreatedAt   time.Time `json:"created_at" gorm:"column:created_at"`
}

func (LedgerEntry) TableName() string {
	return "xp_ledger"
}

// LevelRule adalah satu tingkat pada kurva level. Seluruh isinya data, bukan
// konstanta kode, sehingga pengurus dapat menyesuaikannya tanpa deploy.
type LevelRule struct {
	ID           string  `json:"id" gorm:"column:id;primaryKey"`
	Level        int     `json:"level" gorm:"column:level"`
	Name         string  `json:"name" gorm:"column:name"`
	MinXP        int     `json:"min_xp" gorm:"column:min_xp"`
	BadgeLabel   *string `json:"badge_label" gorm:"column:badge_label"`
	BadgeIconURL *string `json:"badge_icon_url" gorm:"column:badge_icon_url"`
}

func (LevelRule) TableName() string {
	return "level_rules"
}

// XPRule adalah besaran XP untuk sumber tetap (profil lengkap, check-in).
// Reward misi disimpan per misi, bukan di sini.
//
// MonthlyCap adalah batas XP per bulan kalender Asia/Jakarta untuk sumber ini.
// NULL berarti tanpa batas. Batas disimpan sebagai data, bukan konstanta kode,
// supaya pengurus dapat menyesuaikannya tanpa deploy — alasan yang sama dengan
// besaran XP itu sendiri. Hanya sumber donasi yang memakainya saat ini.
type XPRule struct {
	ID          string  `json:"id" gorm:"column:id;primaryKey"`
	SourceType  string  `json:"source_type" gorm:"column:source_type"`
	XP          int     `json:"xp" gorm:"column:xp"`
	Description *string `json:"description" gorm:"column:description"`
	MonthlyCap  *int    `json:"monthly_cap" gorm:"column:monthly_cap"`
}

func (XPRule) TableName() string {
	return "xp_rules"
}

// UpdateXPRule adalah payload admin untuk mengubah besaran XP satu sumber.
//
// MonthlyCap yang tidak dikirim (nil) berarti tanpa batas, bukan "jangan
// diubah": aturan XP selalu dikirim utuh oleh form admin, sehingga menghapus
// batas cukup dengan mengosongkan kolomnya.
type UpdateXPRule struct {
	XP          int     `json:"xp" binding:"required"`
	Description *string `json:"description"`
	MonthlyCap  *int    `json:"monthly_cap"`
}

// AdjustXP adalah koreksi XP oleh admin. Alasan wajib diisi agar jejak
// ledger tetap dapat diaudit.
type AdjustXP struct {
	UserID string `json:"user_id" binding:"required"`
	Delta  int    `json:"delta" binding:"required"`
	Reason string `json:"reason" binding:"required"`
}

// LevelRuleInput adalah satu baris kurva level yang dikirim admin. Dipisah
// dari LevelRule supaya client tidak dapat menyetel id/kolom di luar kontrak.
type LevelRuleInput struct {
	Level        int     `json:"level" binding:"required"`
	Name         string  `json:"name" binding:"required"`
	MinXP        int     `json:"min_xp"`
	BadgeLabel   *string `json:"badge_label"`
	BadgeIconURL *string `json:"badge_icon_url"`
}

// UpdateLevelRules adalah payload admin untuk menimpa seluruh kurva level
// sekaligus, supaya kurva tidak pernah tersimpan dalam keadaan setengah jadi.
type UpdateLevelRules struct {
	Rules []*LevelRuleInput `json:"rules" binding:"required"`
}

// NextLevelInfo menggambarkan level berikutnya yang belum dicapai.
type NextLevelInfo struct {
	Level int    `json:"level"`
	Name  string `json:"name"`
	MinXP int    `json:"min_xp"`
}

// XPSummary adalah ringkasan XP milik pemilik akun.
type XPSummary struct {
	TotalXP         int            `json:"total_xp"`
	Level           int            `json:"level"`
	LevelName       string         `json:"level_name"`
	Badge           string         `json:"badge"`
	ProgressPercent int            `json:"progress_percent"`
	RemainingXP     int            `json:"remaining_xp"`
	NextLevel       *NextLevelInfo `json:"next_level"`
}

// ProfileEnsurer memastikan baris profil komunitas ada sebelum ledger ditulis,
// supaya pemilik XP tidak hilang dari leaderboard yang JOIN ke profil.
// Dipenuhi oleh community.Service.
type ProfileEnsurer interface {
	EnsureProfile(ctx *gin.Context, userID string) (*community.Profile, error)
}

type Repository interface {
	// GrantOnce menulis satu entri ledger bila dedup_key belum pernah ada.
	// Mengembalikan true hanya bila baris baru benar-benar tersisip.
	GrantOnce(ctx *gin.Context, entry *LedgerEntry) (bool, error)
	// GrantCappedOnce memberi reward bersumber tunggal di dalam satu
	// transaksi: kunci advisory per akun, hitung reward sumber itu pada bulan
	// berjalan, putuskan batas, lalu sisipkan. Mengembalikan XP yang berlaku —
	// delta yang baru tersisip, atau delta yang sudah ada bila sumber ini
	// pernah diberi.
	//
	// sourceType dan lockKey dijadikan parameter karena sumber berbatas bulanan
	// sudah lebih dari satu (donasi dan pesanan merchandise); aturan hitungnya
	// sendiri identik, jadi tidak ada gunanya menyalinnya per sumber.
	GrantCappedOnce(ctx *gin.Context, entry *LedgerEntry, sourceType, lockKey string, from, to time.Time, ruleXP, monthlyCap int) (int, error)
	TotalXP(ctx *gin.Context, userID string) (int, error)
	History(ctx *gin.Context, userID string, limit, offset int) ([]*LedgerEntry, error)
	LevelRules(ctx *gin.Context) ([]*LevelRule, error)
	UpdateLevelRules(ctx *gin.Context, rules []*LevelRule) error
	XPRules(ctx *gin.Context) ([]*XPRule, error)
	XPRule(ctx *gin.Context, sourceType string) (*XPRule, error)
	UpdateXPRule(ctx *gin.Context, sourceType string, xp int, description *string, monthlyCap *int) error
	Leaderboard(ctx *gin.Context, from, to *time.Time, limit, offset int) ([]*LeaderboardRow, error)
	// WriteAudit menulis satu baris jejak audit lewat penulis bersama di
	// internal/audit. Modul ini bukan pemilik tunggal tabel itu.
	WriteAudit(ctx *gin.Context, entry *audit.Log) error
	// ListAudit membaca riwayat modul ini. entityTypes adalah cakupan modul;
	// entityID menyaring satu entitas bila diisi.
	ListAudit(ctx *gin.Context, entityTypes []string, entityID string, limit, offset int) ([]*audit.Log, error)
}

type Service interface {
	// OnProfileCompleted dipanggil best-effort setelah profil diperbarui.
	// Tidak melakukan apa pun bila profil belum lengkap atau XP-nya sudah
	// pernah diberikan.
	OnProfileCompleted(ctx *gin.Context, userID string) error

	// OnVerifiedCheckIn dipanggil best-effort setelah check-in tervalidasi.
	// participantUserID harus peserta terverifikasi, bukan pembeli tiket.
	OnVerifiedCheckIn(ctx *gin.Context, participantUserID, eventID string) error

	// GrantMissionReward memberi XP untuk klaim misi yang sudah disetujui.
	GrantMissionReward(ctx *gin.Context, userID, claimID string, xp int) (bool, error)

	// GrantDonationReward memberi XP untuk donasi yang pembayarannya sudah
	// terkonfirmasi. Mengembalikan XP yang benar-benar berlaku (0 bila aturan
	// kosong atau batas bulanan sudah tercapai) supaya pemanggil dapat
	// menyimpannya sebagai snapshot untuk pembalikan saat refund.
	GrantDonationReward(ctx *gin.Context, userID, donationID string) (int, error)

	// ReverseDonationReward membalik XP donasi yang di-refund. Idempoten lewat
	// kunci dedup deterministik, jadi mengulang refund tidak menggandakan
	// pembalikan. Alasan wajib diisi agar jejak ledger tetap dapat diaudit.
	ReverseDonationReward(ctx *gin.Context, userID, donationID string, xp int, reason string) error

	// GrantShopOrderReward memberi XP untuk pesanan merchandise yang
	// pembayarannya sudah terkonfirmasi. Bentuknya sengaja sama persis dengan
	// pasangan donasi — termasuk batas bulanannya — supaya pemanggil dapat
	// memperlakukan keduanya dengan cara yang sama.
	GrantShopOrderReward(ctx *gin.Context, userID, orderID string) (int, error)

	// ReverseShopOrderReward membalik XP pesanan yang di-refund. Idempoten
	// lewat kunci dedup deterministik, jadi mengulang refund tidak
	// menggandakan pembalikan.
	ReverseShopOrderReward(ctx *gin.Context, userID, orderID string, xp int, reason string) error

	Summary(ctx *gin.Context) (*XPSummary, error)
	History(ctx *gin.Context, page, perPage int) ([]*LedgerEntry, bool, error)
	Adjust(ctx *gin.Context, req *AdjustXP) (*LedgerEntry, error)
	LevelRules(ctx *gin.Context) ([]*LevelRule, error)
	UpdateLevelRules(ctx *gin.Context, req *UpdateLevelRules) ([]*LevelRule, error)
	XPRules(ctx *gin.Context) ([]*XPRule, error)
	UpdateXPRule(ctx *gin.Context, sourceType string, req *UpdateXPRule) (*XPRule, error)
	Leaderboard(ctx *gin.Context, period string, page, perPage int) (*LeaderboardPage, error)
	// AuditLogs menampilkan jejak koreksi XP dan perubahan aturan. entityType
	// dan entityID bersifat opsional; keduanya berasal dari query string.
	AuditLogs(ctx *gin.Context, entityType, entityID string, page, perPage int) ([]*audit.Log, bool, error)
}

type Handler interface {
	Summary(ctx *gin.Context)
	History(ctx *gin.Context)
	Adjust(ctx *gin.Context)
	LevelRules(ctx *gin.Context)
	UpdateLevelRules(ctx *gin.Context)
	XPRules(ctx *gin.Context)
	UpdateXPRule(ctx *gin.Context)
	Leaderboard(ctx *gin.Context)
	AuditLogs(ctx *gin.Context)
}
