package mission

import (
	"errors"
	"log"
	"strings"
	"time"

	"mainyuk/internal/apperr"
	"mainyuk/internal/gamification"
	"mainyuk/internal/ratelimit"
	"mainyuk/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type service struct {
	Repository
	RewardGranter    RewardGranter
	ClaimantResolver ClaimantResolver
	// poster boleh nil; nil berarti misi selesai tidak dibagikan ke feed,
	// seperti sebelum Fase 3.
	poster AutoPoster
}

// SetAutoPoster memasang peniti misi selesai ke feed komunitas.
func (s *service) SetAutoPoster(poster AutoPoster) {
	s.poster = poster
}

// postCompletion menitipkan klaim yang disetujui ke feed.
//
// Kegagalannya ditelan dan dilaporkan ke log: klaim yang sudah sah tidak boleh
// dibatalkan hanya karena feed gagal. Dipanggil atas dasar status akhirnya,
// bukan atas dasar "barisnya baru berpindah" — index unik sumber membuat
// percobaan ulang aman dan justru memulihkan postingan yang tertinggal.
func (s *service) postCompletion(c *gin.Context, userID, claimID, missionID string) {
	if s.poster == nil || userID == "" {
		return
	}
	if err := s.poster.PostMissionCompletion(c, userID, claimID, s.missionTitle(c, missionID)); err != nil {
		log.Printf("[mission] gagal membagikan klaim %s ke feed: %v", claimID, err)
	}
}

// removeCompletion mencabut postingan satu klaim. Kegagalannya juga ditelan.
func (s *service) removeCompletion(c *gin.Context, claimID string) {
	if s.poster == nil || claimID == "" {
		return
	}
	if err := s.poster.RemoveMissionCompletion(c, claimID); err != nil {
		log.Printf("[mission] gagal mencabut klaim %s dari feed: %v", claimID, err)
	}
}

// missionTitle mengambil judul misi untuk judul postingan. Kegagalan
// pengambilannya tidak menggagalkan keputusan klaim; postingannya saja yang
// tidak dibuat, karena judul kosong ditolak ShouldAutoPost.
func (s *service) missionTitle(c *gin.Context, missionID string) string {
	mission, err := s.Repository.FindByID(c, missionID)
	if err != nil || mission == nil {
		return ""
	}
	return mission.Title
}

func NewService(repository Repository, rewardGranter RewardGranter, claimantResolver ClaimantResolver) Service {
	return &service{
		Repository:       repository,
		RewardGranter:    rewardGranter,
		ClaimantResolver: claimantResolver,
	}
}

func (s *service) ListPublished(c *gin.Context) ([]*MissionView, error) {
	missions, err := s.Repository.ListPublished(c, time.Now())
	if err != nil {
		return nil, err
	}

	views := make([]*MissionView, 0, len(missions))
	for _, m := range missions {
		views = append(views, s.toView(m, time.Now()))
	}
	return views, nil
}

func (s *service) ShowPublished(c *gin.Context, id string) (*MissionView, error) {
	mission, err := s.Repository.FindPublished(c, id, time.Now())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMissionNotFound
		}
		return nil, err
	}
	return s.toView(mission, time.Now()), nil
}

// toView melengkapi misi dengan periode yang berlaku sekarang.
func (s *service) toView(m *Mission, now time.Time) *MissionView {
	return &MissionView{
		Mission:          m,
		CurrentPeriodKey: PeriodKey(m.Type, now),
		CanClaimNow:      ClaimWindowOpen(now, m.StartsAt, m.EndsAt, m.IsPublished),
		RequiredProof:    RequiredProof(m.VerificationMode),
	}
}

