package fundraising

import (
	"time"

	"mainyuk/internal/audit"
	"mainyuk/internal/community"
	"mainyuk/internal/payment_method"

	"github.com/gin-gonic/gin"
)

// Jenis dana campaign.
const (
	FundOperasional = "operasional"
	FundDakwah      = "dakwah"
	FundSosial      = "sosial"
	FundPendidikan  = "pendidikan"
	FundLainnya     = "lainnya"
)

// Status campaign.
const (
	CampaignDraft     = "draft"
	CampaignPublished = "published"
	CampaignClosed    = "closed"
)

// Status donasi. Ini status UANG, bukan status pengiriman.
//
// Tidak ada 'expired': Fase 2 tidak punya scheduler, dan donasi yang belum
// ditransfer dibiarkan menunggu keputusan pengurus.
const (
	DonationPending   = "pending"
	DonationConfirmed = "confirmed"
	DonationRejected  = "rejected"
	DonationCancelled = "cancelled"
	DonationRefunded  = "refunded"
)

// Status moderasi pesan donasi.
//
// 'none' berarti donatur tidak menulis pesan. Dibedakan dari 'pending' supaya
// donasi tanpa pesan tidak menumpuk di antrean moderasi.
const (
	MessageNone     = "none"
	MessagePending  = "pending"
	MessageApproved = "approved"
	MessageHidden   = "hidden"
)

// Jenis baris update campaign. 'usage' dan 'fee' wajib membawa nominal supaya
// dapat muncul sebagai baris di laporan; 'update' adalah narasi biasa.
const (
	UpdateKindUpdate = "update"
	UpdateKindUsage  = "usage"
	UpdateKindFee    = "fee"
)

// AnonymousDonorName adalah nama tampilan pengganti untuk donatur yang
// identitasnya tidak boleh tampil.
const AnonymousDonorName = "Hamba Allah"

// Campaign adalah satu program penggalangan dana.
type Campaign struct {
	ID            string     `json:"id" gorm:"column:id;primaryKey"`
	Slug          string     `json:"slug" gorm:"column:slug"`
	Title         string     `json:"title" gorm:"column:title"`
	Summary       *string    `json:"summary" gorm:"column:summary"`
	Story         *string    `json:"story" gorm:"column:story"`
	CoverImageURL *string    `json:"cover_image_url" gorm:"column:cover_image_url"`
	FundType      string     `json:"fund_type" gorm:"column:fund_type"`
	Recipient     string     `json:"recipient" gorm:"column:recipient"`
	TargetAmount  int        `json:"target_amount" gorm:"column:target_amount"`
	Status        string     `json:"status" gorm:"column:status"`
	StartsAt      *time.Time `json:"starts_at" gorm:"column:starts_at"`
	EndsAt        *time.Time `json:"ends_at" gorm:"column:ends_at"`
	PublishedAt   *time.Time `json:"published_at" gorm:"column:published_at"`
	ClosedAt      *time.Time `json:"closed_at" gorm:"column:closed_at"`
	CreatedBy     *string    `json:"-" gorm:"column:created_by"`
	CreatedAt     time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt     time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt     *time.Time `json:"-" gorm:"column:deleted_at"`
}

func (Campaign) TableName() string {
	return "campaigns"
}

