package fundraising

import (
	"errors"
	"log"
	"strings"
	"time"

	"mainyuk/internal/apperr"
	"mainyuk/internal/audit"
	"mainyuk/internal/community"
	"mainyuk/internal/gamification"
	"mainyuk/internal/payment"
	"mainyuk/internal/ratelimit"
	"mainyuk/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// service menyatukan aturan fundraising. Semua jalur tulis lewat sini supaya
// validasi, transisi status, XP, dan audit tidak bisa dilewati satu pun.
type service struct {
	Repository
	Donors   DonorResolver
	Rewards  RewardGranter
	Methods  MethodResolver
	Provider PaymentProvider
	// poster boleh nil; nil berarti donasi tidak dibagikan ke feed, seperti
	// sebelum Fase 3.
	poster AutoPoster
}

// SetAutoPoster memasang peniti donasi terkonfirmasi ke feed komunitas.
func (s *service) SetAutoPoster(poster AutoPoster) {
	s.poster = poster
}

// postDonation membagikan donasi terkonfirmasi ke feed.
//
// Dipanggil atas dasar status akhirnya, bukan atas dasar "barisnya baru
// berpindah": dedup ada di index unik sumber, jadi percobaan ulang aman dan
// justru memulihkan postingan yang tertinggal.
//
// Nominal tidak pernah ikut dikirim. Donasi anonim pun tidak pernah dibagikan —
// keputusan itu ada di fungsi murni ShouldAutoPost, dan di sini cukup
// meneruskan flag-nya.
func (s *service) postDonation(c *gin.Context, donation *Donation) {
	if s.poster == nil || donation == nil {
		return
	}
	campaign, err := s.findCampaignByID(c, donation.CampaignID)
	if err != nil || campaign == nil {
		log.Printf("[thread] gagal membaca campaign donasi %s: %v", donation.ID, err)
		return
	}
	if err := s.poster.PostDonation(c, donation.UserID, donation.ID, campaign.Title, donation.IsAnonymous); err != nil {
		log.Printf("[thread] gagal membagikan donasi %s ke feed: %v", donation.ID, err)
	}
}

// removeDonation mencabut postingan donasi, misalnya setelah dananya
// dikembalikan.
func (s *service) removeDonation(c *gin.Context, donationID string) {
	if s.poster == nil {
		return
	}
	if err := s.poster.RemoveDonation(c, donationID); err != nil {
		log.Printf("[thread] gagal mencabut donasi %s dari feed: %v", donationID, err)
	}
}

func NewService(repository Repository, donors DonorResolver, rewards RewardGranter, methods MethodResolver, provider PaymentProvider) Service {
	return &service{
		Repository: repository,
		Donors:     donors,
		Rewards:    rewards,
		Methods:    methods,
		Provider:   provider,
	}
}

// ---------------------------------------------------------------------------
// Permukaan publik
// ---------------------------------------------------------------------------

func (s *service) ListCampaigns(c *gin.Context) ([]*CampaignView, error) {
	now := time.Now()
	campaigns, err := s.Repository.ListPublished(c, now)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(campaigns))
	for _, campaign := range campaigns {
		ids = append(ids, campaign.ID)
	}
	totals, err := s.Repository.CampaignTotalsBatch(c, ids)
	if err != nil {
		return nil, err
	}

	views := make([]*CampaignView, 0, len(campaigns))
	for _, campaign := range campaigns {
		views = append(views, toCampaignView(campaign, totals[campaign.ID], now))
	}
	return views, nil
}

func (s *service) ShowCampaign(c *gin.Context, slug string) (*CampaignView, error) {
	campaign, err := s.findPublishedCampaign(c, slug)
	if err != nil {
		return nil, err
	}
	totals, err := s.Repository.CampaignTotals(c, campaign.ID)
	if err != nil {
		return nil, err
	}
	return toCampaignView(campaign, totals, time.Now()), nil
}