// Claim membuat klaim anggota atas sebuah misi.
//
// Urutan pemeriksaan sengaja dari yang paling murah dan paling menentukan:
// jendela klaim, kelengkapan bukti, lalu batas klaim yang butuh transaksi.
func (s *service) Claim(c *gin.Context, missionID string, req *ClaimMission) (*MissionClaim, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}

	if !ratelimit.MissionClaim.Allow(ratelimit.Key(c, currentUser.ID)) {
		return nil, ratelimit.ErrTooManyRequests
	}

	mission, err := s.Repository.FindByID(c, missionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMissionNotFound
		}
		return nil, err
	}

	now := time.Now()
	if !ClaimWindowOpen(now, mission.StartsAt, mission.EndsAt, mission.IsPublished) {
		return nil, ErrClaimWindowClosed
	}

	var proofURL, proofNote *string
	if req != nil {
		proofURL = trimPtr(req.ProofURL)
		proofNote = trimPtr(req.ProofNote)
	}
	if RequiredProof(mission.VerificationMode) && proofURL == nil && proofNote == nil {
		return nil, invalid("misi ini mewajibkan bukti berupa tautan atau catatan")
	}

	claim := &MissionClaim{
		ID:             uuid.NewString(),
		MissionID:      mission.ID,
		MissionVersion: mission.Version,
		UserID:         currentUser.ID,
		PeriodKey:      PeriodKey(mission.Type, now),
		Status:         StatusPending,
		ClaimLimit:     mission.ClaimLimit,
		ProofURL:       proofURL,
		ProofNote:      proofNote,
	}

	created, err := s.Repository.CreateClaimGuarded(c, claim)
	if err != nil {
		return nil, err
	}
	if !created {
		return nil, ErrClaimLimitReached
	}

	// Misi otomatis langsung diputuskan lewat jalur approval yang sama, supaya
	// hanya ada satu tempat yang menulis status dan memberi XP.
	if IsAutoApproved(mission.VerificationMode) {
		return s.decide(c, claim.ID, currentUser.ID, StatusApproved, "")
	}

	return s.Repository.FindClaim(c, claim.ID)
}

func (s *service) MyClaims(c *gin.Context, page, perPage int) ([]*ClaimView, bool, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, false, apperr.ErrUnauthorized
	}
	page, perPage = gamification.NormalizePagination(page, perPage)

	claims, err := s.Repository.ListClaimsByUser(c, currentUser.ID, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(claims) > perPage
	if hasMore {
		claims = claims[:perPage]
	}
	views, err := s.toClaimViews(c, claims)
	if err != nil {
		return nil, false, err
	}
	return views, hasMore, nil
}

func (s *service) ListAll(c *gin.Context) ([]*Mission, error) {
	return s.Repository.ListAll(c)
}

func (s *service) Create(c *gin.Context, req *CreateMission) (*Mission, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	startsAt, endsAt, err := parseWindow(req.StartsAt, req.EndsAt)
	if err != nil {
		return nil, err
	}
	if err := ValidateMissionType(req.Type); err != nil {
		return nil, err
	}
	if err := ValidateVerificationMode(req.VerificationMode); err != nil {
		return nil, err
	}

	code := strings.TrimSpace(req.Code)
	if code == "" {
		return nil, invalid("kode misi wajib diisi")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, invalid("judul misi wajib diisi")
	}
	if req.RewardXP < 0 {
		return nil, invalid("reward XP tidak boleh negatif")
	}
	if req.ClaimLimit < 1 {
		return nil, invalid("batas klaim minimal 1")
	}

	mission := &Mission{
		ID:               uuid.NewString(),
		Code:             code,
		Title:            title,
		Description:      trimPtr(req.Description),
		Type:             req.Type,
		VerificationMode: req.VerificationMode,
		RewardXP:         req.RewardXP,
		ClaimLimit:       req.ClaimLimit,
		Version:          1,
		StartsAt:         startsAt,
		EndsAt:           endsAt,
		IsPublished:      req.IsPublished,
		SortOrder:        req.SortOrder,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
	if currentUser, ok := user.FromContext(c); ok {
		mission.CreatedBy = &currentUser.ID
	}

	if err := s.Repository.Create(c, mission); err != nil {
		if isUniqueViolation(err) {
			return nil, invalid("kode misi sudah dipakai misi lain yang aktif")
		}
		return nil, err
	}
	return mission, nil
}

func (s *service) Update(c *gin.Context, id string, req *UpdateMission) (*Mission, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	existing, err := s.Repository.FindByID(c, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrMissionNotFound
		}
		return nil, err
	}

	fields := map[string]interface{}{}

	if req.Code != nil {
		code := strings.TrimSpace(*req.Code)
		if code == "" {
			return nil, invalid("kode misi wajib diisi")
		}
		fields["code"] = code
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, invalid("judul misi wajib diisi")
		}
		fields["title"] = title
	}
	if req.Description != nil {
		fields["description"] = trimPtr(req.Description)
	}
	if req.Type != nil {
		if err := ValidateMissionType(*req.Type); err != nil {
			return nil, err
		}
		fields["type"] = *req.Type
	}
	if req.VerificationMode != nil {
		if err := ValidateVerificationMode(*req.VerificationMode); err != nil {
			return nil, err
		}
		fields["verification_mode"] = *req.VerificationMode
	}
	if req.RewardXP != nil {
		if *req.RewardXP < 0 {
			return nil, invalid("reward XP tidak boleh negatif")
		}
		fields["reward_xp"] = *req.RewardXP
	}
	if req.ClaimLimit != nil {
		if *req.ClaimLimit < 1 {
			return nil, invalid("batas klaim minimal 1")
		}
		fields["claim_limit"] = *req.ClaimLimit
	}
	if req.IsPublished != nil {
		fields["is_published"] = *req.IsPublished
	}
	if req.SortOrder != nil {
		fields["sort_order"] = *req.SortOrder
	}

	if req.StartsAt != nil || req.EndsAt != nil {
		startsAt := existing.StartsAt
		endsAt := existing.EndsAt
		if req.StartsAt != nil {
			parsed, err := parseTime(*req.StartsAt)
			if err != nil {
				return nil, err
			}
			startsAt = parsed
		}
		if req.EndsAt != nil {
			parsed, err := parseTime(*req.EndsAt)
			if err != nil {
				return nil, err
			}
			endsAt = parsed
		}
		if !endsAt.After(startsAt) {
			return nil, invalid("waktu berakhir harus setelah waktu mulai")
		}
		fields["starts_at"] = startsAt
		fields["ends_at"] = endsAt
	}

	// Perubahan definisi menaikkan versi misi. Klaim lama menyimpan versinya
	// sendiri, sehingga arti klaim yang sudah berjalan tidak ikut berubah.
	if len(fields) > 0 {
		fields["version"] = existing.Version + 1
	}

	if err := s.Repository.Update(c, id, fields); err != nil {
		if isUniqueViolation(err) {
			return nil, invalid("kode misi sudah dipakai misi lain yang aktif")
		}
		return nil, err
	}
	return s.Repository.FindByID(c, id)
}