// Donation adalah satu donasi. Satu baris = satu donasi = satu pembayaran.
//
// UserID tidak pernah dikirim ke klien: identitas donatur hanya muncul lewat
// bentuk publik yang sudah melewati ResolveDonorName, atau lewat tampilan
// admin.
type Donation struct {
	ID               string     `json:"id" gorm:"column:id;primaryKey"`
	PublicID         string     `json:"public_id" gorm:"column:public_id"`
	CampaignID       string     `json:"campaign_id" gorm:"column:campaign_id"`
	UserID           string     `json:"-" gorm:"column:user_id"`
	Amount           int        `json:"amount" gorm:"column:amount"`
	Status           string     `json:"status" gorm:"column:status"`
	IsAnonymous      bool       `json:"is_anonymous" gorm:"column:is_anonymous"`
	ShowAmount       bool       `json:"show_amount" gorm:"column:show_amount"`
	Message          *string    `json:"message" gorm:"column:message"`
	MessageStatus    string     `json:"message_status" gorm:"column:message_status"`
	PaymentMethodID  *string    `json:"payment_method_id" gorm:"column:payment_method_id"`
	ClientToken      *string    `json:"-" gorm:"column:client_token"`
	PaidAmount       *int       `json:"paid_amount" gorm:"column:paid_amount"`
	PaymentReference *string    `json:"payment_reference" gorm:"column:payment_reference"`
	ProofURL         *string    `json:"proof_url" gorm:"column:proof_url"`
	ConfirmedBy      *string    `json:"-" gorm:"column:confirmed_by"`
	ConfirmedAt      *time.Time `json:"confirmed_at" gorm:"column:confirmed_at"`
	DecisionReason   *string    `json:"decision_reason" gorm:"column:decision_reason"`
	// Snapshot XP yang benar-benar diberikan, dipakai untuk membalik tepat
	// sebesar yang pernah diberikan saat donasi di-refund.
	RewardedXP int        `json:"rewarded_xp" gorm:"column:rewarded_xp"`
	CreatedAt  time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt  time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt  *time.Time `json:"-" gorm:"column:deleted_at"`
}

func (Donation) TableName() string {
	return "donations"
}

// CampaignUpdate adalah narasi penggunaan dana, catatan biaya, atau kabar
// perkembangan campaign.
type CampaignUpdate struct {
	ID          string     `json:"id" gorm:"column:id;primaryKey"`
	CampaignID  string     `json:"campaign_id" gorm:"column:campaign_id"`
	Title       string     `json:"title" gorm:"column:title"`
	Body        string     `json:"body" gorm:"column:body"`
	Kind        string     `json:"kind" gorm:"column:kind"`
	Amount      *int       `json:"amount" gorm:"column:amount"`
	ProofURL    *string    `json:"proof_url" gorm:"column:proof_url"`
	IsPublished bool       `json:"is_published" gorm:"column:is_published"`
	CreatedBy   *string    `json:"-" gorm:"column:created_by"`
	CreatedAt   time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt   *time.Time `json:"-" gorm:"column:deleted_at"`
}

func (CampaignUpdate) TableName() string {
	return "campaign_updates"
}

// AuditLog adalah satu baris jejak audit. Append-only: tidak ada deleted_at
// dan tidak ada jalur update, sama seperti xp_ledger.
//
// Tabelnya lintas modul, jadi model dan penulisnya tinggal di internal/audit.
// Alias ini dipertahankan supaya seluruh call site dan bentuk JSON endpoint
// audit milik fundraising tidak berubah saat penulisnya dipindahkan.
type AuditLog = audit.Log

// ---------------------------------------------------------------------------
// DTO
// ---------------------------------------------------------------------------

// CreateCampaign adalah payload admin untuk membuat campaign.
//
// StartsAt/EndsAt memakai string RFC3339 dengan offset zona waktu eksplisit,
// mengikuti konvensi misi: tanpa offset, "jam 09:00" akan ditafsirkan dalam
// zona server dan jadwal bergeser tanpa disadari.
type CreateCampaign struct {
	Slug          string  `json:"slug" binding:"required"`
	Title         string  `json:"title" binding:"required"`
	Summary       *string `json:"summary"`
	Story         *string `json:"story"`
	CoverImageURL *string `json:"cover_image_url"`
	FundType      string  `json:"fund_type" binding:"required"`
	Recipient     string  `json:"recipient" binding:"required"`
	TargetAmount  int     `json:"target_amount"`
	StartsAt      *string `json:"starts_at"`
	EndsAt        *string `json:"ends_at"`
}

