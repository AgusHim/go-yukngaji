// Package thread mengelola feed komunitas anonim: thread, komentar, reaksi,
// laporan, moderasi, dan auto-post dari aktivitas anggota.
//
// Paket ini terpisah dari internal/comment yang menangani QnA per event. Kedua
// permukaan itu punya aturan identitas yang berbeda: komentar event menampilkan
// username akun, sedangkan feed ini tidak pernah menampilkan apa pun selain
// alias — dan menampilkan "Anonim" bila aliasnya kosong.
package thread

import (
	"time"

	"mainyuk/internal/audit"
	"mainyuk/internal/community"

	"github.com/gin-gonic/gin"
)

// Status konten. Dipakai bersama oleh thread dan komentar.
const (
	StatusPublished = "published"
	StatusHidden    = "hidden"
	StatusDeleted   = "deleted"
)

// Jenis aktivitas sumber auto-post. Nilainya bagian dari kunci dedup
// (source_type, source_ref) dan tersimpan di database — jangan diubah tanpa
// migrasi data.
const (
	SourceEventRegistration = "event_registration"
	SourceDonation          = "donation"
	SourceMission           = "mission"
)

// Urutan feed.
const (
	SortTerbaru = "terbaru"
	SortPopuler = "populer"
)

// Jenis target laporan.
const (
	TargetThread        = "thread"
	TargetThreadComment = "thread_comment"
)

// Alasan laporan. Nilainya tersimpan di database.
const (
	ReasonSpam        = "spam"
	ReasonSara        = "sara"
	ReasonPornografi  = "pornografi"
	ReasonPenipuan    = "penipuan"
	ReasonPerundungan = "perundungan"
	ReasonLainnya     = "lainnya"
)

// Status laporan.
const (
	ReportOpen      = "open"
	ReportActioned  = "actioned"
	ReportDismissed = "dismissed"
)

// AnonymousAuthorName adalah nama tampil penulis yang identitasnya tidak boleh
// dibuka. Sama dengan nama donatur anonim supaya istilahnya konsisten.
const AnonymousAuthorName = "Anonim"

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

// Thread adalah satu kiriman feed. Thread manual tidak punya SourceType;
// auto-post selalu punya keduanya.
type Thread struct {
	ID         string  `json:"id" gorm:"column:id;primaryKey"`
	PublicID   string  `json:"public_id" gorm:"column:public_id"`
	UserID     string  `json:"-" gorm:"column:user_id"`
	Title      string  `json:"title" gorm:"column:title"`
	Body       string  `json:"body" gorm:"column:body"`
	Status     string  `json:"status" gorm:"column:status"`
	SourceType *string `json:"source_type" gorm:"column:source_type"`
	SourceRef  *string `json:"-" gorm:"column:source_ref"`

	CommentCount  int `json:"comment_count" gorm:"column:comment_count"`
	ReactionCount int `json:"reaction_count" gorm:"column:reaction_count"`

	HiddenBy       *string    `json:"hidden_by" gorm:"column:hidden_by"`
	HiddenAt       *time.Time `json:"hidden_at" gorm:"column:hidden_at"`
	DecisionReason *string    `json:"decision_reason" gorm:"column:decision_reason"`

	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time  `json:"-" gorm:"column:updated_at"`
	DeletedAt *time.Time `json:"-" gorm:"column:deleted_at"`
}

func (Thread) TableName() string { return "threads" }

// ThreadComment adalah satu komentar datar pada sebuah thread. Tidak ada
// balasan bersarang: kedalaman satu sudah cukup untuk kebutuhan sekarang dan
// menghapus seluruh kelas masalah penomoran.
type ThreadComment struct {
	ID       string `json:"id" gorm:"column:id;primaryKey"`
	PublicID string `json:"public_id" gorm:"column:public_id"`
	ThreadID string `json:"-" gorm:"column:thread_id"`
	UserID   string `json:"-" gorm:"column:user_id"`
	Body     string `json:"body" gorm:"column:body"`
	Status   string `json:"status" gorm:"column:status"`

	HiddenBy       *string    `json:"hidden_by" gorm:"column:hidden_by"`
	HiddenAt       *time.Time `json:"hidden_at" gorm:"column:hidden_at"`
	DecisionReason *string    `json:"decision_reason" gorm:"column:decision_reason"`

	CreatedAt time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time  `json:"-" gorm:"column:updated_at"`
	DeletedAt *time.Time `json:"-" gorm:"column:deleted_at"`
}