func (s *service) Delete(c *gin.Context, id string) error {
	if _, err := s.Repository.FindByID(c, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrMissionNotFound
		}
		return err
	}
	// Dikumpulkan sebelum misinya ditandai terhapus, supaya pencabutannya tetap
	// punya daftar sumber yang lengkap.
	claimIDs, err := s.Repository.ApprovedClaimIDsByMission(c, id)
	if err != nil {
		return err
	}
	if err := s.Repository.SoftDelete(c, id); err != nil {
		return err
	}
	for _, claimID := range claimIDs {
		s.removeCompletion(c, claimID)
	}
	return nil
}

func (s *service) ClaimsForReview(c *gin.Context, status string, page, perPage int) ([]*AdminClaimView, bool, error) {
	if status != "" {
		if err := ValidateClaimStatus(status); err != nil {
			return nil, false, err
		}
	}
	page, perPage = gamification.NormalizePagination(page, perPage)

	claims, err := s.Repository.ListClaimsForReview(c, status, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(claims) > perPage
	if hasMore {
		claims = claims[:perPage]
	}

	views := make([]*AdminClaimView, 0, len(claims))
	for _, claim := range claims {
		view := &AdminClaimView{MissionClaim: claim}
		if mission, err := s.Repository.FindByID(c, claim.MissionID); err == nil {
			view.MissionTitle = mission.Title
			view.MissionType = mission.Type
		}
		// Identitas pengklaim diambil terpisah; kegagalannya tidak boleh
		// menghilangkan klaim dari antrean pemeriksaan.
		if s.ClaimantResolver != nil {
			if claimant, err := s.ClaimantResolver.AdminIdentity(c, claim.UserID); err == nil {
				view.Claimant = claimant
			}
		}
		views = append(views, view)
	}
	return views, hasMore, nil
}

func (s *service) Approve(c *gin.Context, claimID string, req *DecideClaim) (*MissionClaim, error) {
	reason := ""
	if req != nil {
		reason = strings.TrimSpace(req.Reason)
	}
	return s.decide(c, claimID, reviewerID(c), StatusApproved, reason)
}

func (s *service) Reject(c *gin.Context, claimID string, req *DecideClaim) (*MissionClaim, error) {
	reason := ""
	if req != nil {
		reason = strings.TrimSpace(req.Reason)
	}
	if reason == "" {
		return nil, invalid("alasan penolakan wajib diisi")
	}
	return s.decide(c, claimID, reviewerID(c), StatusRejected, reason)
}

// decide adalah satu-satunya jalur yang mengubah status klaim.
//
// Idempoten dan tahan retry: menyetujui klaim yang sudah disetujui bukan
// kegagalan, melainkan kesempatan memperbaiki pemberian XP yang gagal
// sebelumnya. Kunci dedup ledger membuat perbaikan itu tidak pernah menggandakan
// reward.
func (s *service) decide(c *gin.Context, claimID, reviewerID, decision, reason string) (*MissionClaim, error) {
	claim, err := s.Repository.FindClaim(c, claimID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrClaimNotFound
		}
		return nil, err
	}

	if claim.Status == decision {
		if decision == StatusApproved {
			// Retry: pastikan reward-nya benar-benar ada.
			if err := s.grantReward(c, claim); err != nil {
				return nil, err
			}
			// Sekaligus memulihkan postingan yang mungkin tertinggal bila
			// proses mati setelah persetujuan tetapi sebelum feed ditulis.
			s.postCompletion(c, claim.UserID, claim.ID, claim.MissionID)
		}
		return claim, nil
	}
	if claim.Status != StatusPending {
		return nil, ErrClaimAlreadyDecided
	}

	var changed bool
	if decision == StatusApproved {
		mission, err := s.Repository.FindByID(c, claim.MissionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, ErrMissionNotFound
			}
			return nil, err
		}
		changed, err = s.Repository.ApproveClaim(c, claimID, reviewerID, reason, mission.RewardXP)
		if err != nil {
			return nil, err
		}
	} else {
		changed, err = s.Repository.RejectClaim(c, claimID, reviewerID, reason)
		if err != nil {
			return nil, err
		}
	}

	updated, err := s.Repository.FindClaim(c, claimID)
	if err != nil {
		return nil, err
	}

	if !changed {
		// Balapan: pengurus lain sudah memutuskan lebih dulu. Perlakukan
		// sebagai idempoten bila keputusannya sama.
		if updated.Status != decision {
			return nil, ErrClaimAlreadyDecided
		}
	}

	if decision == StatusApproved {
		if err := s.grantReward(c, updated); err != nil {
			return nil, err
		}
		s.postCompletion(c, updated.UserID, updated.ID, updated.MissionID)
	} else {
		// Klaim yang ditolak tidak boleh meninggalkan postingan di feed.
		s.removeCompletion(c, updated.ID)
	}
	return updated, nil
}