// UpdateCampaign adalah payload admin untuk mengubah campaign. Field yang
// tidak dikirim tidak diubah.
type UpdateCampaign struct {
	Slug          *string `json:"slug"`
	Title         *string `json:"title"`
	Summary       *string `json:"summary"`
	Story         *string `json:"story"`
	CoverImageURL *string `json:"cover_image_url"`
	FundType      *string `json:"fund_type"`
	Recipient     *string `json:"recipient"`
	TargetAmount  *int    `json:"target_amount"`
	StartsAt      *string `json:"starts_at"`
	EndsAt        *string `json:"ends_at"`
}

// SetCampaignStatus adalah payload admin untuk memindahkan status campaign.
type SetCampaignStatus struct {
	Status string `json:"status" binding:"required"`
	Reason string `json:"reason"`
}

// CreateDonation adalah payload anggota untuk membuat donasi.
//
// ClientToken wajib: tanpanya, perlindungan terhadap klik ganda hilang dan
// donasi bisa tercipta dua kali. Klien membuatnya sekali per pemuatan form.
type CreateDonation struct {
	CampaignSlug    string  `json:"campaign_slug" binding:"required"`
	Amount          int     `json:"amount" binding:"required"`
	IsAnonymous     bool    `json:"is_anonymous"`
	ShowAmount      *bool   `json:"show_amount"`
	Message         *string `json:"message"`
	PaymentMethodID *string `json:"payment_method_id"`
	ClientToken     string  `json:"client_token" binding:"required"`
}

// ConfirmDonation adalah payload admin saat memverifikasi transfer.
type ConfirmDonation struct {
	PaidAmount       int     `json:"paid_amount"`
	PaymentReference string  `json:"payment_reference"`
	ProofURL         *string `json:"proof_url"`
	Reason           *string `json:"reason"`
}

// DecideDonation adalah payload admin untuk menolak atau me-refund donasi.
// Alasan wajib diisi di lapisan service.
type DecideDonation struct {
	Reason string `json:"reason"`
}

// ModerateMessage adalah payload admin untuk memutuskan pesan donasi.
type ModerateMessage struct {
	Decision string `json:"decision" binding:"required"`
	Reason   string `json:"reason"`
}

// CampaignUpdateInput adalah payload admin untuk menulis update campaign.
type CampaignUpdateInput struct {
	Title       string  `json:"title" binding:"required"`
	Body        string  `json:"body" binding:"required"`
	Kind        string  `json:"kind"`
	Amount      *int    `json:"amount"`
	ProofURL    *string `json:"proof_url"`
	IsPublished *bool   `json:"is_published"`
}

// DonationFilter menyaring daftar donasi pada antrean admin.
type DonationFilter struct {
	Status        string
	MessageStatus string
	CampaignID    string
}

// ---------------------------------------------------------------------------
// Bentuk yang dikirim ke klien
// ---------------------------------------------------------------------------

// CampaignView adalah campaign beserta ringkasan progresnya.
type CampaignView struct {
	*Campaign
	RaisedAmount int `json:"raised_amount"`
	DonorCount   int `json:"donor_count"`
	// ProgressPercent dibulatkan ke bawah dan dipagari 100, supaya bilah
	// progres tidak pernah melampaui lebarnya.
	ProgressPercent int  `json:"progress_percent"`
	IsOpen          bool `json:"is_open"`
}

// PublicDonation adalah bentuk donasi di permukaan publik.
//
// Amount bernilai nil berarti nominalnya disembunyikan. Menyembunyikan nominal
// karena itu adalah tipe, bukan konvensi yang bisa terlupa di satu endpoint.
type PublicDonation struct {
	PublicID    string    `json:"public_id"`
	DonorName   string    `json:"donor_name"`
	DonorHidden bool      `json:"donor_hidden"`
	AvatarURL   *string   `json:"avatar_url"`
	Amount      *int      `json:"amount"`
	Message     *string   `json:"message"`
	CreatedAt   time.Time `json:"created_at"`
}

// DonationView adalah donasi milik pemanggil sendiri. Selalu lengkap: pemilik
// berhak melihat donasinya apa adanya.
type DonationView struct {
	*Donation
	CampaignSlug  string `json:"campaign_slug"`
	CampaignTitle string `json:"campaign_title"`
}