// ListDonors mengembalikan daftar donatur publik satu campaign.
//
// Anonimitas ditegakkan satu kali di sini lewat ResolveDonorName +
// ToPublicDonation, jadi tidak ada endpoint yang bisa lupa menerapkannya.
func (s *service) ListDonors(c *gin.Context, slug string, page, perPage int) ([]*PublicDonation, bool, error) {
	campaign, err := s.findPublishedCampaign(c, slug)
	if err != nil {
		return nil, false, err
	}
	page, perPage = gamification.NormalizePagination(page, perPage)

	donations, err := s.Repository.ListDonors(c, campaign.ID, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(donations) > perPage
	if hasMore {
		donations = donations[:perPage]
	}
	return s.toPublicDonations(c, donations), hasMore, nil
}

// ListMessages mengembalikan donasi terkonfirmasi yang pesannya sudah
// disetujui. Ini satu-satunya jalur yang boleh menampilkan pesan donasi.
func (s *service) ListMessages(c *gin.Context, slug string, page, perPage int) ([]*PublicDonation, bool, error) {
	campaign, err := s.findPublishedCampaign(c, slug)
	if err != nil {
		return nil, false, err
	}
	page, perPage = gamification.NormalizePagination(page, perPage)

	donations, err := s.Repository.ListApprovedMessages(c, campaign.ID, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(donations) > perPage
	if hasMore {
		donations = donations[:perPage]
	}
	return s.toPublicDonations(c, donations), hasMore, nil
}

func (s *service) ListUpdates(c *gin.Context, slug string) ([]*CampaignUpdate, error) {
	campaign, err := s.findPublishedCampaign(c, slug)
	if err != nil {
		return nil, err
	}
	return s.Repository.ListUpdates(c, campaign.ID, true)
}

// PublicReport adalah laporan ringkas yang aman ditampilkan di halaman
// campaign: total per status tanpa identitas donatur.
func (s *service) PublicReport(c *gin.Context, slug string) (*ReportView, error) {
	campaign, err := s.findPublishedCampaign(c, slug)
	if err != nil {
		return nil, err
	}
	return s.buildReport(c, campaign, true)
}

// ---------------------------------------------------------------------------
// Anggota
// ---------------------------------------------------------------------------

// CreateDonation membuat donasi baru untuk pemanggil yang sudah masuk.
//
// Identitas donatur diambil dari konteks auth, tidak pernah dari body. Token
// idempotensi membuat penekanan tombol dua kali menghasilkan donasi yang sama,
// bukan dua donasi.
func (s *service) CreateDonation(c *gin.Context, req *CreateDonation) (*DonationResult, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	if !ratelimit.Donation.Allow(ratelimit.Key(c, currentUser.ID)) {
		return nil, ratelimit.ErrTooManyRequests
	}

	clientToken := strings.TrimSpace(req.ClientToken)
	if clientToken == "" {
		return nil, invalid("token idempotensi donasi wajib diisi")
	}
	if err := ValidateDonationAmount(req.Amount); err != nil {
		return nil, err
	}

	now := time.Now()
	campaign, err := s.findPublishedCampaign(c, req.CampaignSlug)
	if err != nil {
		return nil, err
	}
	if !DonationWindowOpen(now, campaign.Status, campaign.StartsAt, campaign.EndsAt) {
		return nil, ErrDonationWindowClosed
	}

	// Anonim menyembunyikan nama DAN nominal sekaligus. Dipaksa di server,
	// bukan diserahkan ke klien, supaya tidak ada jalur yang bisa memecah
	// keduanya.
	showAmount := true
	if req.ShowAmount != nil {
		showAmount = *req.ShowAmount
	}
	if req.IsAnonymous {
		showAmount = false
	}

	message, messageStatus := NormalizeMessage(req.Message)

	donation := &Donation{
		ID:              uuid.NewString(),
		PublicID:        newPublicID(),
		CampaignID:      campaign.ID,
		UserID:          currentUser.ID,
		Amount:          req.Amount,
		Status:          DonationPending,
		IsAnonymous:     req.IsAnonymous,
		ShowAmount:      showAmount,
		Message:         message,
		MessageStatus:   messageStatus,
		PaymentMethodID: trimPtr(req.PaymentMethodID),
		ClientToken:     &clientToken,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	stored, created, err := s.Repository.CreateDonationGuarded(c, donation)
	if err != nil {
		return nil, err
	}
	if !created {
		log.Printf("[fundraising] donasi idempoten dipakai ulang untuk user %s", currentUser.ID)
	}

	return s.donationResult(c, stored, campaign)
}

func (s *service) MyDonations(c *gin.Context, page, perPage int) ([]*DonationView, bool, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, false, apperr.ErrUnauthorized
	}
	page, perPage = gamification.NormalizePagination(page, perPage)

	donations, err := s.Repository.ListDonationsByUser(c, currentUser.ID, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(donations) > perPage
	if hasMore {
		donations = donations[:perPage]
	}

	views := make([]*DonationView, 0, len(donations))
	for _, donation := range donations {
		views = append(views, s.toDonationView(c, donation))
	}
	return views, hasMore, nil
}

func (s *service) MyDonation(c *gin.Context, publicID string) (*DonationResult, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}

	donation, err := s.Repository.FindDonationByPublicID(c, publicID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDonationNotFound
		}
		return nil, err
	}
	// Donasi orang lain tidak dibedakan dari donasi yang tidak ada: pemanggil
	// tidak berhak tahu bahwa id itu ada.
	if donation.UserID != currentUser.ID {
		return nil, ErrDonationNotFound
	}

	campaign, err := s.Repository.FindByID(c, donation.CampaignID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}
	return s.donationResult(c, donation, campaign)
}

// ---------------------------------------------------------------------------
// Pengurus — campaign
// ---------------------------------------------------------------------------

func (s *service) ListAllCampaigns(c *gin.Context, status string) ([]*CampaignView, error) {
	if status != "" && !IsValidCampaignStatus(status) {
		return nil, invalid("status campaign tidak dikenal: %s", status)
	}
	now := time.Now()
	campaigns, err := s.Repository.ListAll(c, status)
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(campaigns))
	for _, campaign := range campaigns {
		ids = append(ids, campaign.ID)
	}
	totals, err := s.Repository.CampaignTotalsBatch(c, ids)
	if err != nil {
		return nil, err
	}

	views := make([]*CampaignView, 0, len(campaigns))
	for _, campaign := range campaigns {
		views = append(views, toCampaignView(campaign, totals[campaign.ID], now))
	}
	return views, nil
}