func (ThreadComment) TableName() string { return "thread_comments" }

// ThreadReaction adalah satu reaksi anggota pada satu thread. Tanpa deleted_at:
// batal reaksi adalah hard delete, sehingga index unik (thread_id, user_id)
// cukup untuk menjamin satu reaksi per akun per thread.
type ThreadReaction struct {
	ID        string    `json:"id" gorm:"column:id;primaryKey"`
	ThreadID  string    `json:"-" gorm:"column:thread_id"`
	UserID    string    `json:"-" gorm:"column:user_id"`
	Kind      string    `json:"kind" gorm:"column:kind"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
}

func (ThreadReaction) TableName() string { return "thread_reactions" }

// Report adalah laporan anggota atas sebuah thread atau komentar.
type Report struct {
	ID             string  `json:"id" gorm:"column:id;primaryKey"`
	PublicID       string  `json:"public_id" gorm:"column:public_id"`
	ReporterUserID *string `json:"-" gorm:"column:reporter_user_id"`
	TargetType     string  `json:"target_type" gorm:"column:target_type"`
	TargetID       string  `json:"-" gorm:"column:target_id"`
	Reason         string  `json:"reason" gorm:"column:reason"`
	Note           *string `json:"note" gorm:"column:note"`
	Status         string  `json:"status" gorm:"column:status"`

	HandledBy      *string    `json:"handled_by" gorm:"column:handled_by"`
	HandledAt      *time.Time `json:"handled_at" gorm:"column:handled_at"`
	DecisionReason *string    `json:"decision_reason" gorm:"column:decision_reason"`

	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	UpdatedAt time.Time `json:"-" gorm:"column:updated_at"`
}

func (Report) TableName() string { return "thread_reports" }

// SharePrefs adalah preferensi berbagi aktivitas ke feed. Default-nya semua
// true (kebijakan opt-out); ketiadaan baris berarti semua true, sehingga akun
// lama tidak perlu di-backfill.
type SharePrefs struct {
	UserID                 string    `json:"-" gorm:"column:user_id;primaryKey"`
	ShareEventRegistration bool      `json:"share_event_registration" gorm:"column:share_event_registration"`
	ShareDonation          bool      `json:"share_donation" gorm:"column:share_donation"`
	ShareMission           bool      `json:"share_mission" gorm:"column:share_mission"`
	CreatedAt              time.Time `json:"-" gorm:"column:created_at"`
	UpdatedAt              time.Time `json:"-" gorm:"column:updated_at"`
}

func (SharePrefs) TableName() string { return "thread_share_prefs" }

// ---------------------------------------------------------------------------
// DTO
// ---------------------------------------------------------------------------

// ThreadAuthor adalah satu-satunya bentuk penulis yang boleh dikirim ke luar.
// Tidak ada id akun, email, telepon, atau data profil internal — lihat author.go.
type ThreadAuthor struct {
	PublicID   string  `json:"public_id"`
	Alias      string  `json:"alias"`
	AvatarURL  *string `json:"avatar_url"`
	ShowBadges bool    `json:"show_badges"`
}

// ThreadView adalah bentuk thread untuk jalur publik.
//
// Excerpt dipakai daftar feed, Body dipakai halaman detail — keduanya tidak
// pernah terisi bersamaan, supaya daftar tidak mengangkut seluruh isi kiriman.
type ThreadView struct {
	PublicID      string        `json:"public_id"`
	Title         string        `json:"title"`
	Body          string        `json:"body,omitempty"`
	Excerpt       string        `json:"excerpt,omitempty"`
	Author        *ThreadAuthor `json:"author"`
	Status        string        `json:"status"`
	SourceType    *string       `json:"source_type"`
	CommentCount  int           `json:"comment_count"`
	ReactionCount int           `json:"reaction_count"`
	Reacted       bool          `json:"reacted"`
	IsMine        bool          `json:"is_mine"`
	CreatedAt     time.Time     `json:"created_at"`
}

// ThreadCommentView adalah bentuk komentar untuk jalur publik.
type ThreadCommentView struct {
	PublicID  string        `json:"public_id"`
	Body      string        `json:"body"`
	Author    *ThreadAuthor `json:"author"`
	Status    string        `json:"status"`
	IsMine    bool          `json:"is_mine"`
	CreatedAt time.Time     `json:"created_at"`
}

// CreateThread adalah payload pembuatan thread. Tidak ada user_id: identitas
// selalu diambil dari konteks auth, tidak pernah dari body.
type CreateThread struct {
	Title string `json:"title" binding:"required"`
	Body  string `json:"body" binding:"required"`
}

// CreateThreadComment adalah payload pembuatan komentar.
type CreateThreadComment struct {
	Body string `json:"body" binding:"required"`
}

// CreateReport adalah payload laporan anggota. Target dikirim sebagai public_id
// supaya id internal tidak pernah melintasi kabel.
type CreateReport struct {
	TargetType string  `json:"target_type" binding:"required"`
	TargetID   string  `json:"target_id" binding:"required"`
	Reason     string  `json:"reason" binding:"required"`
	Note       *string `json:"note"`
}

// UpdateSharePrefs hanya mengubah field yang dikirim, sehingga form dapat
// menyimpan sebagian pengaturan tanpa menimpa sisanya.
type UpdateSharePrefs struct {
	ShareEventRegistration *bool `json:"share_event_registration"`
	ShareDonation          *bool `json:"share_donation"`
	ShareMission           *bool `json:"share_mission"`
}

// DecideContent adalah payload tindakan moderator atas konten. Alasan wajib.
type DecideContent struct {
	Reason string `json:"reason"`
}

// DecideReport adalah payload keputusan atas laporan. Alasan wajib.
// HideTarget menyembunyikan konten yang dilaporkan sekaligus.
type DecideReport struct {
	Reason     string `json:"reason"`
	HideTarget bool   `json:"hide_target"`
}

// RestrictAccount adalah payload pembatasan akun. Alasan wajib saat memblokir.
type RestrictAccount struct {
	Blocked bool   `json:"blocked"`
	Reason  string `json:"reason"`
}

// ReportTargetView merangkum konten yang dilaporkan, supaya antrean moderator
// tidak perlu memuat halaman thread satu per satu.
type ReportTargetView struct {
	Type     string `json:"type"`
	PublicID string `json:"public_id"`
	Title    string `json:"title"`
	Excerpt  string `json:"excerpt"`
	Status   string `json:"status"`
}

// ModerationReportView adalah satu baris antrean laporan.
type ModerationReportView struct {
	Report *Report           `json:"report"`
	Target *ReportTargetView `json:"target"`
	Author *ThreadAuthor     `json:"author"`
}

// Entity type yang muncul di audit_logs untuk tindakan moderasi. Dipakai
// sebagai saringan riwayat, supaya jejak moderasi tidak tercampur dengan jejak
// dana dan XP yang memakai tabel yang sama.
var moderationEntityTypes = []string{
	"thread", "thread_comment", "report", "account", "thread_share_prefs",
}

// isKnownTarget dan isKnownReason memeriksa input laporan di Go lebih dulu,
// supaya pemanggil mendapat pesan yang jelas alih-alih pelanggaran constraint.
func isKnownTarget(targetType string) bool {
	return targetType == TargetThread || targetType == TargetThreadComment
}

func isKnownReason(reason string) bool {
	switch reason {
	case ReasonSpam, ReasonSara, ReasonPornografi, ReasonPenipuan, ReasonPerundungan, ReasonLainnya:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// Interface
// ---------------------------------------------------------------------------

// AuthorResolver adalah kebutuhan paket ini terhadap identitas komunitas.
// Sempit dan di sisi pemakai, seperti pola donor fundraising.
type AuthorResolver interface {
	// EnsureProfile mengembalikan profil mentah termasuk IsBlocked dan
	// ProfileVisibility — satu-satunya jalur yang membuka keduanya untuk akun
	// lain. Dipakai di jalur tulis: penegakan blokir dan penentuan kelayakan
	// auto-post.
	EnsureProfile(ctx *gin.Context, userID string) (*community.Profile, error)
	// ProfilesOf mengambil profil beberapa akun sekaligus untuk keperluan
	// tampil, tanpa membuat baris baru. Jalur baca tidak boleh menulis.
	ProfilesOf(ctx *gin.Context, userIDs []string) (map[string]*community.Profile, error)
	// AdminIdentity membuka alias penulis untuk petugas berizin, walaupun
	// profilnya privat atau terblokir. Id akun tetap tidak ikut.
	AdminIdentity(ctx *gin.Context, userID string) (*community.PublicProfile, error)
	// SetBlockedByPublicID mengembalikan identitas publik akun dan apakah
	// statusnya benar-benar berubah. Perubahan itulah yang boleh diaudit;
	// permintaan yang tidak mengubah apa pun tidak menghasilkan baris audit.
	SetBlockedByPublicID(ctx *gin.Context, publicID string, blocked bool) (*community.PublicProfile, bool, error)
	ListBlocked(ctx *gin.Context, page, perPage int) ([]*community.PublicProfile, bool, error)
}

// AutoPoster adalah seam yang dipakai modul sumber aktivitas (order, donasi,
// misi) untuk menitipkan postingan ke feed.
//
// Judul sumber dikirim sebagai argumen supaya paket ini tidak perlu mengimpor
// balik modul-modul itu, dan supaya tidak ada impor melingkar.
//
// Kegagalan tidak pernah menggagalkan aksi sumbernya: pemanggil mencatatnya ke
// log dan menelannya.
type AutoPoster interface {
	PostEventRegistration(ctx *gin.Context, userID, orderID, eventTitle string) error
	// isAnonymous dikirim pemanggil karena hanya modul donasi yang tahu nilai
	// itu. Donasi anonim tidak pernah dibagikan: halaman campaign menyembunyikan
	// identitas donaturnya, dan feed tidak boleh jadi jalan memutarnya.
	PostDonation(ctx *gin.Context, userID, donationID, campaignTitle string, isAnonymous bool) error
	PostMissionCompletion(ctx *gin.Context, userID, claimID, missionTitle string) error

	RemoveEventRegistration(ctx *gin.Context, orderID string) error
	RemoveDonation(ctx *gin.Context, donationID string) error
	RemoveMissionCompletion(ctx *gin.Context, claimID string) error
}

// Repository adalah seluruh akses database paket ini.
type Repository interface {
	// Thread
	CreateThread(ctx *gin.Context, t *Thread) error
	// CreateThreadIfAbsent mengembalikan false bila (source_type, source_ref)
	// sudah pernah dipakai. Itu yang membuat auto-post tepat sekali walau
	// percobaan ulang terjadi.
	CreateThreadIfAbsent(ctx *gin.Context, t *Thread) (bool, error)
	FindThreadByPublicID(ctx *gin.Context, publicID string) (*Thread, error)
	FindThreadByID(ctx *gin.Context, id string) (*Thread, error)
	ListThreads(ctx *gin.Context, sort string, limit, offset int) ([]*Thread, error)
	ListThreadsForReview(ctx *gin.Context, status string, limit, offset int) ([]*Thread, error)
	TransitionThread(ctx *gin.Context, id, from, to string, fields map[string]any) (bool, error)
	SoftDeleteOwnThread(ctx *gin.Context, publicID, userID string) (bool, error)
	// SoftDeleteThreadBySource mencabut satu auto-post lewat rujukan sumbernya.
	SoftDeleteThreadBySource(ctx *gin.Context, sourceType, sourceRef string) (bool, error)
	// SoftDeleteThreadsBySource mencabut seluruh auto-post satu jenis aktivitas
	// milik satu akun, dan mengembalikan berapa baris yang tercabut.
	SoftDeleteThreadsBySource(ctx *gin.Context, userID, sourceType string) (int64, error)
	AdjustThreadCounters(ctx *gin.Context, id string, commentDelta, reactionDelta int) error
	ReactionsByUser(ctx *gin.Context, userID string, threadIDs []string) (map[string]bool, error)

	// Komentar
	CreateComment(ctx *gin.Context, cm *ThreadComment) error
	FindCommentByPublicID(ctx *gin.Context, publicID string) (*ThreadComment, error)
	FindCommentByID(ctx *gin.Context, id string) (*ThreadComment, error)
	ListComments(ctx *gin.Context, threadID string, limit, offset int) ([]*ThreadComment, error)
	TransitionComment(ctx *gin.Context, id, from, to string, fields map[string]any) (bool, error)
	SoftDeleteOwnComment(ctx *gin.Context, publicID, userID string) (bool, error)

	// Reaksi
	AddReaction(ctx *gin.Context, r *ThreadReaction) (bool, error)
	RemoveReaction(ctx *gin.Context, threadID, userID string) (bool, error)

	// Laporan
	CreateReport(ctx *gin.Context, r *Report) (bool, error)
	FindReportByID(ctx *gin.Context, id string) (*Report, error)
	ListReports(ctx *gin.Context, status string, limit, offset int) ([]*Report, error)
	TransitionReport(ctx *gin.Context, id, from, to string, fields map[string]any) (bool, error)

	// Preferensi berbagi
	FindSharePrefs(ctx *gin.Context, userID string) (*SharePrefs, error)
	UpsertSharePrefs(ctx *gin.Context, p *SharePrefs) error

	// Audit
	WriteAudit(ctx *gin.Context, entry *audit.Log) error
	ListAudit(ctx *gin.Context, entityTypes []string, limit, offset int) ([]*audit.Log, error)
}

// Service adalah seluruh aturan paket ini.
type Service interface {
	AutoPoster

	ListThreads(ctx *gin.Context, sort string, page, perPage int) ([]*ThreadView, bool, error)
	ShowThread(ctx *gin.Context, publicID string) (*ThreadView, error)
	CreateThread(ctx *gin.Context, req *CreateThread) (*ThreadView, error)
	DeleteThread(ctx *gin.Context, publicID string) error

	ListComments(ctx *gin.Context, threadPublicID string, page, perPage int) ([]*ThreadCommentView, bool, error)
	CreateComment(ctx *gin.Context, threadPublicID string, req *CreateThreadComment) (*ThreadCommentView, error)
	DeleteComment(ctx *gin.Context, threadPublicID, commentPublicID string) error

	React(ctx *gin.Context, threadPublicID string) error
	Unreact(ctx *gin.Context, threadPublicID string) error
	Report(ctx *gin.Context, req *CreateReport) error

	MySharePrefs(ctx *gin.Context) (*SharePrefs, error)
	UpdateSharePrefs(ctx *gin.Context, req *UpdateSharePrefs) (*SharePrefs, error)

	ListReports(ctx *gin.Context, status string, page, perPage int) ([]*ModerationReportView, bool, error)
	ActionReport(ctx *gin.Context, id string, req *DecideReport) (*Report, error)
	DismissReport(ctx *gin.Context, id string, req *DecideReport) (*Report, error)

	ListThreadsForReview(ctx *gin.Context, status string, page, perPage int) ([]*ThreadView, bool, error)
	ModerateThread(ctx *gin.Context, id, action string, req *DecideContent) (*Thread, error)
	ModerateComment(ctx *gin.Context, id, action string, req *DecideContent) (*ThreadComment, error)

	RestrictedAccounts(ctx *gin.Context, page, perPage int) ([]*community.PublicProfile, bool, error)
	RestrictAccount(ctx *gin.Context, publicID string, req *RestrictAccount) (*community.PublicProfile, error)
	ModerationAudit(ctx *gin.Context, page, perPage int) ([]*audit.Log, bool, error)
}

// Handler adalah lapisan HTTP paket ini.
type Handler interface {
	ListThreads(ctx *gin.Context)
	ShowThread(ctx *gin.Context)
	ListComments(ctx *gin.Context)

	CreateThread(ctx *gin.Context)
	DeleteThread(ctx *gin.Context)
	CreateComment(ctx *gin.Context)
	DeleteComment(ctx *gin.Context)
	React(ctx *gin.Context)
	Unreact(ctx *gin.Context)
	Report(ctx *gin.Context)

	MySharePrefs(ctx *gin.Context)
	UpdateSharePrefs(ctx *gin.Context)

	ListReports(ctx *gin.Context)
	ActionReport(ctx *gin.Context)
	DismissReport(ctx *gin.Context)
	ListThreadsForReview(ctx *gin.Context)
	ModerateThread(ctx *gin.Context)
	ModerateComment(ctx *gin.Context)
	RestrictedAccounts(ctx *gin.Context)
	RestrictAccount(ctx *gin.Context)
	ModerationAudit(ctx *gin.Context)
}