// AdminDonationView adalah donasi pada antrean verifikasi pengurus.
//
// Menyertakan identitas publik donatur supaya pengurus tahu siapa yang
// diperiksa. Yang dibuka tetap hanya identitas publik — id akun internal,
// email, dan telepon tidak ikut.
type AdminDonationView struct {
	*Donation
	Donor         *community.PublicProfile `json:"donor"`
	CampaignSlug  string                   `json:"campaign_slug"`
	CampaignTitle string                   `json:"campaign_title"`
}

// DonationResult adalah hasil pembuatan atau pembacaan satu donasi, lengkap
// dengan instruksi pembayaran yang harus ditampilkan.
type DonationResult struct {
	Donation *DonationView `json:"donation"`
	Charge   *Charge       `json:"charge"`
}

// ReportTotals adalah rangkuman angka satu campaign.
//
// Progres (ConfirmedAmount) selalu bruto: biaya dan penggunaan dana dicatat
// sebagai baris terpisah dan tidak mengurangi dana terkumpul. NetAmount
// sengaja tidak dipagari nol — saldo negatif adalah sinyal nyata bahwa dana
// terpakai melebihi yang terkumpul, dan memagarinya justru menyembunyikannya.
type ReportTotals struct {
	ConfirmedAmount int `json:"confirmed_amount"`
	ConfirmedCount  int `json:"confirmed_count"`
	PendingAmount   int `json:"pending_amount"`
	PendingCount    int `json:"pending_count"`
	RejectedAmount  int `json:"rejected_amount"`
	RejectedCount   int `json:"rejected_count"`
	RefundedAmount  int `json:"refunded_amount"`
	RefundedCount   int `json:"refunded_count"`
	FeeAmount       int `json:"fee_amount"`
	UsageAmount     int `json:"usage_amount"`
	NetAmount       int `json:"net_amount"`
}

// ReportInput adalah agregat mentah yang dirangkum ComputeReport.
type ReportInput struct {
	ConfirmedAmount int
	ConfirmedCount  int
	PendingAmount   int
	PendingCount    int
	RejectedAmount  int
	RejectedCount   int
	RefundedAmount  int
	RefundedCount   int
	FeeAmount       int
	UsageAmount     int
}

// ReportView adalah laporan satu campaign.
type ReportView struct {
	Campaign *Campaign         `json:"campaign"`
	Totals   ReportTotals      `json:"totals"`
	Updates  []*CampaignUpdate `json:"updates"`
}

// CampaignTotals adalah agregat progres satu campaign.
type CampaignTotals struct {
	RaisedAmount int
	DonorCount   int
}

// UpdateTotals adalah agregat baris penggunaan dana dan biaya.
type UpdateTotals struct {
	UsageAmount int
	FeeAmount   int
}

// ---------------------------------------------------------------------------
// Ketergantungan ke modul lain (interface sempit di sisi pemakai)
// ---------------------------------------------------------------------------

// DonorResolver mengambil profil donatur, baik untuk permukaan publik maupun
// untuk antrean pengurus.
//
// EnsureProfile dipakai sebagai PEMBACA, bukan sekadar penjamin: ia satu-satunya
// jalur yang mengembalikan Profile mentah — termasuk IsBlocked dan
// ProfileVisibility — untuk akun orang lain. Aturan anonimitas membutuhkan
// kedua field itu, dan ShowPublic tidak dapat dipakai karena ia menjawab 404
// untuk profil privat sehingga donaturnya akan hilang dari daftar, bukan
// tampil sebagai "Hamba Allah".
//
// AdminIdentity melengkapi antrean pengurus. Berbeda dari ShowPublic,
// visibilitas privat dan status blokir tidak disembunyikan di sana: pengurus
// harus tetap dapat mengenali pemilik donasi yang sedang diverifikasi. Yang
// tetap tidak dibuka hanya data akun internal (id akun, email, telepon).
type DonorResolver interface {
	EnsureProfile(ctx *gin.Context, userID string) (*community.Profile, error)
	AdminIdentity(ctx *gin.Context, userID string) (*community.PublicProfile, error)
}