func (s *service) CreateCampaign(c *gin.Context, req *CreateCampaign) (*Campaign, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	slug := NormalizeSlug(req.Slug)
	if slug == "" {
		return nil, invalid("slug campaign wajib diisi")
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, invalid("judul campaign wajib diisi")
	}
	recipient := strings.TrimSpace(req.Recipient)
	if recipient == "" {
		return nil, invalid("penerima dana wajib diisi")
	}
	if !IsValidFundType(req.FundType) {
		return nil, invalid("jenis dana tidak dikenal: %s", req.FundType)
	}
	if req.TargetAmount < 0 {
		return nil, invalid("target dana tidak boleh negatif")
	}

	startsAt, endsAt, err := parseOptionalWindow(req.StartsAt, req.EndsAt)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	campaign := &Campaign{
		ID:            uuid.NewString(),
		Slug:          slug,
		Title:         title,
		Summary:       trimPtr(req.Summary),
		Story:         trimPtr(req.Story),
		CoverImageURL: trimPtr(req.CoverImageURL),
		FundType:      req.FundType,
		Recipient:     recipient,
		TargetAmount:  req.TargetAmount,
		Status:        CampaignDraft,
		StartsAt:      startsAt,
		EndsAt:        endsAt,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if currentUser, ok := user.FromContext(c); ok {
		campaign.CreatedBy = &currentUser.ID
	}

	if err := s.Repository.CreateCampaign(c, campaign); err != nil {
		if isUniqueViolation(err) {
			return nil, invalid("slug campaign sudah dipakai campaign lain yang aktif")
		}
		return nil, err
	}

	s.writeAudit(c, "campaign.create", "campaign", campaign.ID, nil, map[string]any{
		"slug":   campaign.Slug,
		"status": campaign.Status,
	})
	return campaign, nil
}

func (s *service) UpdateCampaign(c *gin.Context, id string, req *UpdateCampaign) (*Campaign, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	existing, err := s.findCampaignByID(c, id)
	if err != nil {
		return nil, err
	}

	fields := map[string]interface{}{}

	if req.Slug != nil {
		slug := NormalizeSlug(*req.Slug)
		if slug == "" {
			return nil, invalid("slug campaign wajib diisi")
		}
		fields["slug"] = slug
	}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return nil, invalid("judul campaign wajib diisi")
		}
		fields["title"] = title
	}
	if req.Summary != nil {
		fields["summary"] = trimPtr(req.Summary)
	}
	if req.Story != nil {
		fields["story"] = trimPtr(req.Story)
	}
	if req.CoverImageURL != nil {
		fields["cover_image_url"] = trimPtr(req.CoverImageURL)
	}
	if req.FundType != nil {
		if !IsValidFundType(*req.FundType) {
			return nil, invalid("jenis dana tidak dikenal: %s", *req.FundType)
		}
		fields["fund_type"] = *req.FundType
	}
	if req.Recipient != nil {
		recipient := strings.TrimSpace(*req.Recipient)
		if recipient == "" {
			return nil, invalid("penerima dana wajib diisi")
		}
		fields["recipient"] = recipient
	}
	if req.TargetAmount != nil {
		if *req.TargetAmount < 0 {
			return nil, invalid("target dana tidak boleh negatif")
		}
		fields["target_amount"] = *req.TargetAmount
	}

	if req.StartsAt != nil || req.EndsAt != nil {
		startsAt := existing.StartsAt
		endsAt := existing.EndsAt
		if req.StartsAt != nil {
			parsed, err := parseOptionalTime(*req.StartsAt)
			if err != nil {
				return nil, err
			}
			startsAt = parsed
		}
		if req.EndsAt != nil {
			parsed, err := parseOptionalTime(*req.EndsAt)
			if err != nil {
				return nil, err
			}
			endsAt = parsed
		}
		if startsAt != nil && endsAt != nil && !endsAt.After(*startsAt) {
			return nil, invalid("waktu berakhir harus setelah waktu mulai")
		}
		fields["starts_at"] = startsAt
		fields["ends_at"] = endsAt
	}

	if len(fields) == 0 {
		return existing, nil
	}

	if err := s.Repository.UpdateCampaign(c, id, fields); err != nil {
		if isUniqueViolation(err) {
			return nil, invalid("slug campaign sudah dipakai campaign lain yang aktif")
		}
		return nil, err
	}

	s.writeAudit(c, "campaign.update", "campaign", id, nil, map[string]any{"fields": fieldNames(fields)})
	return s.Repository.FindByID(c, id)
}

