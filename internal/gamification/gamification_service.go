package gamification

import (
	"errors"
	"log"
	"strings"
	"time"

	"mainyuk/internal/apperr"
	"mainyuk/internal/audit"
	"mainyuk/internal/community"
	"mainyuk/internal/period"
	"mainyuk/internal/ratelimit"
	"mainyuk/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type service struct {
	Repository
	ProfileEnsurer ProfileEnsurer
	UserService    user.Service
}

func NewService(repository Repository, profileEnsurer ProfileEnsurer, userService user.Service) Service {
	return &service{
		Repository:     repository,
		ProfileEnsurer: profileEnsurer,
		UserService:    userService,
	}
}

// OnProfileCompleted memberi reward "profil lengkap" satu kali seumur akun.
//
// Best-effort: dipanggil setelah profil diperbarui, dan kegagalannya tidak
// boleh menggagalkan penyimpanan profil. Kuncinya tidak memuat waktu atau
// referensi apa pun, sehingga mengubah-ubah field profil tidak bisa dipakai
// memanen XP berulang.
func (s *service) OnProfileCompleted(c *gin.Context, userID string) error {
	if userID == "" {
		return nil
	}

	u, err := s.UserService.Show(c, userID)
	if err != nil || u == nil {
		return err
	}

	completion := community.ProfileCompletion{
		Name:            u.Name,
		Phone:           u.Phone,
		ProvinceCode:    u.ProvinceCode,
		DistrictCode:    u.DistrictCode,
		SubDistrictCode: u.SubDistrictCode,
	}
	if !u.BirthDate.IsZero() {
		birthDate := u.BirthDate
		completion.BirthDate = &birthDate
	}
	if !community.IsProfileComplete(completion) {
		return nil
	}

	rule, err := s.Repository.XPRule(c, SourceProfileComplete)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Aturan dinonaktifkan admin: bukan kegagalan, hanya tanpa reward.
			return nil
		}
		return err
	}
	if rule.XP <= 0 {
		return nil
	}

	_, err = s.grant(c, &LedgerEntry{
		ID:         uuid.NewString(),
		UserID:     userID,
		Delta:      rule.XP,
		SourceType: SourceProfileComplete,
		DedupKey:   ProfileDedupKey(userID),
	})
	return err
}

// OnVerifiedCheckIn memberi reward check-in satu kali per akun per event.
//
// participantUserID harus peserta terverifikasi. Pemanggil sengaja tidak
// memakai fallback pembeli tiket: XP tidak boleh jatuh ke akun yang tidak
// menghadiri acara.
func (s *service) OnVerifiedCheckIn(c *gin.Context, participantUserID, eventID string) error {
	if participantUserID == "" || eventID == "" {
		return nil
	}

	rule, err := s.Repository.XPRule(c, SourceCheckIn)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if rule.XP <= 0 {
		return nil
	}

	refType := "event"
	_, err = s.grant(c, &LedgerEntry{
		ID:         uuid.NewString(),
		UserID:     participantUserID,
		Delta:      rule.XP,
		SourceType: SourceCheckIn,
		RefType:    &refType,
		RefID:      &eventID,
		DedupKey:   CheckInDedupKey(participantUserID, eventID),
	})
	return err
}

// GrantMissionReward memberi XP untuk klaim misi yang sudah disetujui.
// Besaran XP ditentukan pemanggil (snapshot reward misi saat approve) supaya
// mengubah definisi misi tidak mengubah reward klaim yang sudah berjalan.
func (s *service) GrantMissionReward(c *gin.Context, userID, claimID string, xp int) (bool, error) {
	if userID == "" || claimID == "" || xp <= 0 {
		return false, nil
	}
	refType := "mission_claim"
	return s.grant(c, &LedgerEntry{
		ID:         uuid.NewString(),
		UserID:     userID,
		Delta:      xp,
		SourceType: SourceMission,
		RefType:    &refType,
		RefID:      &claimID,
		DedupKey:   MissionDedupKey(userID, claimID),
	})
}