// RewardGranter menghubungkan donasi terkonfirmasi dengan ledger XP.
type RewardGranter interface {
	GrantDonationReward(ctx *gin.Context, userID, donationID string) (int, error)
	ReverseDonationReward(ctx *gin.Context, userID, donationID string, xp int, reason string) error
}

// MethodResolver mengambil metode pembayaran untuk menyusun instruksi transfer.
type MethodResolver interface {
	Show(ctx *gin.Context, id string) (*payment_method.PaymentMethod, error)
}

// AutoPoster adalah seam sempit ke feed komunitas, dipenuhi oleh thread.Service.
//
// isAnonymous ikut dikirim karena hanya modul ini yang tahu nilainya, dan hanya
// modul ini yang tahu bahwa donasi anonim menyembunyikan identitas donaturnya
// di halaman campaign. Judul campaign dikirim sebagai argumen supaya feed tidak
// perlu mengimpor balik paket ini.
//
// Kegagalan auto-post tidak pernah menggagalkan donasi yang sudah sah.
type AutoPoster interface {
	PostDonation(ctx *gin.Context, userID, donationID, campaignTitle string, isAnonymous bool) error
	RemoveDonation(ctx *gin.Context, donationID string) error
}

// ---------------------------------------------------------------------------
// Kontrak lapisan
// ---------------------------------------------------------------------------

type Repository interface {
	// Campaign
	ListPublished(ctx *gin.Context, now time.Time) ([]*Campaign, error)
	FindPublishedBySlug(ctx *gin.Context, slug string, now time.Time) (*Campaign, error)
	FindByID(ctx *gin.Context, id string) (*Campaign, error)
	ListAll(ctx *gin.Context, status string) ([]*Campaign, error)
	CreateCampaign(ctx *gin.Context, campaign *Campaign) error
	UpdateCampaign(ctx *gin.Context, id string, fields map[string]interface{}) error
	TransitionCampaign(ctx *gin.Context, id, from, to string, fields map[string]interface{}) (bool, error)
	SoftDeleteCampaign(ctx *gin.Context, id string) error

	// Agregat
	CampaignTotals(ctx *gin.Context, campaignID string) (*CampaignTotals, error)
	CampaignTotalsBatch(ctx *gin.Context, campaignIDs []string) (map[string]*CampaignTotals, error)
	UpdateTotals(ctx *gin.Context, campaignID string) (*UpdateTotals, error)
	DonationAggregates(ctx *gin.Context, campaignID string) (*ReportInput, error)

	// Donasi
	CreateDonationGuarded(ctx *gin.Context, donation *Donation) (*Donation, bool, error)
	FindDonationByPublicID(ctx *gin.Context, publicID string) (*Donation, error)
	FindDonationByID(ctx *gin.Context, id string) (*Donation, error)
	ListDonationsByUser(ctx *gin.Context, userID string, limit, offset int) ([]*Donation, error)
	ListDonationsForReview(ctx *gin.Context, filter DonationFilter, limit, offset int) ([]*Donation, error)
	TransitionDonation(ctx *gin.Context, id, from, to string, fields map[string]interface{}) (bool, error)
	SetDonationReward(ctx *gin.Context, id string, xp int) error
	ListDonors(ctx *gin.Context, campaignID string, limit, offset int) ([]*Donation, error)
	ListApprovedMessages(ctx *gin.Context, campaignID string, limit, offset int) ([]*Donation, error)
	TransitionMessage(ctx *gin.Context, id, from, to string, fields map[string]interface{}) (bool, error)

	// Update campaign
	ListUpdates(ctx *gin.Context, campaignID string, publishedOnly bool) ([]*CampaignUpdate, error)
	FindUpdate(ctx *gin.Context, id string) (*CampaignUpdate, error)
	CreateUpdate(ctx *gin.Context, update *CampaignUpdate) error
	UpdateUpdate(ctx *gin.Context, id string, fields map[string]interface{}) error
	SoftDeleteUpdate(ctx *gin.Context, id string) error

	// Audit
	WriteAudit(ctx *gin.Context, entry *AuditLog) error
	ListAudit(ctx *gin.Context, entityType, entityID string, limit, offset int) ([]*AuditLog, error)
}