// SetCampaignStatus memindahkan status campaign lewat state machine.
//
// Transisi tidak sah ditolak sebelum menyentuh database, dan perpindahan yang
// benar-benar terjadi dicatat ke audit.
func (s *service) SetCampaignStatus(c *gin.Context, id string, req *SetCampaignStatus) (*Campaign, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	if !IsValidCampaignStatus(req.Status) {
		return nil, invalid("status campaign tidak dikenal: %s", req.Status)
	}

	campaign, err := s.findCampaignByID(c, id)
	if err != nil {
		return nil, err
	}
	if campaign.Status == req.Status {
		return campaign, nil
	}
	if !CanTransitionCampaign(campaign.Status, req.Status) {
		return nil, ErrInvalidTransition
	}

	fields := map[string]interface{}{}
	now := time.Now()
	switch req.Status {
	case CampaignPublished:
		fields["published_at"] = now
	case CampaignClosed:
		fields["closed_at"] = now
	}

	changed, err := s.Repository.TransitionCampaign(c, id, campaign.Status, req.Status, fields)
	if err != nil {
		return nil, err
	}

	updated, err := s.Repository.FindByID(c, id)
	if err != nil {
		return nil, err
	}
	if !changed && updated.Status != req.Status {
		// Balapan: pengurus lain memindahkan lebih dulu ke status lain.
		return nil, ErrInvalidTransition
	}
	if changed {
		s.writeAudit(c, "campaign.status", "campaign", id, trimPtr(&req.Reason), map[string]any{
			"from": campaign.Status,
			"to":   req.Status,
		})
	}
	return updated, nil
}

func (s *service) DeleteCampaign(c *gin.Context, id string) error {
	if _, err := s.findCampaignByID(c, id); err != nil {
		return err
	}
	if err := s.Repository.SoftDeleteCampaign(c, id); err != nil {
		return err
	}
	s.writeAudit(c, "campaign.delete", "campaign", id, nil, nil)
	return nil
}

func (s *service) CampaignReport(c *gin.Context, id string) (*ReportView, error) {
	campaign, err := s.findCampaignByID(c, id)
	if err != nil {
		return nil, err
	}
	return s.buildReport(c, campaign, false)
}

// ---------------------------------------------------------------------------
// Pengurus — donasi
// ---------------------------------------------------------------------------