// GrantDonationReward memberi XP untuk donasi yang pembayarannya sudah
// terkonfirmasi.
//
// Mengembalikan XP yang benar-benar berlaku (0 bila aturan kosong atau batas
// bulanan sudah tercapai). Nilai itu disimpan pemanggil sebagai snapshot
// rewarded_xp pada donasi, sehingga pembalikan saat refund tetap tepat
// walaupun aturan XP berubah setelahnya.
//
// Berbeda dari jalur reward lain, penulisan ledger di sini tidak lewat grant:
// keputusannya bergantung pada isi ledger bulan berjalan, jadi hitung-lalu-
// sisip harus berada dalam satu transaksi. Baris profil tetap dipastikan ada
// lebih dulu di luar transaksi itu.
func (s *service) GrantDonationReward(c *gin.Context, userID, donationID string) (int, error) {
	if userID == "" || donationID == "" {
		return 0, nil
	}

	rule, err := s.Repository.XPRule(c, SourceDonation)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Aturan dinonaktifkan admin: bukan kegagalan, hanya tanpa reward.
			return 0, nil
		}
		return 0, err
	}
	if rule.XP <= 0 {
		return 0, nil
	}

	if s.ProfileEnsurer != nil {
		if _, err := s.ProfileEnsurer.EnsureProfile(c, userID); err != nil {
			return 0, err
		}
	}

	monthlyCap := 0
	if rule.MonthlyCap != nil {
		monthlyCap = *rule.MonthlyCap
	}
	from, to := MonthlyCapWindow(time.Now())

	refType := "donation"
	entry := &LedgerEntry{
		ID:         uuid.NewString(),
		UserID:     userID,
		SourceType: SourceDonation,
		RefType:    &refType,
		RefID:      &donationID,
		DedupKey:   DonationDedupKey(userID, donationID),
	}
	return s.Repository.GrantCappedOnce(c, entry, SourceDonation, "donation_reward:"+userID, from, to, rule.XP, monthlyCap)
}

// GrantShopOrderReward memberi XP untuk pesanan merchandise yang pembayarannya
// sudah terkonfirmasi.
//
// Bentuknya sengaja cermin dari GrantDonationReward — termasuk batas bulanan
// yang dihitung per akun dan per sumber — supaya dua jalur uang yang berbeda
// tidak memberi reward dengan aturan yang berbeda diam-diam. Kuota donasi dan
// kuota pesanan terpisah: membeli merchandise tidak menghabiskan kuota donasi.
func (s *service) GrantShopOrderReward(c *gin.Context, userID, orderID string) (int, error) {
	if userID == "" || orderID == "" {
		return 0, nil
	}

	rule, err := s.Repository.XPRule(c, SourceShopOrder)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// Aturan dinonaktifkan admin: bukan kegagalan, hanya tanpa reward.
			return 0, nil
		}
		return 0, err
	}
	if rule.XP <= 0 {
		return 0, nil
	}

	if s.ProfileEnsurer != nil {
		if _, err := s.ProfileEnsurer.EnsureProfile(c, userID); err != nil {
			return 0, err
		}
	}

	monthlyCap := 0
	if rule.MonthlyCap != nil {
		monthlyCap = *rule.MonthlyCap
	}
	from, to := MonthlyCapWindow(time.Now())

	refType := "shop_order"
	entry := &LedgerEntry{
		ID:         uuid.NewString(),
		UserID:     userID,
		SourceType: SourceShopOrder,
		RefType:    &refType,
		RefID:      &orderID,
		DedupKey:   ShopOrderDedupKey(userID, orderID),
	}
	return s.Repository.GrantCappedOnce(c, entry, SourceShopOrder, "shop_reward:"+userID, from, to, rule.XP, monthlyCap)
}

// ReverseShopOrderReward membalik XP pesanan merchandise yang di-refund.
//
// Idempoten lewat kunci dedup deterministik, jadi mengulang refund tidak
// menggandakan pembalikan. Entrinya bersumber shop_order_reversal — terpisah
// dari koreksi manual dan tidak ikut terhitung pada batas XP pesanan.
func (s *service) ReverseShopOrderReward(c *gin.Context, userID, orderID string, xp int, reason string) error {
	if userID == "" || orderID == "" || xp <= 0 {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return invalid("alasan pembalikan XP wajib diisi")
	}

	refType := "shop_order"
	entry := &LedgerEntry{
		ID:         uuid.NewString(),
		UserID:     userID,
		Delta:      -xp,
		SourceType: SourceShopOrderReversal,
		RefType:    &refType,
		RefID:      &orderID,
		DedupKey:   ShopOrderReversalDedupKey(userID, orderID),
		Note:       &reason,
	}
	if actor, ok := user.FromContext(c); ok {
		actorID := actor.ID
		entry.ActorUserID = &actorID
	}

	_, err := s.grant(c, entry)
	return err
}

