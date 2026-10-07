package mission

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// isUniqueViolation melaporkan apakah err berasal dari pelanggaran constraint
// unik PostgreSQL (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

func (r *repository) ListPublished(c *gin.Context, now time.Time) ([]*Mission, error) {
	missions := []*Mission{}
	err := r.db.
		Where("deleted_at IS NULL").
		Where("is_published = true").
		Where("ends_at > ?", now).
		Order("sort_order ASC, starts_at ASC").
		Find(&missions).Error
	if err != nil {
		return nil, err
	}
	return missions, nil
}

func (r *repository) FindPublished(c *gin.Context, id string, now time.Time) (*Mission, error) {
	mission := &Mission{}
	err := r.db.
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Where("is_published = true").
		Where("ends_at > ?", now).
		First(mission).Error
	if err != nil {
		return nil, err
	}
	return mission, nil
}

func (r *repository) FindByID(c *gin.Context, id string) (*Mission, error) {
	mission := &Mission{}
	err := r.db.Where("id = ?", id).Where("deleted_at IS NULL").First(mission).Error
	if err != nil {
		return nil, err
	}
	return mission, nil
}

func (r *repository) ListAll(c *gin.Context) ([]*Mission, error) {
	missions := []*Mission{}
	err := r.db.
		Where("deleted_at IS NULL").
		Order("sort_order ASC, starts_at DESC").
		Find(&missions).Error
	if err != nil {
		return nil, err
	}
	return missions, nil
}

func (r *repository) Create(c *gin.Context, m *Mission) error {
	return r.db.Create(m).Error
}

func (r *repository) Update(c *gin.Context, id string, fields map[string]interface{}) error {
	fields["updated_at"] = time.Now()
	return r.db.Model(&Mission{}).Where("id = ?", id).Updates(fields).Error
}

func (r *repository) SoftDelete(c *gin.Context, id string) error {
	return r.db.Model(&Mission{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": time.Now(),
			"updated_at": time.Now(),
		}).Error
}

// errLimitReached menandai batas klaim tercapai di dalam transaksi.
//
// Dipakai sebagai nilai balik dari fungsi transaksi supaya transaksinya
// di-rollback secara bersih: mengembalikan nil setelah pelanggaran unik akan
// mencoba commit transaksi yang sudah dibatalkan PostgreSQL.
var errLimitReached = errors.New("batas klaim tercapai")

// CreateClaimGuarded menyisipkan klaim bila batas klaim masih tersedia.
//
// Dua lapis penjagaan, keduanya di dalam satu transaksi:
//
//  1. Kunci advisory transaksional per (misi, akun) menyerialkan hitung-lalu-
//     sisip, sehingga misi dengan claim_limit > 1 — yang tidak tercakup index
//     unik parsial — tetap aman dari balapan.
//  2. Index unik parsial uniq_mission_claims_aktif adalah jaminan keras untuk
//     claim_limit = 1, berlaku walaupun kedua permintaan datang dari proses
//     yang berbeda.
func (r *repository) CreateClaimGuarded(c *gin.Context, claim *MissionClaim) (bool, error) {
	created := false

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			"SELECT pg_advisory_xact_lock(hashtext(?))",
			claim.MissionID+":"+claim.UserID,
		).Error; err != nil {
			return err
		}

		count := int64(0)
		if err := tx.Model(&MissionClaim{}).
			Where("mission_id = ?", claim.MissionID).
			Where("user_id = ?", claim.UserID).
			Where("period_key = ?", claim.PeriodKey).
			Where("status IN ?", []string{StatusPending, StatusApproved}).
			Where("deleted_at IS NULL").
			Count(&count).Error; err != nil {
			return err
		}
		if !CanClaim(claim.ClaimLimit, int(count)) {
			return nil
		}

		if err := tx.Create(claim).Error; err != nil {
			if isUniqueViolation(err) {
				return errLimitReached
			}
			return err
		}
		created = true
		return nil
	})

	if err != nil {
		if errors.Is(err, errLimitReached) {
			return false, nil
		}
		return false, err
	}
	return created, nil
}

func (r *repository) FindClaim(c *gin.Context, id string) (*MissionClaim, error) {
	claim := &MissionClaim{}
	err := r.db.Where("id = ?", id).Where("deleted_at IS NULL").First(claim).Error
	if err != nil {
		return nil, err
	}
	return claim, nil
}

// ApprovedClaimIDsByMission mengambil id klaim yang disetujui pada satu misi.
//
// Hanya klaim yang disetujui yang pernah menghasilkan auto-post, jadi hanya
// itu yang perlu dicabut saat misinya dihapus.
func (r *repository) ApprovedClaimIDsByMission(c *gin.Context, missionID string) ([]string, error) {
	ids := []string{}
	err := r.db.Model(&MissionClaim{}).
		Where("mission_id = ?", missionID).
		Where("status = ?", StatusApproved).
		Where("deleted_at IS NULL").
		Pluck("id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}

func (r *repository) ListClaimsByUser(c *gin.Context, userID string, limit, offset int) ([]*MissionClaim, error) {
	claims := []*MissionClaim{}
	err := r.db.
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&claims).Error
	if err != nil {
		return nil, err
	}
	return claims, nil
}

func (r *repository) ListClaimsForReview(c *gin.Context, status string, limit, offset int) ([]*MissionClaim, error) {
	claims := []*MissionClaim{}

	query := r.db.Where("deleted_at IS NULL")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.
		Order("created_at ASC, id ASC").
		Limit(limit).
		Offset(offset).
		Find(&claims).Error
	if err != nil {
		return nil, err
	}
	return claims, nil
}

// ApproveClaim menyetujui klaim secara atomik.
//
// Guard status='pending' adalah inti idempotensinya: dua pengurus yang
// menyetujui bersamaan akan diserialkan oleh lock baris, dan yang kedua
// mengevaluasi ulang terhadap baris yang sudah ter-commit sehingga tidak
// menemukan baris pending. RowsAffected = 0 karena itu berarti "sudah
// diputuskan", bukan kegagalan.
//
// reward_xp di-snapshot di sini dari nilai yang berlaku saat keputusan dibuat,
// sehingga mengubah definisi misi tidak mengubah reward klaim yang berjalan.
func (r *repository) ApproveClaim(c *gin.Context, id, reviewerID, reason string, rewardXP int) (bool, error) {
	now := time.Now()
	fields := map[string]interface{}{
		"status":      StatusApproved,
		"reward_xp":   rewardXP,
		"reviewed_by": reviewerID,
		"reviewed_at": now,
		"updated_at":  now,
	}
	if reason != "" {
		fields["decision_reason"] = reason
	}

	res := r.db.Model(&MissionClaim{}).
		Where("id = ?", id).
		Where("status = ?", StatusPending).
		Where("deleted_at IS NULL").
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// RejectClaim menolak klaim. Tidak ada XP yang diberikan pada jalur ini.
func (r *repository) RejectClaim(c *gin.Context, id, reviewerID, reason string) (bool, error) {
	now := time.Now()
	fields := map[string]interface{}{
		"status":      StatusRejected,
		"reviewed_by": reviewerID,
		"reviewed_at": now,
		"updated_at":  now,
	}
	if reason != "" {
		fields["decision_reason"] = reason
	}

	res := r.db.Model(&MissionClaim{}).
		Where("id = ?", id).
		Where("status = ?", StatusPending).
		Where("deleted_at IS NULL").
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}