func (s *service) DonationsForReview(c *gin.Context, filter DonationFilter, page, perPage int) ([]*AdminDonationView, bool, error) {
	if filter.Status != "" && !isValidDonationStatus(filter.Status) {
		return nil, false, invalid("status donasi tidak dikenal: %s", filter.Status)
	}
	if filter.MessageStatus != "" && !IsValidMessageStatus(filter.MessageStatus) {
		return nil, false, invalid("status pesan tidak dikenal: %s", filter.MessageStatus)
	}
	page, perPage = gamification.NormalizePagination(page, perPage)

	donations, err := s.Repository.ListDonationsForReview(c, filter, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(donations) > perPage
	if hasMore {
		donations = donations[:perPage]
	}

	views := make([]*AdminDonationView, 0, len(donations))
	for _, donation := range donations {
		slug, title := s.campaignLabel(c, donation.CampaignID)
		var donor *community.PublicProfile
		// Identitas donatur diambil terpisah; kegagalannya tidak boleh
		// menghilangkan donasi dari antrean pemeriksaan.
		if s.Donors != nil {
			if identity, err := s.Donors.AdminIdentity(c, donation.UserID); err == nil {
				donor = identity
			}
		}
		views = append(views, ToAdminDonationView(donation, donor, slug, title))
	}
	return views, hasMore, nil
}

// ConfirmDonation memverifikasi transfer dan memberi XP.
//
// Idempoten: mengonfirmasi donasi yang sudah terkonfirmasi bukan kegagalan,
// melainkan kesempatan memperbaiki pemberian XP yang gagal sebelumnya. Kunci
// dedup ledger membuat perbaikan itu tidak pernah menggandakan reward.
func (s *service) ConfirmDonation(c *gin.Context, id string, req *ConfirmDonation) (*Donation, error) {
	if !ratelimit.DonationReview.Allow(ratelimit.Key(c, reviewerID(c))) {
		return nil, ratelimit.ErrTooManyRequests
	}

	donation, err := s.findDonationByID(c, id)
	if err != nil {
		return nil, err
	}

	if donation.Status == DonationConfirmed {
		if err := s.grantDonationReward(c, donation); err != nil {
			return nil, err
		}
		return s.Repository.FindDonationByID(c, id)
	}
	if donation.Status != DonationPending {
		return nil, ErrDonationAlreadyDecided
	}

	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	paidAmount := req.PaidAmount
	if paidAmount <= 0 {
		paidAmount = donation.Amount
	}
	if err := VerifyManualPayment(donation.Amount, paidAmount, req.PaymentReference); err != nil {
		return nil, err
	}

	now := time.Now()
	fields := map[string]interface{}{
		"paid_amount":       paidAmount,
		"payment_reference": strings.TrimSpace(req.PaymentReference),
		"proof_url":         trimPtr(req.ProofURL),
		"confirmed_at":      now,
		"decision_reason":   trimPtr(req.Reason),
	}
	if actor := reviewerID(c); actor != "" {
		fields["confirmed_by"] = actor
	}

	changed, err := s.Repository.TransitionDonation(c, id, DonationPending, DonationConfirmed, fields)
	if err != nil {
		return nil, err
	}

	updated, err := s.Repository.FindDonationByID(c, id)
	if err != nil {
		return nil, err
	}
	if !changed {
		// Balapan: pengurus lain memutuskan lebih dulu.
		if updated.Status != DonationConfirmed {
			return nil, ErrDonationAlreadyDecided
		}
	} else {
		s.writeAudit(c, "donation.confirm", "donation", id, trimPtr(req.Reason), map[string]any{
			"amount":      donation.Amount,
			"paid_amount": paidAmount,
		})
	}

	if err := s.grantDonationReward(c, updated); err != nil {
		return nil, err
	}
	s.postDonation(c, updated)
	return s.Repository.FindDonationByID(c, id)
}

func (s *service) RejectDonation(c *gin.Context, id string, req *DecideDonation) (*Donation, error) {
	reason, err := requireReason(req)
	if err != nil {
		return nil, err
	}

	donation, err := s.findDonationByID(c, id)
	if err != nil {
		return nil, err
	}
	if donation.Status == DonationRejected {
		return donation, nil
	}
	if donation.Status != DonationPending {
		return nil, ErrDonationAlreadyDecided
	}

	changed, err := s.Repository.TransitionDonation(c, id, DonationPending, DonationRejected, map[string]interface{}{
		"decision_reason": reason,
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.Repository.FindDonationByID(c, id)
	if err != nil {
		return nil, err
	}
	if !changed && updated.Status != DonationRejected {
		return nil, ErrDonationAlreadyDecided
	}
	if changed {
		s.writeAudit(c, "donation.reject", "donation", id, &reason, nil)
	}
	return updated, nil
}

// RefundDonation mengembalikan dana dan membalik XP yang pernah diberikan.
//
// Pembalikan memakai snapshot rewarded_xp, bukan nilai aturan saat ini, supaya
// XP yang dikembalikan persis sebesar yang pernah diberikan walaupun pengurus
// mengubah xp_rules setelahnya.
func (s *service) RefundDonation(c *gin.Context, id string, req *DecideDonation) (*Donation, error) {
	reason, err := requireReason(req)
	if err != nil {
		return nil, err
	}

	donation, err := s.findDonationByID(c, id)
	if err != nil {
		return nil, err
	}
	if donation.Status == DonationRefunded {
		return donation, nil
	}
	if donation.Status != DonationConfirmed {
		return nil, ErrDonationAlreadyDecided
	}

	changed, err := s.Repository.TransitionDonation(c, id, DonationConfirmed, DonationRefunded, map[string]interface{}{
		"decision_reason": reason,
	})
	if err != nil {
		return nil, err
	}

	updated, err := s.Repository.FindDonationByID(c, id)
	if err != nil {
		return nil, err
	}
	if !changed && updated.Status != DonationRefunded {
		return nil, ErrDonationAlreadyDecided
	}

	if changed {
		s.writeAudit(c, "donation.refund", "donation", id, &reason, map[string]any{
			"rewarded_xp": updated.RewardedXP,
		})
	}

	// XP dibalik hanya bila pernah diberikan. Kunci dedup deterministik
	// membuat pengulangan refund tidak pernah membalik dua kali.
	if updated.RewardedXP > 0 && s.Rewards != nil {
		if err := s.Rewards.ReverseDonationReward(c, updated.UserID, updated.ID, updated.RewardedXP, reason); err != nil {
			log.Printf("[xp] gagal membalik reward donasi %s: %v", updated.ID, err)
			return nil, err
		}
	}

	// Donasi yang dananya dikembalikan tidak boleh meninggalkan postingan di
	// feed. Dipanggil di luar blok `changed` supaya refund yang diulang tetap
	// mencabutnya bila percobaan pertama gagal.
	s.removeDonation(c, updated.ID)
	return updated, nil
}

// ModerateMessage memutuskan tampil-tidaknya pesan donasi.
//
// Pesan hanya muncul di permukaan publik setelah disetujui. Donasi tanpa pesan
// berstatus 'none' dan tidak pernah masuk antrean ini.
func (s *service) ModerateMessage(c *gin.Context, id string, req *ModerateMessage) (*Donation, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	target := ""
	switch strings.ToLower(strings.TrimSpace(req.Decision)) {
	case "approve", "approved":
		target = MessageApproved
	case "hide", "hidden":
		target = MessageHidden
	default:
		return nil, invalid("keputusan moderasi tidak dikenal: %s", req.Decision)
	}

	donation, err := s.findDonationByID(c, id)
	if err != nil {
		return nil, err
	}
	if donation.Message == nil || donation.MessageStatus == MessageNone {
		return nil, invalid("donasi ini tidak punya pesan untuk dimoderasi")
	}
	if donation.MessageStatus == target {
		return donation, nil
	}
	if !CanTransitionMessage(donation.MessageStatus, target) {
		return nil, ErrInvalidTransition
	}

	changed, err := s.Repository.TransitionMessage(c, id, donation.MessageStatus, target, nil)
	if err != nil {
		return nil, err
	}

	updated, err := s.Repository.FindDonationByID(c, id)
	if err != nil {
		return nil, err
	}
	if !changed && updated.MessageStatus != target {
		return nil, ErrInvalidTransition
	}

	if changed {
		action := "message.approve"
		if target == MessageHidden {
			action = "message.hide"
		}
		s.writeAudit(c, action, "donation", id, trimPtr(&req.Reason), map[string]any{
			"from": donation.MessageStatus,
			"to":   target,
		})
	}
	return updated, nil
}

// ---------------------------------------------------------------------------
// Pengurus — update campaign
// ---------------------------------------------------------------------------

func (s *service) CreateUpdate(c *gin.Context, campaignID string, req *CampaignUpdateInput) (*CampaignUpdate, error) {
	campaign, err := s.findCampaignByID(c, campaignID)
	if err != nil {
		return nil, err
	}
	fields, err := s.updateFields(req)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	update := &CampaignUpdate{
		ID:          uuid.NewString(),
		CampaignID:  campaign.ID,
		Title:       fields.title,
		Body:        fields.body,
		Kind:        fields.kind,
		Amount:      fields.amount,
		ProofURL:    fields.proofURL,
		IsPublished: fields.isPublished,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if currentUser, ok := user.FromContext(c); ok {
		update.CreatedBy = &currentUser.ID
	}

	if err := s.Repository.CreateUpdate(c, update); err != nil {
		return nil, err
	}

	s.writeAudit(c, "campaign_update.create", "campaign_update", update.ID, nil, map[string]any{
		"campaign_id": campaign.ID,
		"kind":        update.Kind,
		"amount":      update.Amount,
	})
	return update, nil
}

func (s *service) UpdateUpdate(c *gin.Context, id string, req *CampaignUpdateInput) (*CampaignUpdate, error) {
	existing, err := s.Repository.FindUpdate(c, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUpdateNotFound
		}
		return nil, err
	}
	fields, err := s.updateFields(req)
	if err != nil {
		return nil, err
	}

	patch := map[string]interface{}{
		"title":        fields.title,
		"body":         fields.body,
		"kind":         fields.kind,
		"amount":       fields.amount,
		"proof_url":    fields.proofURL,
		"is_published": fields.isPublished,
	}
	if err := s.Repository.UpdateUpdate(c, id, patch); err != nil {
		return nil, err
	}

	s.writeAudit(c, "campaign_update.update", "campaign_update", id, nil, map[string]any{
		"campaign_id": existing.CampaignID,
	})
	return s.Repository.FindUpdate(c, id)
}

func (s *service) DeleteUpdate(c *gin.Context, id string) error {
	existing, err := s.Repository.FindUpdate(c, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUpdateNotFound
		}
		return err
	}
	if err := s.Repository.SoftDeleteUpdate(c, id); err != nil {
		return err
	}
	s.writeAudit(c, "campaign_update.delete", "campaign_update", id, nil, map[string]any{
		"campaign_id": existing.CampaignID,
	})
	return nil
}

func (s *service) ListAuditLogs(c *gin.Context, entityType, entityID string, page, perPage int) ([]*AuditLog, bool, error) {
	page, perPage = gamification.NormalizePagination(page, perPage)
	entries, err := s.Repository.ListAudit(c, entityType, entityID, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(entries) > perPage
	if hasMore {
		entries = entries[:perPage]
	}
	return entries, hasMore, nil
}

// ---------------------------------------------------------------------------
// Pembantu
// ---------------------------------------------------------------------------

// updateFields memvalidasi payload update campaign.
//
// 'usage' dan 'fee' wajib membawa nominal: keduanya muncul sebagai baris
// laporan, dan baris tanpa nominal akan tampak seperti pencatatan yang gagal.
type updateInput struct {
	title       string
	body        string
	kind        string
	amount      *int
	proofURL    *string
	isPublished bool
}

func (s *service) updateFields(req *CampaignUpdateInput) (*updateInput, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, invalid("judul update wajib diisi")
	}
	body := strings.TrimSpace(req.Body)
	if body == "" {
		return nil, invalid("isi update wajib diisi")
	}

	kind := strings.TrimSpace(req.Kind)
	if kind == "" {
		kind = UpdateKindUpdate
	}
	switch kind {
	case UpdateKindUpdate, UpdateKindUsage, UpdateKindFee:
	default:
		return nil, invalid("jenis update tidak dikenal: %s", kind)
	}

	if kind != UpdateKindUpdate {
		if req.Amount == nil {
			return nil, invalid("baris penggunaan dana dan biaya wajib mencantumkan nominal")
		}
		if *req.Amount < 0 {
			return nil, invalid("nominal tidak boleh negatif")
		}
	}

	isPublished := false
	if req.IsPublished != nil {
		isPublished = *req.IsPublished
	}

	return &updateInput{
		title:       title,
		body:        body,
		kind:        kind,
		amount:      req.Amount,
		proofURL:    trimPtr(req.ProofURL),
		isPublished: isPublished,
	}, nil
}

// buildReport menyusun laporan satu campaign.
//
// Progres selalu bruto; biaya dan penggunaan dana hanya muncul sebagai baris
// terpisah. publishedOnly menyaring update agar halaman publik tidak
// menampilkan catatan yang belum diterbitkan pengurus.
func (s *service) buildReport(c *gin.Context, campaign *Campaign, publishedOnly bool) (*ReportView, error) {
	in, err := s.Repository.DonationAggregates(c, campaign.ID)
	if err != nil {
		return nil, err
	}
	updates, err := s.Repository.ListUpdates(c, campaign.ID, publishedOnly)
	if err != nil {
		return nil, err
	}
	return &ReportView{
		Campaign: campaign,
		Totals:   ComputeReport(*in),
		Updates:  updates,
	}, nil
}

// donationResult menyusun donasi + instruksi pembayarannya untuk klien.
func (s *service) donationResult(c *gin.Context, donation *Donation, campaign *Campaign) (*DonationResult, error) {
	view := &DonationView{
		Donation:      donation,
		CampaignSlug:  campaign.Slug,
		CampaignTitle: campaign.Title,
	}

	var charge *Charge
	if s.Provider != nil {
		// Sejak Fase 4 provider tidak lagi menerima baris donasi utuh, hanya
		// nominal dan metode pembayarannya — itulah yang membuat adapter yang
		// sama dapat dipakai merchandise tanpa mengenal tabel donasi.
		built, err := s.Provider.CreateCharge(c, payment.ChargeRequest{
			Amount:          donation.Amount,
			PaymentMethodID: donation.PaymentMethodID,
		})
		if err != nil {
			return nil, err
		}
		charge = built
	}
	return &DonationResult{Donation: view, Charge: charge}, nil
}

func (s *service) toDonationView(c *gin.Context, donation *Donation) *DonationView {
	slug, title := s.campaignLabel(c, donation.CampaignID)
	return &DonationView{
		Donation:      donation,
		CampaignSlug:  slug,
		CampaignTitle: title,
	}
}

// toPublicDonations menerapkan aturan anonimitas satu kali untuk semua
// donasi yang akan tampil di permukaan publik.
func (s *service) toPublicDonations(c *gin.Context, donations []*Donation) []*PublicDonation {
	views := make([]*PublicDonation, 0, len(donations))
	for _, donation := range donations {
		var profile *community.Profile
		if s.Donors != nil {
			// Kegagalan membaca profil diperlakukan sebagai anonim: lebih baik
			// donaturnya tampil sebagai "Hamba Allah" daripada identitasnya
			// bocor karena pemeriksaan visibilitas tidak sempat berjalan.
			if resolved, err := s.Donors.EnsureProfile(c, donation.UserID); err == nil {
				profile = resolved
			}
		}
		name, hidden := ResolveDonorName(profile, "")
		var avatar *string
		if profile != nil {
			avatar = profile.AvatarURL
		}
		views = append(views, ToPublicDonation(donation, name, hidden, avatar))
	}
	return views
}

// grantDonationReward memberi XP untuk donasi yang sudah terkonfirmasi dan
// menyimpan snapshot-nya.
//
// XP yang diberikan dihitung gamification (termasuk batas bulanan); nilai yang
// dikembalikan adalah yang benar-benar masuk ledger, dan itulah yang disimpan
// supaya pembalikan saat refund tepat. Donasi yang kena batas tetap sah dan
// tetap dihitung progres — hanya XP-nya nol.
func (s *service) grantDonationReward(c *gin.Context, donation *Donation) error {
	if s.Rewards == nil || donation == nil || donation.Status != DonationConfirmed {
		return nil
	}
	xp, err := s.Rewards.GrantDonationReward(c, donation.UserID, donation.ID)
	if err != nil {
		return err
	}
	if xp <= 0 {
		return nil
	}
	return s.Repository.SetDonationReward(c, donation.ID, xp)
}

// writeAudit mencatat satu keputusan pengurus.
//
// Hanya dipanggil setelah penulisan terjaga yang mendahuluinya benar-benar
// mengubah baris, sehingga percobaan ulang tidak menghasilkan baris audit
// ganda. Kegagalan menulis audit tidak membatalkan aksi yang sudah terjadi —
// ia dilaporkan ke log, bukan dikembalikan sebagai galat.
func (s *service) writeAudit(c *gin.Context, action, entityType, entityID string, reason *string, detail map[string]any) {
	entry := audit.NewEntry(reviewerID(c), action, entityType, entityID, reason, detail)

	if err := s.Repository.WriteAudit(c, entry); err != nil {
		log.Printf("[audit] gagal menulis audit %s %s/%s: %v", action, entityType, entityID, err)
	}
}

func (s *service) findPublishedCampaign(c *gin.Context, slug string) (*Campaign, error) {
	campaign, err := s.Repository.FindPublishedBySlug(c, strings.TrimSpace(slug), time.Now())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}
	return campaign, nil
}

func (s *service) findCampaignByID(c *gin.Context, id string) (*Campaign, error) {
	campaign, err := s.Repository.FindByID(c, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrCampaignNotFound
		}
		return nil, err
	}
	return campaign, nil
}