// ReverseDonationReward membalik XP donasi yang di-refund.
//
// Idempoten lewat kunci dedup deterministik, jadi mengulang refund tidak
// menggandakan pembalikan. Entrinya bersumber donation_reversal — terpisah
// dari koreksi manual dan tidak ikut terhitung pada batas XP donasi.
func (s *service) ReverseDonationReward(c *gin.Context, userID, donationID string, xp int, reason string) error {
	if userID == "" || donationID == "" || xp <= 0 {
		return nil
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return invalid("alasan pembalikan XP wajib diisi")
	}

	refType := "donation"
	entry := &LedgerEntry{
		ID:         uuid.NewString(),
		UserID:     userID,
		Delta:      -xp,
		SourceType: SourceDonationReversal,
		RefType:    &refType,
		RefID:      &donationID,
		DedupKey:   DonationReversalDedupKey(userID, donationID),
		Note:       &reason,
	}
	if actor, ok := user.FromContext(c); ok {
		actorID := actor.ID
		entry.ActorUserID = &actorID
	}

	_, err := s.grant(c, entry)
	return err
}

// grant menulis satu entri ledger. Semua jalur pemberian XP melewati sini
// supaya aturan "satu kunci dedup = satu baris" hanya punya satu tempat.
func (s *service) grant(c *gin.Context, entry *LedgerEntry) (bool, error) {
	if entry.Delta == 0 {
		return false, invalid("delta XP tidak boleh nol")
	}

	// Pastikan baris profil ada lebih dulu: leaderboard JOIN ke profil, jadi
	// akun tanpa baris profil akan kehilangan XP-nya dari papan peringkat.
	if s.ProfileEnsurer != nil {
		if _, err := s.ProfileEnsurer.EnsureProfile(c, entry.UserID); err != nil {
			return false, err
		}
	}
	return s.Repository.GrantOnce(c, entry)
}

// Summary mengembalikan total XP, level berlaku, dan progres ke level berikutnya.
func (s *service) Summary(c *gin.Context) (*XPSummary, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, apperr.ErrUnauthorized
	}

	total, err := s.Repository.TotalXP(c, currentUser.ID)
	if err != nil {
		return nil, err
	}
	rules, err := s.Repository.LevelRules(c)
	if err != nil {
		return nil, err
	}

	views := ToLevelViews(rules)
	current := ResolveLevel(views, total)
	percent, remaining := ProgressToNext(views, total)

	summary := &XPSummary{
		TotalXP:         total,
		Level:           current.Level,
		LevelName:       current.Name,
		Badge:           current.Badge,
		ProgressPercent: percent,
		RemainingXP:     remaining,
	}
	if next, ok := NextLevel(views, total); ok {
		summary.NextLevel = &NextLevelInfo{
			Level: next.Level,
			Name:  next.Name,
			MinXP: next.MinXP,
		}
	}
	return summary, nil
}

