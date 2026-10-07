package mission

import (
	"time"

	"mainyuk/internal/community"

	"github.com/gin-gonic/gin"
)

// Jenis misi. Nilainya ikut menentukan bagaimana period_key dihitung.
const (
	TypeDaily   = "daily"
	TypeWeekly  = "weekly"
	TypeSpecial = "special"
)

// Cara pembuktian pemenuhan misi.
const (
	// VerificationAuto: pemenuhan dinilai sistem, klaim langsung disetujui.
	VerificationAuto = "auto"
	// VerificationSelfClaim: anggota mengklaim sendiri, tercatat sebagai bukti.
	VerificationSelfClaim = "self_claim"
	// VerificationProofApproval: anggota mengirim bukti, pengurus memutuskan.
	VerificationProofApproval = "proof_approval"
)

// Status klaim.
const (
	StatusPending   = "pending"
	StatusApproved  = "approved"
	StatusRejected  = "rejected"
	StatusCancelled = "cancelled"
)

// Mission adalah definisi misi. Rentang waktu disimpan sebagai instant UTC;
// perhitungan periode di zona Asia/Jakarta dilakukan saat dibaca.
type Mission struct {
	ID               string     `json:"id" gorm:"column:id;primaryKey"`
	Code             string     `json:"code" gorm:"column:code"`
	Title            string     `json:"title" gorm:"column:title"`
	Description      *string    `json:"description" gorm:"column:description"`
	Type             string     `json:"type" gorm:"column:type"`
	VerificationMode string     `json:"verification_mode" gorm:"column:verification_mode"`
	RewardXP         int        `json:"reward_xp" gorm:"column:reward_xp"`
	ClaimLimit       int        `json:"claim_limit" gorm:"column:claim_limit"`
	Version          int        `json:"version" gorm:"column:version"`
	StartsAt         time.Time  `json:"starts_at" gorm:"column:starts_at"`
	EndsAt           time.Time  `json:"ends_at" gorm:"column:ends_at"`
	IsPublished      bool       `json:"is_published" gorm:"column:is_published"`
	SortOrder        int        `json:"sort_order" gorm:"column:sort_order"`
	CreatedBy        *string    `json:"-" gorm:"column:created_by"`
	CreatedAt        time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt        time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt        *time.Time `json:"-" gorm:"column:deleted_at"`
}

func (Mission) TableName() string {
	return "missions"
}

// MissionClaim adalah klaim satu anggota atas satu misi pada satu periode.
type MissionClaim struct {
	ID             string     `json:"id" gorm:"column:id;primaryKey"`
	MissionID      string     `json:"mission_id" gorm:"column:mission_id"`
	MissionVersion int        `json:"mission_version" gorm:"column:mission_version"`
	UserID         string     `json:"-" gorm:"column:user_id"`
	PeriodKey      string     `json:"period_key" gorm:"column:period_key"`
	Status         string     `json:"status" gorm:"column:status"`
	ClaimLimit     int        `json:"-" gorm:"column:claim_limit"`
	RewardXP       *int       `json:"reward_xp" gorm:"column:reward_xp"`
	ProofURL       *string    `json:"proof_url" gorm:"column:proof_url"`
	ProofNote      *string    `json:"proof_note" gorm:"column:proof_note"`
	DecisionReason *string    `json:"decision_reason" gorm:"column:decision_reason"`
	ReviewedBy     *string    `json:"-" gorm:"column:reviewed_by"`
	ReviewedAt     *time.Time `json:"reviewed_at" gorm:"column:reviewed_at"`
	CreatedAt      time.Time  `json:"created_at" gorm:"column:created_at"`
	UpdatedAt      time.Time  `json:"updated_at" gorm:"column:updated_at"`
	DeletedAt      *time.Time `json:"-" gorm:"column:deleted_at"`
}

func (MissionClaim) TableName() string {
	return "mission_claims"
}

// CreateMission adalah payload admin untuk membuat misi.
//
// StartsAt/EndsAt memakai string RFC3339 dan wajib berisi offset zona waktu
// yang eksplisit, supaya "jam 09:00" tidak bergantung pada zona server.
type CreateMission struct {
	Code             string  `json:"code" binding:"required"`
	Title            string  `json:"title" binding:"required"`
	Description      *string `json:"description"`
	Type             string  `json:"type" binding:"required"`
	VerificationMode string  `json:"verification_mode" binding:"required"`
	RewardXP         int     `json:"reward_xp"`
	ClaimLimit       int     `json:"claim_limit"`
	StartsAt         string  `json:"starts_at" binding:"required"`
	EndsAt           string  `json:"ends_at" binding:"required"`
	IsPublished      bool    `json:"is_published"`
	SortOrder        int     `json:"sort_order"`
}

// UpdateMission adalah payload admin untuk mengubah misi. Field yang tidak
// dikirim tidak diubah.
type UpdateMission struct {
	Code             *string `json:"code"`
	Title            *string `json:"title"`
	Description      *string `json:"description"`
	Type             *string `json:"type"`
	VerificationMode *string `json:"verification_mode"`
	RewardXP         *int    `json:"reward_xp"`
	ClaimLimit       *int    `json:"claim_limit"`
	StartsAt         *string `json:"starts_at"`
	EndsAt           *string `json:"ends_at"`
	IsPublished      *bool   `json:"is_published"`
	SortOrder        *int    `json:"sort_order"`
}

// ClaimMission adalah payload anggota untuk mengklaim misi.
type ClaimMission struct {
	ProofURL  *string `json:"proof_url"`
	ProofNote *string `json:"proof_note"`
}

// DecideClaim adalah payload admin untuk memutuskan klaim.
type DecideClaim struct {
	Reason string `json:"reason"`
}