func (s *service) findDonationByID(c *gin.Context, id string) (*Donation, error) {
	donation, err := s.Repository.FindDonationByID(c, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrDonationNotFound
		}
		return nil, err
	}
	return donation, nil
}

// campaignLabel mengambil slug dan judul campaign untuk melengkapi tampilan
// donasi. Campaign yang gagal dibaca menghasilkan label kosong, bukan
// menghilangkan donasinya.
func (s *service) campaignLabel(c *gin.Context, campaignID string) (string, string) {
	campaign, err := s.Repository.FindByID(c, campaignID)
	if err != nil || campaign == nil {
		return "", ""
	}
	return campaign.Slug, campaign.Title
}

// toCampaignView melengkapi campaign dengan progres yang dihitung saat diakses.
func toCampaignView(campaign *Campaign, totals *CampaignTotals, now time.Time) *CampaignView {
	raised, donors := 0, 0
	if totals != nil {
		raised, donors = totals.RaisedAmount, totals.DonorCount
	}
	return &CampaignView{
		Campaign:        campaign,
		RaisedAmount:    raised,
		DonorCount:      donors,
		ProgressPercent: ProgressPercent(raised, campaign.TargetAmount),
		IsOpen:          DonationWindowOpen(now, campaign.Status, campaign.StartsAt, campaign.EndsAt),
	}
}