// History mengembalikan riwayat ledger milik pemanggil, terbaru lebih dulu.
func (s *service) History(c *gin.Context, page, perPage int) ([]*LedgerEntry, bool, error) {
	currentUser, ok := user.FromContext(c)
	if !ok {
		return nil, false, apperr.ErrUnauthorized
	}
	page, perPage = NormalizePagination(page, perPage)

	// Ambil satu baris ekstra untuk mengetahui ada tidaknya halaman berikutnya
	// tanpa COUNT(*) atas seluruh ledger.
	entries, err := s.Repository.History(c, currentUser.ID, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(entries) > perPage
	if hasMore {
		entries = entries[:perPage]
	}
	return entries, hasMore, nil
}

// Adjust mencatat koreksi XP oleh admin. Ledger bersifat append-only, jadi
// koreksi selalu berupa baris baru beralasan, bukan perubahan baris lama.
func (s *service) Adjust(c *gin.Context, req *AdjustXP) (*LedgerEntry, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	if strings.TrimSpace(req.UserID) == "" {
		return nil, invalid("akun tujuan wajib diisi")
	}
	if req.Delta == 0 {
		return nil, invalid("delta XP tidak boleh nol")
	}
	if strings.TrimSpace(req.Reason) == "" {
		return nil, invalid("alasan koreksi wajib diisi")
	}

	// Tolak lebih awal bila akunnya tidak ada, supaya tidak ada baris ledger
	// yang menggantung pada user_id yang tidak dikenal.
	if _, err := s.UserService.Show(c, req.UserID); err != nil {
		return nil, ErrUserNotFound
	}

	actor := ""
	if currentUser, ok := user.FromContext(c); ok {
		if !ratelimit.XpAdjust.Allow(ratelimit.Key(c, currentUser.ID)) {
			return nil, ratelimit.ErrTooManyRequests
		}
		actor = currentUser.ID
	}

	reason := strings.TrimSpace(req.Reason)
	entry := &LedgerEntry{
		ID:         uuid.NewString(),
		UserID:     req.UserID,
		Delta:      req.Delta,
		SourceType: SourceAdjustment,
		DedupKey:   AdjustmentDedupKey(),
		Note:       &reason,
		CreatedAt:  time.Now(),
	}
	if actor != "" {
		entry.ActorUserID = &actor
	}

	// Koreksi tidak pernah di-dedup, jadi hasilnya selalu baris baru.
	inserted, err := s.grant(c, entry)
	if err != nil {
		return nil, err
	}

	// Jejak audit ditulis setelah ledgernya benar-benar tersimpan, sehingga
	// percobaan ulang tidak pernah menghasilkan baris audit tanpa baris ledger
	// yang mendasarinya. Yang dicatat hanya pengenal buram: tidak ada nama
	// maupun email, sama seperti isi ledger itu sendiri.
	if inserted {
		s.writeAudit(c, "xp.adjust", entityXPAdjustment, entry.ID, &reason, map[string]any{
			"user_id":     entry.UserID,
			"delta":       entry.Delta,
			"source_type": entry.SourceType,
		})
	}
	return entry, nil
}

// writeAudit mencatat satu perubahan yang hanya boleh dilakukan pengurus.
//
// Hanya dipanggil setelah penulisan terjaga yang mendahuluinya benar-benar
// berubah. Kegagalan menulis audit tidak membatalkan perubahan yang sudah
// terjadi — ia dilaporkan ke log, bukan dikembalikan sebagai galat.
func (s *service) writeAudit(c *gin.Context, action, entityType, entityID string, reason *string, detail map[string]any) {
	entry := audit.NewEntry(actorID(c), action, entityType, entityID, reason, detail)

	if err := s.Repository.WriteAudit(c, entry); err != nil {
		log.Printf("[audit] gagal menulis audit %s %s/%s: %v", action, entityType, entityID, err)
	}
}

// actorID mengambil identitas pemanggil dari konteks autentikasi.
func actorID(c *gin.Context) string {
	if currentUser, ok := user.FromContext(c); ok {
		return currentUser.ID
	}
	return ""
}

// AuditLogs menyusun satu halaman jejak audit modul ini.
//
// entityType yang dikirim pemanggil disaring terhadap cakupan modul lebih dulu
// supaya penyaring dari luar tidak dapat membuka baris milik modul lain.
func (s *service) AuditLogs(c *gin.Context, entityType, entityID string, page, perPage int) ([]*audit.Log, bool, error) {
	page, perPage = NormalizePagination(page, perPage)

	types := auditEntityTypes
	if entityType != "" {
		types = nil
		for _, candidate := range auditEntityTypes {
			if candidate == entityType {
				types = []string{candidate}
				break
			}
		}
	}

	entries, err := s.Repository.ListAudit(c, types, strings.TrimSpace(entityID), perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, false, err
	}

	hasMore := len(entries) > perPage
	if hasMore {
		entries = entries[:perPage]
	}
	return entries, hasMore, nil
}

func (s *service) LevelRules(c *gin.Context) ([]*LevelRule, error) {
	return s.Repository.LevelRules(c)
}

// UpdateLevelRules menimpa seluruh kurva level sekaligus.
//
// Himpunan level yang dikirim wajib sama dengan yang ada di database: baris
// yang tidak dikenal akan membuat UPDATE diam-diam tidak berefek, sehingga
// admin bisa mengira kurva tersimpan padahal tidak.
func (s *service) UpdateLevelRules(c *gin.Context, req *UpdateLevelRules) ([]*LevelRule, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	if err := ValidateLevelRules(req.Rules); err != nil {
		return nil, err
	}

	existing, err := s.Repository.LevelRules(c)
	if err != nil {
		return nil, err
	}
	known := map[int]bool{}
	for _, rule := range existing {
		known[rule.Level] = true
	}
	for _, rule := range req.Rules {
		if !known[rule.Level] {
			return nil, invalid("level %d tidak ada pada kurva saat ini", rule.Level)
		}
	}
	if len(req.Rules) != len(existing) {
		return nil, invalid("jumlah level tidak boleh berubah; kirim seluruh %d level", len(existing))
	}

	updated := make([]*LevelRule, 0, len(req.Rules))
	for _, r := range req.Rules {
		updated = append(updated, &LevelRule{
			Level:        r.Level,
			Name:         strings.TrimSpace(r.Name),
			MinXP:        r.MinXP,
			BadgeLabel:   r.BadgeLabel,
			BadgeIconURL: r.BadgeIconURL,
		})
	}
	if err := s.Repository.UpdateLevelRules(c, updated); err != nil {
		return nil, err
	}

	rules, err := s.Repository.LevelRules(c)
	if err != nil {
		return nil, err
	}

	// Kurva selalu ditimpa seluruhnya, jadi tidak ada satu id yang mewakilinya;
	// jumlah level dan ambang tertinggi dicatat supaya perubahan besar terlihat
	// dari jejaknya tanpa perlu membuka tabelnya.
	detail := map[string]any{"level_count": len(rules)}
	if len(rules) > 0 {
		detail["max_min_xp"] = rules[len(rules)-1].MinXP
	}
	s.writeAudit(c, "level_rule.update", entityLevelRule, "", nil, detail)

	return rules, nil
}

func (s *service) XPRules(c *gin.Context) ([]*XPRule, error) {
	return s.Repository.XPRules(c)
}

// UpdateXPRule mengubah besaran XP satu sumber. Admin tidak boleh menyentuh
// sumber misi lewat jalur ini — reward misi disimpan per misi.
func (s *service) UpdateXPRule(c *gin.Context, sourceType string, req *UpdateXPRule) (*XPRule, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	if req.XP <= 0 {
		return nil, invalid("besaran XP harus lebih dari nol")
	}
	if sourceType == SourceMission {
		return nil, invalid("reward misi diatur per misi, bukan di aturan XP")
	}
	if _, err := s.Repository.XPRule(c, sourceType); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrRuleNotFound
		}
		return nil, err
	}
	if err := s.Repository.UpdateXPRule(c, sourceType, req.XP, req.Description, req.MonthlyCap); err != nil {
		return nil, err
	}

	rule, err := s.Repository.XPRule(c, sourceType)
	if err != nil {
		return nil, err
	}
	s.writeAudit(c, "xp_rule.update", entityXPRule, sourceType, nil, map[string]any{
		"xp":          rule.XP,
		"monthly_cap": rule.MonthlyCap,
	})
	return rule, nil
}