type Service interface {
	// Permukaan publik
	ListCampaigns(ctx *gin.Context) ([]*CampaignView, error)
	ShowCampaign(ctx *gin.Context, slug string) (*CampaignView, error)
	ListDonors(ctx *gin.Context, slug string, page, perPage int) ([]*PublicDonation, bool, error)
	ListMessages(ctx *gin.Context, slug string, page, perPage int) ([]*PublicDonation, bool, error)
	ListUpdates(ctx *gin.Context, slug string) ([]*CampaignUpdate, error)
	PublicReport(ctx *gin.Context, slug string) (*ReportView, error)

	// Anggota
	CreateDonation(ctx *gin.Context, req *CreateDonation) (*DonationResult, error)
	MyDonations(ctx *gin.Context, page, perPage int) ([]*DonationView, bool, error)
	MyDonation(ctx *gin.Context, publicID string) (*DonationResult, error)

	// Pengurus
	ListAllCampaigns(ctx *gin.Context, status string) ([]*CampaignView, error)
	CreateCampaign(ctx *gin.Context, req *CreateCampaign) (*Campaign, error)
	UpdateCampaign(ctx *gin.Context, id string, req *UpdateCampaign) (*Campaign, error)
	SetCampaignStatus(ctx *gin.Context, id string, req *SetCampaignStatus) (*Campaign, error)
	DeleteCampaign(ctx *gin.Context, id string) error
	CampaignReport(ctx *gin.Context, id string) (*ReportView, error)
	DonationsForReview(ctx *gin.Context, filter DonationFilter, page, perPage int) ([]*AdminDonationView, bool, error)
	ConfirmDonation(ctx *gin.Context, id string, req *ConfirmDonation) (*Donation, error)
	RejectDonation(ctx *gin.Context, id string, req *DecideDonation) (*Donation, error)
	RefundDonation(ctx *gin.Context, id string, req *DecideDonation) (*Donation, error)
	ModerateMessage(ctx *gin.Context, id string, req *ModerateMessage) (*Donation, error)
	// SetAutoPoster memasang peniti donasi terkonfirmasi ke feed komunitas.
	// Boleh tidak dipasang: tanpa itu, donasi berjalan seperti sebelum Fase 3.
	SetAutoPoster(poster AutoPoster)
	CreateUpdate(ctx *gin.Context, campaignID string, req *CampaignUpdateInput) (*CampaignUpdate, error)
	UpdateUpdate(ctx *gin.Context, id string, req *CampaignUpdateInput) (*CampaignUpdate, error)
	DeleteUpdate(ctx *gin.Context, id string) error
	ListAuditLogs(ctx *gin.Context, entityType, entityID string, page, perPage int) ([]*AuditLog, bool, error)
}

type Handler interface {
	ListCampaigns(ctx *gin.Context)
	ShowCampaign(ctx *gin.Context)
	ListDonors(ctx *gin.Context)
	ListMessages(ctx *gin.Context)
	ListUpdates(ctx *gin.Context)
	PublicReport(ctx *gin.Context)

	CreateDonation(ctx *gin.Context)
	MyDonations(ctx *gin.Context)
	MyDonation(ctx *gin.Context)

	ListAllCampaigns(ctx *gin.Context)
	CreateCampaign(ctx *gin.Context)
	UpdateCampaign(ctx *gin.Context)
	SetCampaignStatus(ctx *gin.Context)
	DeleteCampaign(ctx *gin.Context)
	CampaignReport(ctx *gin.Context)
	DonationsForReview(ctx *gin.Context)
	ConfirmDonation(ctx *gin.Context)
	RejectDonation(ctx *gin.Context)
	RefundDonation(ctx *gin.Context)
	ModerateMessage(ctx *gin.Context)
	CreateUpdate(ctx *gin.Context)
	UpdateUpdate(ctx *gin.Context)
	DeleteUpdate(ctx *gin.Context)
	ListAuditLogs(ctx *gin.Context)
}