// isValidDonationStatus melaporkan apakah status donasi dikenal.
func isValidDonationStatus(status string) bool {
	switch status {
	case DonationPending, DonationConfirmed, DonationRejected, DonationCancelled, DonationRefunded:
		return true
	}
	return false
}

// requireReason menuntut alasan yang tidak kosong untuk keputusan yang
// mengubah uang.
func requireReason(req *DecideDonation) (string, error) {
	if req == nil {
		return "", apperr.ErrInvalidRequest
	}
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return "", invalid("alasan wajib diisi")
	}
	return reason, nil
}

// reviewerID mengambil identitas pengurus dari konteks autentikasi.
func reviewerID(c *gin.Context) string {
	if currentUser, ok := user.FromContext(c); ok {
		return currentUser.ID
	}
	return ""
}

// newPublicID membuat rujukan publik donasi: 20 karakter heksadesimal dari
// UUID tanpa tanda hubung. Cukup panjang untuk tidak mudah ditebak, cukup
// pendek untuk ditempel di tautan.
func newPublicID() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")[:20]
}

// parseOptionalWindow memvalidasi rentang waktu campaign.
//
// Keduanya opsional: campaign boleh berjalan tanpa jadwal. Bila keduanya ada,
// akhir harus setelah awal.
func parseOptionalWindow(startsAt, endsAt *string) (*time.Time, *time.Time, error) {
	start, err := parseOptionalTimePtr(startsAt)
	if err != nil {
		return nil, nil, err
	}
	end, err := parseOptionalTimePtr(endsAt)
	if err != nil {
		return nil, nil, err
	}
	if start != nil && end != nil && !end.After(*start) {
		return nil, nil, invalid("waktu berakhir harus setelah waktu mulai")
	}
	return start, end, nil
}

func parseOptionalTimePtr(value *string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	return parseOptionalTime(*value)
}

// parseOptionalTime mengurai waktu RFC3339; string kosong berarti tanpa jadwal.
//
// Offset zona waktu wajib eksplisit: tanpa itu, jam akan ditafsirkan dalam zona
// server dan jadwal campaign bergeser tanpa disadari.
func parseOptionalTime(value string) (*time.Time, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, invalid("format waktu tidak valid, gunakan RFC3339 dengan offset zona waktu: %s", value)
	}
	return &parsed, nil
}

// fieldNames mengembalikan nama field yang diubah, untuk jejak audit tanpa
// ikut menyimpan nilainya.
func fieldNames(fields map[string]interface{}) []string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	return names
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