// Leaderboard menyusun satu halaman papan peringkat.
//
// Identitas yang dikirim hanya yang bersifat publik. Level diresolusi per
// baris di Go karena aturan level adalah data yang dapat diubah admin.
func (s *service) Leaderboard(c *gin.Context, periodKey string, page, perPage int) (*LeaderboardPage, error) {
	if periodKey == "" {
		periodKey = PeriodWeekly
	}
	if !IsValidPeriod(periodKey) {
		return nil, invalid("periode leaderboard tidak dikenal: %s", periodKey)
	}
	page, perPage = NormalizePagination(page, perPage)

	from, to := PeriodRange(periodKey, time.Now(), period.JakartaLocation)

	rows, err := s.Repository.Leaderboard(c, from, to, perPage+1, (page-1)*perPage)
	if err != nil {
		return nil, err
	}
	hasMore := len(rows) > perPage
	if hasMore {
		rows = rows[:perPage]
	}

	rules, err := s.Repository.LevelRules(c)
	if err != nil {
		return nil, err
	}
	views := ToLevelViews(rules)

	entries := make([]*LeaderboardEntry, 0, len(rows))
	for i, row := range rows {
		level := ResolveLevel(views, row.NetXP)
		entry := &LeaderboardEntry{
			Rank:      (page-1)*perPage + i + 1,
			PublicID:  row.PublicID,
			Alias:     displayAlias(row.Alias),
			AvatarURL: row.AvatarURL,
			Level:     level.Level,
			LevelName: level.Name,
			XP:        row.NetXP,
		}
		// Badge ikut disembunyikan bila pemiliknya mematikan tampilan badge.
		if row.ShowBadges {
			entry.Badge = level.Badge
		}
		entries = append(entries, entry)
	}

	return &LeaderboardPage{
		Period:  periodKey,
		Page:    page,
		PerPage: perPage,
		HasMore: hasMore,
		Entries: entries,
	}, nil
}

// displayAlias memberi nama tampilan netral untuk akun yang belum memilih alias.
func displayAlias(alias *string) string {
	if alias != nil && strings.TrimSpace(*alias) != "" {
		return *alias
	}
	return "Anggota"
}