// MissionView adalah bentuk misi yang dikirim ke anggota. Sama dengan model,
// tetapi memuat periode yang berlaku sekarang supaya client tidak perlu
// menghitung zona waktu sendiri.
type MissionView struct {
	*Mission
	CurrentPeriodKey string `json:"current_period_key"`
	// CanClaimNow menandakan jendela klaim sedang terbuka untuk periode ini.
	CanClaimNow bool `json:"can_claim_now"`
	// RequiredProof menandakan klaim wajib menyertakan bukti.
	RequiredProof bool `json:"required_proof"`
}

// ClaimView adalah klaim beserta ringkasan misinya untuk daftar riwayat
// milik anggota sendiri.
type ClaimView struct {
	*MissionClaim
	MissionTitle string `json:"mission_title"`
	MissionType  string `json:"mission_type"`
}

// AdminClaimView adalah klaim pada antrean pemeriksaan pengurus.
//
// Menyertakan identitas publik pengklaim supaya pengurus tahu siapa yang
// diperiksa. Identitasnya tetap hanya yang publik — id akun internal, email,
// telepon, dan alamat tidak ikut.
type AdminClaimView struct {
	*MissionClaim
	MissionTitle string                   `json:"mission_title"`
	MissionType  string                   `json:"mission_type"`
	Claimant     *community.PublicProfile `json:"claimant"`
}

// RewardGranter memberi XP untuk klaim yang sudah disetujui.
// Dipenuhi oleh gamification.Service.
type RewardGranter interface {
	GrantMissionReward(c *gin.Context, userID, claimID string, xp int) (bool, error)
}

// ClaimantResolver mengambil identitas publik pemilik klaim untuk keperluan
// moderasi. Dipenuhi oleh community.Service.
type ClaimantResolver interface {
	AdminIdentity(ctx *gin.Context, userID string) (*community.PublicProfile, error)
}

// AutoPoster adalah seam sempit ke feed komunitas, dipenuhi oleh
// thread.Service.
//
// Judul misi dikirim sebagai argumen supaya paket ini tidak perlu tahu apa pun
// tentang aturan feed — dan supaya feed tidak perlu mengimpor balik paket ini.
// Kegagalannya tidak pernah menggagalkan keputusan klaim yang sudah sah.
type AutoPoster interface {
	PostMissionCompletion(ctx *gin.Context, userID, claimID, missionTitle string) error
	RemoveMissionCompletion(ctx *gin.Context, claimID string) error
}

type Repository interface {
	ListPublished(ctx *gin.Context, now time.Time) ([]*Mission, error)
	FindPublished(ctx *gin.Context, id string, now time.Time) (*Mission, error)
	FindByID(ctx *gin.Context, id string) (*Mission, error)

	ListAll(ctx *gin.Context) ([]*Mission, error)
	Create(ctx *gin.Context, m *Mission) error
	Update(ctx *gin.Context, id string, fields map[string]interface{}) error
	SoftDelete(ctx *gin.Context, id string) error

	// CreateClaimGuarded menyisipkan klaim bila batas klaim untuk periode ini
	// masih tersedia. Mengembalikan false (tanpa error) bila batas tercapai.
	CreateClaimGuarded(ctx *gin.Context, claim *MissionClaim) (bool, error)
	FindClaim(ctx *gin.Context, id string) (*MissionClaim, error)
	// ApprovedClaimIDsByMission mengambil id klaim yang disetujui pada satu
	// misi, dipakai untuk mencabut auto-post ketika misinya dihapus.
	ApprovedClaimIDsByMission(ctx *gin.Context, missionID string) ([]string, error)
	ListClaimsByUser(ctx *gin.Context, userID string, limit, offset int) ([]*MissionClaim, error)
	ListClaimsForReview(ctx *gin.Context, status string, limit, offset int) ([]*MissionClaim, error)
	ApproveClaim(ctx *gin.Context, id, reviewerID, reason string, rewardXP int) (bool, error)
	RejectClaim(ctx *gin.Context, id, reviewerID, reason string) (bool, error)
}

type Service interface {
	ListPublished(ctx *gin.Context) ([]*MissionView, error)
	ShowPublished(ctx *gin.Context, id string) (*MissionView, error)
	Claim(ctx *gin.Context, missionID string, req *ClaimMission) (*MissionClaim, error)
	MyClaims(ctx *gin.Context, page, perPage int) ([]*ClaimView, bool, error)

	ListAll(ctx *gin.Context) ([]*Mission, error)
	Create(ctx *gin.Context, req *CreateMission) (*Mission, error)
	Update(ctx *gin.Context, id string, req *UpdateMission) (*Mission, error)
	Delete(ctx *gin.Context, id string) error
	// SetAutoPoster memasang peniti misi selesai ke feed komunitas. Boleh
	// tidak dipasang: tanpa itu, misi berjalan seperti sebelum Fase 3.
	SetAutoPoster(poster AutoPoster)
	ClaimsForReview(ctx *gin.Context, status string, page, perPage int) ([]*AdminClaimView, bool, error)
	Approve(ctx *gin.Context, claimID string, req *DecideClaim) (*MissionClaim, error)
	Reject(ctx *gin.Context, claimID string, req *DecideClaim) (*MissionClaim, error)
}

type Handler interface {
	ListPublished(ctx *gin.Context)
	ShowPublished(ctx *gin.Context)
	Claim(ctx *gin.Context)
	MyClaims(ctx *gin.Context)

	ListAll(ctx *gin.Context)
	Create(ctx *gin.Context)
	Update(ctx *gin.Context)
	Delete(ctx *gin.Context)
	ClaimsForReview(ctx *gin.Context)
	Approve(ctx *gin.Context)
	Reject(ctx *gin.Context)
}