// grantReward memberi XP untuk klaim yang sudah disetujui.
//
// Kegagalan dikembalikan, bukan ditelan: pemanggil dapat mengulang approve,
// dan jalur retry di atas akan mencoba lagi. Reward tidak pernah digandakan
// karena kuncinya terikat pada id klaim.
func (s *service) grantReward(c *gin.Context, claim *MissionClaim) error {
	if s.RewardGranter == nil || claim == nil || claim.Status != StatusApproved {
		return nil
	}
	reward := 0
	if claim.RewardXP != nil {
		reward = *claim.RewardXP
	}
	if reward <= 0 {
		return nil
	}
	_, err := s.RewardGranter.GrantMissionReward(c, claim.UserID, claim.ID, reward)
	if err != nil {
		log.Printf("[xp] gagal memberi reward misi untuk klaim %s: %v", claim.ID, err)
	}
	return err
}

// toClaimViews melengkapi klaim dengan judul misinya untuk daftar riwayat.
func (s *service) toClaimViews(c *gin.Context, claims []*MissionClaim) ([]*ClaimView, error) {
	views := make([]*ClaimView, 0, len(claims))
	for _, claim := range claims {
		view := &ClaimView{MissionClaim: claim}
		// Misi yang sudah dihapus tetap ditampilkan sebagai klaim tanpa judul,
		// bukan menghilangkan riwayat klaimnya.
		if mission, err := s.Repository.FindByID(c, claim.MissionID); err == nil {
			view.MissionTitle = mission.Title
			view.MissionType = mission.Type
		}
		views = append(views, view)
	}
	return views, nil
}

// reviewerID mengambil identitas pemutus dari konteks autentikasi.
func reviewerID(c *gin.Context) string {
	if currentUser, ok := user.FromContext(c); ok {
		return currentUser.ID
	}
	return ""
}

// parseWindow memvalidasi rentang waktu misi.
//
// Waktu wajib memuat offset zona waktu eksplisit: tanpa itu, "09:00" akan
// ditafsirkan dalam zona server dan jadwal misi bergeser tanpa disadari.
func parseWindow(startsAt, endsAt string) (time.Time, time.Time, error) {
	start, err := parseTime(startsAt)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	end, err := parseTime(endsAt)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, invalid("waktu berakhir harus setelah waktu mulai")
	}
	return start, end, nil
}

func parseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, invalid("format waktu tidak valid, gunakan RFC3339 dengan offset zona waktu: %s", value)
	}
	return parsed, nil
}

// trimPtr merapikan string opsional; nilai kosong menjadi nil supaya tidak
// tersimpan sebagai string kosong yang sulit dibedakan dari "tidak diisi".
func trimPtr(s *string) *string {
	if s == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*s)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
