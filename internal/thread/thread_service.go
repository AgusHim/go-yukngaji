package thread

import (
	"errors"
	"log"
	"strings"
	"time"

	"mainyuk/internal/apperr"
	"mainyuk/internal/audit"
	"mainyuk/internal/community"
	"mainyuk/internal/gamification"
	"mainyuk/internal/ratelimit"
	"mainyuk/internal/user"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// feedExcerptLen adalah panjang ringkasan yang dikirim daftar feed. Halaman
// detail mengirim badan penuh; daftar tidak perlu mengangkut seluruh kiriman.
const feedExcerptLen = 220

// noteMaxLen adalah batas panjang catatan pada laporan.
const noteMaxLen = 500

// Action audit paket ini. Semuanya tindakan moderator atau pencabutan izin,
// bukan tindakan anggota biasa: reaksi dan komentar tidak berjejak di audit.
const (
	actionThreadHide        = "thread.hide"
	actionThreadRestore     = "thread.restore"
	actionThreadDelete      = "thread.delete"
	actionCommentHide       = "thread_comment.hide"
	actionCommentRestore    = "thread_comment.restore"
	actionCommentDelete     = "thread_comment.delete"
	actionReportAction      = "report.action"
	actionReportDismiss     = "report.dismiss"
	actionAccountRestrict   = "account.restrict"
	actionAccountUnrestrict = "account.unrestrict"
	actionShareRevoke       = "share.revoke"
)

type service struct {
	Repository
	authors AuthorResolver
}

func NewService(repository Repository, authors AuthorResolver) Service {
	return &service{Repository: repository, authors: authors}
}

// ---------------------------------------------------------------------------
// Penjaga jalur tulis
// ---------------------------------------------------------------------------

func currentUserID(c *gin.Context) string {
	if u, ok := user.FromContext(c); ok {
		return u.ID
	}
	return ""
}

// guardWrite menegakkan tiga hal sebelum setiap aksi tulis: identitas
// terverifikasi, batas laju, dan akun tidak sedang dibatasi.
//
// Urutannya disengaja. Pembatas laju diperiksa sebelum menyentuh database,
// supaya akun terblokir tidak bisa memakai penolakan blokir sebagai cara
// mengukur sesuatu; dan pemeriksaan blokir terjadi sebelum apa pun ditulis,
// supaya akun yang dibatasi tidak bisa menitipkan satu baris pun — termasuk
// lewat auto-post, yang memakai jalur yang sama.
//
// limiter boleh nil untuk aksi yang tidak dibatasi laju (hapus, batal reaksi).
func (s *service) guardWrite(c *gin.Context, limiter *ratelimit.Limiter) (string, *community.Profile, error) {
	userID := currentUserID(c)
	if userID == "" {
		return "", nil, apperr.ErrUnauthorized
	}
	if limiter != nil && !limiter.Allow(ratelimit.Key(c, userID)) {
		return "", nil, ratelimit.ErrTooManyRequests
	}

	profile, err := s.authors.EnsureProfile(c, userID)
	if err != nil {
		return "", nil, err
	}
	if IsBlockedAccount(profile) {
		return "", nil, ErrAccountRestricted
	}
	return userID, profile, nil
}

// pageWindow merapikan page/per_page dan mengembalikan lebar halaman beserta
// offsetnya. Pemanggil menambah satu ke lebarnya untuk mendeteksi has_more.
func pageWindow(page, perPage int) (int, int) {
	page, perPage = gamification.NormalizePagination(page, perPage)
	return perPage, (page - 1) * perPage
}

func notFoundAs(err error, sentinel error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return sentinel
	}
	return err
}

// ---------------------------------------------------------------------------
// Bentuk tampilan
// ---------------------------------------------------------------------------

// toThreadView adalah satu-satunya tempat ThreadView disusun, supaya tidak ada
// jalur yang lupa memakai ToThreadAuthor.
func (s *service) toThreadView(t *Thread, profile *community.Profile, viewerID string, reacted bool, excerpt bool) *ThreadView {
	view := &ThreadView{
		PublicID:      t.PublicID,
		Title:         t.Title,
		Author:        ToThreadAuthor(profile),
		Status:        t.Status,
		SourceType:    t.SourceType,
		CommentCount:  t.CommentCount,
		ReactionCount: t.ReactionCount,
		Reacted:       reacted,
		IsMine:        viewerID != "" && viewerID == t.UserID,
		CreatedAt:     t.CreatedAt,
	}
	if excerpt {
		view.Excerpt = Excerpt(t.Body, feedExcerptLen)
	} else {
		view.Body = t.Body
	}
	return view
}

func (s *service) toCommentView(cm *ThreadComment, profile *community.Profile, viewerID string) *ThreadCommentView {
	return &ThreadCommentView{
		PublicID:  cm.PublicID,
		Body:      cm.Body,
		Author:    ToThreadAuthor(profile),
		Status:    cm.Status,
		IsMine:    viewerID != "" && viewerID == cm.UserID,
		CreatedAt: cm.CreatedAt,
	}
}

// profilesFor mengambil profil seluruh penulis dalam satu query. Akun yang
// belum punya profil tidak muncul di peta hasil dan tampil sebagai anonim.
func (s *service) profilesFor(c *gin.Context, userIDs []string) (map[string]*community.Profile, error) {
	return s.authors.ProfilesOf(c, userIDs)
}

func collectUserIDs(threads []*Thread) []string {
	seen := make(map[string]bool, len(threads))
	ids := make([]string, 0, len(threads))
	for _, t := range threads {
		if !seen[t.UserID] {
			seen[t.UserID] = true
			ids = append(ids, t.UserID)
		}
	}
	return ids
}

// ---------------------------------------------------------------------------
// Feed
// ---------------------------------------------------------------------------

func (s *service) ListThreads(c *gin.Context, sort string, page, perPage int) ([]*ThreadView, bool, error) {
	if sort != SortPopuler {
		sort = SortTerbaru
	}
	perPage, offset := pageWindow(page, perPage)

	rows, err := s.Repository.ListThreads(c, sort, perPage+1, offset)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > perPage
	if hasMore {
		rows = rows[:perPage]
	}

	profiles, err := s.profilesFor(c, collectUserIDs(rows))
	if err != nil {
		return nil, false, err
	}

	viewerID := currentUserID(c)
	reacted, err := s.Repository.ReactionsByUser(c, viewerID, threadIDs(rows))
	if err != nil {
		return nil, false, err
	}

	views := make([]*ThreadView, 0, len(rows))
	for _, t := range rows {
		views = append(views, s.toThreadView(t, profiles[t.UserID], viewerID, reacted[t.ID], true))
	}
	return views, hasMore, nil
}

func threadIDs(threads []*Thread) []string {
	ids := make([]string, 0, len(threads))
	for _, t := range threads {
		ids = append(ids, t.ID)
	}
	return ids
}

func (s *service) ShowThread(c *gin.Context, publicID string) (*ThreadView, error) {
	t, err := s.Repository.FindThreadByPublicID(c, publicID)
	if err != nil {
		return nil, notFoundAs(err, ErrThreadNotFound)
	}
	// Thread yang disembunyikan atau dihapus tidak pernah muncul di jalur
	// publik, dan statusnya tidak dibedakan dari "tidak ada".
	if t.Status != StatusPublished {
		return nil, ErrThreadNotFound
	}

	profiles, err := s.profilesFor(c, []string{t.UserID})
	if err != nil {
		return nil, err
	}
	profile := profiles[t.UserID]
	if IsBlockedAccount(profile) {
		return nil, ErrThreadNotFound
	}

	viewerID := currentUserID(c)
	reacted, err := s.Repository.ReactionsByUser(c, viewerID, []string{t.ID})
	if err != nil {
		return nil, err
	}
	return s.toThreadView(t, profile, viewerID, reacted[t.ID], false), nil
}

func (s *service) CreateThread(c *gin.Context, req *CreateThread) (*ThreadView, error) {
	userID, profile, err := s.guardWrite(c, ratelimit.Thread)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	// Disimpan dalam bentuk yang sudah dirapikan, supaya yang diperiksa
	// validasi sama persis dengan yang dilihat constraint database.
	title := NormalizeContent(req.Title)
	body := NormalizeContent(req.Body)
	if err := CheckThreadContent(title, body); err != nil {
		return nil, err
	}

	now := time.Now()
	t := &Thread{
		ID:        uuid.NewString(),
		PublicID:  community.NewPublicID(),
		UserID:    userID,
		Title:     title,
		Body:      body,
		Status:    StatusPublished,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Repository.CreateThread(c, t); err != nil {
		return nil, err
	}
	return s.toThreadView(t, profile, userID, false, false), nil
}

// DeleteThread menghapus thread milik pemanggil.
//
// Syarat kepemilikan ada di dalam query, dan kegagalannya dilaporkan sebagai
// "tidak ditemukan" — bukan "bukan milikmu" — supaya keberadaan thread orang
// lain tidak bisa disimpulkan dari perbedaan pesan.
func (s *service) DeleteThread(c *gin.Context, publicID string) error {
	userID, _, err := s.guardWrite(c, nil)
	if err != nil {
		return err
	}
	deleted, err := s.Repository.SoftDeleteOwnThread(c, publicID, userID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrThreadNotFound
	}
	return nil
}

// ---------------------------------------------------------------------------
// Komentar
// ---------------------------------------------------------------------------

func (s *service) ListComments(c *gin.Context, threadPublicID string, page, perPage int) ([]*ThreadCommentView, bool, error) {
	t, err := s.findOpenThread(c, threadPublicID)
	if err != nil {
		return nil, false, err
	}
	perPage, offset := pageWindow(page, perPage)

	rows, err := s.Repository.ListComments(c, t.ID, perPage+1, offset)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > perPage
	if hasMore {
		rows = rows[:perPage]
	}

	authorIDs := make([]string, 0, len(rows))
	for _, cm := range rows {
		authorIDs = append(authorIDs, cm.UserID)
	}
	profiles, err := s.profilesFor(c, authorIDs)
	if err != nil {
		return nil, false, err
	}

	viewerID := currentUserID(c)
	views := make([]*ThreadCommentView, 0, len(rows))
	for _, cm := range rows {
		views = append(views, s.toCommentView(cm, profiles[cm.UserID], viewerID))
	}
	return views, hasMore, nil
}

func (s *service) CreateComment(c *gin.Context, threadPublicID string, req *CreateThreadComment) (*ThreadCommentView, error) {
	userID, profile, err := s.guardWrite(c, ratelimit.ThreadComment)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	t, err := s.findOpenThread(c, threadPublicID)
	if err != nil {
		return nil, err
	}

	body := NormalizeContent(req.Body)
	if err := CheckCommentContent(body); err != nil {
		return nil, err
	}

	now := time.Now()
	cm := &ThreadComment{
		ID:        uuid.NewString(),
		PublicID:  community.NewPublicID(),
		ThreadID:  t.ID,
		UserID:    userID,
		Body:      body,
		Status:    StatusPublished,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.Repository.CreateComment(c, cm); err != nil {
		return nil, err
	}
	if err := s.Repository.AdjustThreadCounters(c, t.ID, 1, 0); err != nil {
		// Komentarnya sudah tersimpan; counter yang meleset tidak boleh
		// membatalkan komentar yang sah. Dilaporkan supaya bisa diperbaiki.
		log.Printf("[thread] gagal menambah comment_count thread %s: %v", t.ID, err)
	}
	return s.toCommentView(cm, profile, userID), nil
}

func (s *service) DeleteComment(c *gin.Context, threadPublicID, commentPublicID string) error {
	userID, _, err := s.guardWrite(c, nil)
	if err != nil {
		return err
	}
	t, err := s.findOpenThread(c, threadPublicID)
	if err != nil {
		return err
	}

	cm, err := s.Repository.FindCommentByPublicID(c, commentPublicID)
	if err != nil {
		return notFoundAs(err, ErrCommentNotFound)
	}
	// Komentar yang bukan milik thread ini diperlakukan sebagai tidak ada,
	// supaya id komentar tidak bisa dipakai menurunkan counter thread lain.
	if cm.ThreadID != t.ID {
		return ErrCommentNotFound
	}

	deleted, err := s.Repository.SoftDeleteOwnComment(c, commentPublicID, userID)
	if err != nil {
		return err
	}
	if !deleted {
		return ErrCommentNotFound
	}
	if err := s.Repository.AdjustThreadCounters(c, t.ID, -1, 0); err != nil {
		log.Printf("[thread] gagal mengurangi comment_count thread %s: %v", t.ID, err)
	}
	return nil
}

// findOpenThread mengambil thread yang masih boleh dikomentari. Thread yang
// disembunyikan atau dihapus diperlakukan sebagai tidak ada.
func (s *service) findOpenThread(c *gin.Context, publicID string) (*Thread, error) {
	t, err := s.Repository.FindThreadByPublicID(c, publicID)
	if err != nil {
		return nil, notFoundAs(err, ErrThreadNotFound)
	}
	if t.Status != StatusPublished {
		return nil, ErrThreadNotFound
	}
	return t, nil
}

// ---------------------------------------------------------------------------
// Reaksi
// ---------------------------------------------------------------------------

// React bersifat idempoten: bereaksi dua kali tidak menambah counter dua kali,
// dan itu yang dijaga index unik (thread_id, user_id) di database.
func (s *service) React(c *gin.Context, threadPublicID string) error {
	userID, _, err := s.guardWrite(c, ratelimit.ThreadReaction)
	if err != nil {
		return err
	}
	t, err := s.findOpenThread(c, threadPublicID)
	if err != nil {
		return err
	}

	created, err := s.Repository.AddReaction(c, &ThreadReaction{
		ID:        uuid.NewString(),
		ThreadID:  t.ID,
		UserID:    userID,
		Kind:      "like",
		CreatedAt: time.Now(),
	})
	if err != nil {
		return err
	}
	if created {
		if err := s.Repository.AdjustThreadCounters(c, t.ID, 0, 1); err != nil {
			log.Printf("[thread] gagal menambah reaction_count thread %s: %v", t.ID, err)
		}
	}
	return nil
}

// Unreact juga idempoten: membatalkan reaksi yang tidak ada tidak menurunkan
// counter, karena barisnya memang tidak terhapus.
func (s *service) Unreact(c *gin.Context, threadPublicID string) error {
	userID, _, err := s.guardWrite(c, nil)
	if err != nil {
		return err
	}
	t, err := s.findOpenThread(c, threadPublicID)
	if err != nil {
		return err
	}

	removed, err := s.Repository.RemoveReaction(c, t.ID, userID)
	if err != nil {
		return err
	}
	if removed {
		if err := s.Repository.AdjustThreadCounters(c, t.ID, 0, -1); err != nil {
			log.Printf("[thread] gagal mengurangi reaction_count thread %s: %v", t.ID, err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Laporan
// ---------------------------------------------------------------------------

// Report mencatat laporan anggota.
//
// Laporan kedua atas target yang sama oleh akun yang sama bukan galat: hasilnya
// sudah tercapai. Pelanggaran index uniknya diperlakukan sebagai sukses
// idempoten, bukan sebagai kegagalan yang perlu ditangani pemanggil.
func (s *service) Report(c *gin.Context, req *CreateReport) error {
	userID, _, err := s.guardWrite(c, ratelimit.ThreadReport)
	if err != nil {
		return err
	}
	if req == nil {
		return apperr.ErrInvalidRequest
	}
	if !isKnownTarget(req.TargetType) {
		return invalid("Jenis target laporan tidak dikenal.")
	}
	if !isKnownReason(req.Reason) {
		return invalid("Alasan laporan tidak dikenal.")
	}

	// Target dikirim sebagai public_id; id internalnya diselesaikan di sini
	// dan tidak pernah keluar lagi.
	targetID, err := s.resolveTargetID(c, req.TargetType, req.TargetID)
	if err != nil {
		return err
	}

	note := normalizeNote(req.Note)
	now := time.Now()
	reporter := userID
	_, err = s.Repository.CreateReport(c, &Report{
		ID:             uuid.NewString(),
		PublicID:       community.NewPublicID(),
		ReporterUserID: &reporter,
		TargetType:     req.TargetType,
		TargetID:       targetID,
		Reason:         req.Reason,
		Note:           note,
		Status:         ReportOpen,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	return err
}

func (s *service) resolveTargetID(c *gin.Context, targetType, publicID string) (string, error) {
	switch targetType {
	case TargetThread:
		t, err := s.Repository.FindThreadByPublicID(c, publicID)
		if err != nil {
			return "", notFoundAs(err, ErrThreadNotFound)
		}
		return t.ID, nil
	case TargetThreadComment:
		cm, err := s.Repository.FindCommentByPublicID(c, publicID)
		if err != nil {
			return "", notFoundAs(err, ErrCommentNotFound)
		}
		return cm.ID, nil
	}
	return "", invalid("Jenis target laporan tidak dikenal.")
}

func normalizeNote(note *string) *string {
	if note == nil {
		return nil
	}
	trimmed := strings.TrimSpace(NormalizeContent(*note))
	if trimmed == "" {
		return nil
	}
	if len([]rune(trimmed)) > noteMaxLen {
		trimmed = string([]rune(trimmed)[:noteMaxLen])
	}
	return &trimmed
}

// ---------------------------------------------------------------------------
// Preferensi berbagi aktivitas
// ---------------------------------------------------------------------------

// MySharePrefs selalu mengembalikan ketiga nilai, walaupun barisnya belum ada.
// Ketiadaan baris berarti semua nyala, dan itu dinyatakan di sini sekali,
// supaya tidak ada pemanggil yang menebak-nebak.
func (s *service) MySharePrefs(c *gin.Context) (*SharePrefs, error) {
	userID := currentUserID(c)
	if userID == "" {
		return nil, apperr.ErrUnauthorized
	}

	prefs, err := s.Repository.FindSharePrefs(c, userID)
	if err != nil {
		return nil, err
	}
	if prefs == nil {
		return &SharePrefs{
			UserID:                 userID,
			ShareEventRegistration: true,
			ShareDonation:          true,
			ShareMission:           true,
		}, nil
	}
	return prefs, nil
}

// UpdateSharePrefs menyimpan preferensi, lalu mencabut auto-post untuk setiap
// aktivitas yang baru saja dimatikan.
//
// Pencabutan dilakukan di sini, bukan dihitung ulang saat feed dibaca, karena
// tidak ada scheduler di sistem ini. Efeknya: mematikan preferensi langsung
// menghapus kiriman lamanya, dan menyalakannya kembali tidak menghidupkannya —
// index unik sumbernya sengaja tidak peduli deleted_at.
func (s *service) UpdateSharePrefs(c *gin.Context, req *UpdateSharePrefs) (*SharePrefs, error) {
	userID, _, err := s.guardWrite(c, nil)
	if err != nil {
		return nil, err
	}
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}

	prefs, err := s.MySharePrefs(c)
	if err != nil {
		return nil, err
	}

	before := map[string]bool{
		SourceEventRegistration: prefs.ShareEventRegistration,
		SourceDonation:          prefs.ShareDonation,
		SourceMission:           prefs.ShareMission,
	}
	if req.ShareEventRegistration != nil {
		prefs.ShareEventRegistration = *req.ShareEventRegistration
	}
	if req.ShareDonation != nil {
		prefs.ShareDonation = *req.ShareDonation
	}
	if req.ShareMission != nil {
		prefs.ShareMission = *req.ShareMission
	}
	after := map[string]bool{
		SourceEventRegistration: prefs.ShareEventRegistration,
		SourceDonation:          prefs.ShareDonation,
		SourceMission:           prefs.ShareMission,
	}

	now := time.Now()
	prefs.UserID = userID
	if prefs.CreatedAt.IsZero() {
		prefs.CreatedAt = now
	}
	prefs.UpdatedAt = now
	if err := s.Repository.UpsertSharePrefs(c, prefs); err != nil {
		return nil, err
	}

	for _, activity := range []string{SourceEventRegistration, SourceDonation, SourceMission} {
		if !before[activity] || after[activity] {
			continue
		}
		revoked, err := s.Repository.SoftDeleteThreadsBySource(c, userID, activity)
		if err != nil {
			// Preferensinya sudah tersimpan; kegagalan mencabut kiriman lama
			// dilaporkan, tidak membatalkan pilihan anggota.
			log.Printf("[thread] gagal mencabut auto-post %s milik %s: %v", activity, userID, err)
			continue
		}
		reason := "preferensi berbagi dimatikan"
		s.writeAudit(c, actionShareRevoke, "thread_share_prefs", userID, &reason, map[string]any{
			"activity": activity,
			"revoked":  revoked,
		})
	}
	return prefs, nil
}

// ---------------------------------------------------------------------------
// Auto-post
// ---------------------------------------------------------------------------

func (s *service) PostEventRegistration(c *gin.Context, userID, orderID, eventTitle string) error {
	return s.autoPost(c, userID, SourceEventRegistration, orderID, eventTitle, false)
}

func (s *service) PostDonation(c *gin.Context, userID, donationID, campaignTitle string, isAnonymous bool) error {
	return s.autoPost(c, userID, SourceDonation, donationID, campaignTitle, isAnonymous)
}

func (s *service) PostMissionCompletion(c *gin.Context, userID, claimID, missionTitle string) error {
	return s.autoPost(c, userID, SourceMission, claimID, missionTitle, false)
}

// autoPost menitipkan satu aktivitas ke feed.
//
// Tidak ada pemeriksaan identitas terverifikasi di sini: pemanggilnya adalah
// modul lain yang sudah memutuskan bahwa aksinya sah, dan konteks permintaan
// yang sama sudah membawa identitasnya. Yang tetap diperiksa adalah blokir,
// privasi, anonimitas, dan preferensi — lewat satu fungsi murni.
func (s *service) autoPost(c *gin.Context, userID, activity, sourceRef, sourceTitle string, isAnonymous bool) error {
	if userID == "" {
		return apperr.ErrUnauthorized
	}

	profile, err := s.authors.EnsureProfile(c, userID)
	if err != nil {
		return err
	}
	prefs, err := s.Repository.FindSharePrefs(c, userID)
	if err != nil {
		return err
	}

	if !ShouldAutoPost(AutoPostContext{
		Activity:    activity,
		Prefs:       prefs,
		Profile:     profile,
		IsAnonymous: isAnonymous,
		SourceRef:   sourceRef,
		Title:       sourceTitle,
	}) {
		return nil
	}

	title := AutoPostTitle(activity, sourceTitle)
	body := AutoPostBody(activity, sourceTitle)
	if title == "" || body == "" {
		return nil
	}
	// Judul sumber datang dari modul lain, jadi ia bisa saja memuat kata
	// terlarang atau tautan. Kalau begitu, aktivitasnya tidak dibagikan —
	// bukan gagal, hanya tidak diterbitkan.
	if err := CheckThreadContent(title, body); err != nil {
		return err
	}

	now := time.Now()
	sourceType := activity
	ref := sourceRef
	_, err = s.Repository.CreateThreadIfAbsent(c, &Thread{
		ID:         uuid.NewString(),
		PublicID:   community.NewPublicID(),
		UserID:     userID,
		Title:      title,
		Body:       body,
		Status:     StatusPublished,
		SourceType: &sourceType,
		SourceRef:  &ref,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	return err
}

func (s *service) RemoveEventRegistration(c *gin.Context, orderID string) error {
	return s.removeBySource(c, SourceEventRegistration, orderID)
}

func (s *service) RemoveDonation(c *gin.Context, donationID string) error {
	return s.removeBySource(c, SourceDonation, donationID)
}

func (s *service) RemoveMissionCompletion(c *gin.Context, claimID string) error {
	return s.removeBySource(c, SourceMission, claimID)
}

// removeBySource mencabut auto-post milik satu sumber.
//
// Tidak ada yang perlu dicatat ke audit di sini: yang diaudit adalah peristiwa
// sumbernya (refund donasi, penolakan klaim, pembatalan order) dan itu sudah
// punya jejaknya sendiri. Mencatatnya lagi di sini akan menggandakan satu
// keputusan menjadi dua baris.
func (s *service) removeBySource(c *gin.Context, sourceType, sourceRef string) error {
	if sourceRef == "" {
		return nil
	}
	_, err := s.Repository.SoftDeleteThreadBySource(c, sourceType, sourceRef)
	return err
}

// ---------------------------------------------------------------------------
// Moderasi
// ---------------------------------------------------------------------------

func (s *service) ListReports(c *gin.Context, status string, page, perPage int) ([]*ModerationReportView, bool, error) {
	if status != "" && !IsKnownReportStatus(status) {
		return nil, false, invalid("Status laporan tidak dikenal.")
	}
	perPage, offset := pageWindow(page, perPage)

	rows, err := s.Repository.ListReports(c, status, perPage+1, offset)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > perPage
	if hasMore {
		rows = rows[:perPage]
	}

	views := make([]*ModerationReportView, 0, len(rows))
	for _, report := range rows {
		target, author, err := s.describeTarget(c, report)
		if err != nil {
			return nil, false, err
		}
		views = append(views, &ModerationReportView{Report: report, Target: target, Author: author})
	}
	return views, hasMore, nil
}

// describeTarget merangkum konten yang dilaporkan, dan membuka identitas
// penulisnya lewat AdminIdentity — inilah "pengaitan alias ke akun asli hanya
// untuk petugas berizin". Id akun, email, dan telepon tetap tidak ikut.
//
// Target yang sudah terhapus tetap ditampilkan sebagai baris tersendiri: laporan
// yang menunjuk ke konten yang sudah hilang tetap harus bisa diputuskan.
func (s *service) describeTarget(c *gin.Context, report *Report) (*ReportTargetView, *ThreadAuthor, error) {
	switch report.TargetType {
	case TargetThread:
		t, err := s.Repository.FindThreadByID(c, report.TargetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &ReportTargetView{Type: TargetThread, Status: StatusDeleted}, nil, nil
			}
			return nil, nil, err
		}
		author, err := s.moderationAuthor(c, t.UserID)
		if err != nil {
			return nil, nil, err
		}
		return &ReportTargetView{
			Type:     TargetThread,
			PublicID: t.PublicID,
			Title:    t.Title,
			Excerpt:  Excerpt(t.Body, feedExcerptLen),
			Status:   t.Status,
		}, author, nil

	case TargetThreadComment:
		cm, err := s.Repository.FindCommentByID(c, report.TargetID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return &ReportTargetView{Type: TargetThreadComment, Status: StatusDeleted}, nil, nil
			}
			return nil, nil, err
		}
		author, err := s.moderationAuthor(c, cm.UserID)
		if err != nil {
			return nil, nil, err
		}
		return &ReportTargetView{
			Type:     TargetThreadComment,
			PublicID: cm.PublicID,
			Excerpt:  Excerpt(cm.Body, feedExcerptLen),
			Status:   cm.Status,
		}, author, nil
	}

	return &ReportTargetView{Type: report.TargetType}, nil, nil
}

func (s *service) moderationAuthor(c *gin.Context, userID string) (*ThreadAuthor, error) {
	identity, err := s.authors.AdminIdentity(c, userID)
	if err != nil {
		if errors.Is(err, community.ErrProfileNotFound) {
			return ToModerationAuthor(nil), nil
		}
		return nil, err
	}
	return ToModerationAuthor(identity), nil
}

func (s *service) ActionReport(c *gin.Context, id string, req *DecideReport) (*Report, error) {
	reason, err := s.decisionReason(req)
	if err != nil {
		return nil, err
	}

	report, err := s.Repository.FindReportByID(c, id)
	if err != nil {
		return nil, notFoundAs(err, ErrReportNotFound)
	}

	if req.HideTarget {
		if err := s.hideTarget(c, report, reason); err != nil {
			return nil, err
		}
	}

	if err := s.transitionReport(c, report, ReportActioned, actionReportAction, reason); err != nil {
		return nil, err
	}
	return s.Repository.FindReportByID(c, id)
}

func (s *service) DismissReport(c *gin.Context, id string, req *DecideReport) (*Report, error) {
	reason, err := s.decisionReason(req)
	if err != nil {
		return nil, err
	}

	report, err := s.Repository.FindReportByID(c, id)
	if err != nil {
		return nil, notFoundAs(err, ErrReportNotFound)
	}
	if err := s.transitionReport(c, report, ReportDismissed, actionReportDismiss, reason); err != nil {
		return nil, err
	}
	return s.Repository.FindReportByID(c, id)
}

// transitionReport memindahkan laporan dan menulis jejaknya.
//
// RowsAffected = 0 berarti laporan sudah diputuskan sebelumnya. Itu diperlakukan
// sebagai sukses idempoten: hasil yang diminta sudah tercapai, dan tidak ada
// baris audit kedua untuk satu keputusan yang sama.
func (s *service) transitionReport(c *gin.Context, report *Report, to, action string, reason *string) error {
	moved, err := s.Repository.TransitionReport(c, report.ID, ReportOpen, to, map[string]any{
		"handled_by":      currentUserID(c),
		"handled_at":      time.Now(),
		"decision_reason": reason,
	})
	if err != nil {
		return err
	}
	if !moved {
		return nil
	}
	s.writeAudit(c, action, "report", report.ID, reason, map[string]any{
		"target_type": report.TargetType,
		"target_id":   report.TargetID,
		"status":      to,
	})
	return nil
}

func (s *service) hideTarget(c *gin.Context, report *Report, reason *string) error {
	switch report.TargetType {
	case TargetThread:
		_, err := s.ModerateThread(c, report.TargetID, "hide", &DecideContent{Reason: derefString(reason)})
		return err
	case TargetThreadComment:
		_, err := s.ModerateComment(c, report.TargetID, "hide", &DecideContent{Reason: derefString(reason)})
		return err
	}
	return nil
}

// ListThreadsForReview menampilkan konten untuk pemeriksaan proaktif, tanpa
// menunggu laporan. Penulis yang dibatasi tidak disembunyikan di sini —
// justru merekalah yang perlu diperiksa.
func (s *service) ListThreadsForReview(c *gin.Context, status string, page, perPage int) ([]*ThreadView, bool, error) {
	if status != "" && !IsKnownContentStatus(status) {
		return nil, false, invalid("Status konten tidak dikenal.")
	}
	perPage, offset := pageWindow(page, perPage)

	rows, err := s.Repository.ListThreadsForReview(c, status, perPage+1, offset)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > perPage
	if hasMore {
		rows = rows[:perPage]
	}

	profiles, err := s.profilesFor(c, collectUserIDs(rows))
	if err != nil {
		return nil, false, err
	}

	views := make([]*ThreadView, 0, len(rows))
	for _, t := range rows {
		view := s.toThreadView(t, profiles[t.UserID], "", false, true)
		// Antrean moderasi memakai aturan penamaan petugas: alias tetap
		// terlihat walaupun profil penulisnya privat atau terblokir.
		author, err := s.moderationAuthor(c, t.UserID)
		if err != nil {
			return nil, false, err
		}
		view.Author = author
		views = append(views, view)
	}
	return views, hasMore, nil
}

// ModerateThread menjalankan satu tindakan moderator atas thread.
//
// action: "hide", "restore", atau "delete". Alasan wajib untuk hide dan delete;
// restore boleh tanpa alasan karena ia mengembalikan keadaan, bukan membatasi.
func (s *service) ModerateThread(c *gin.Context, id, action string, req *DecideContent) (*Thread, error) {
	reason, err := s.decisionReasonForAction(action, req)
	if err != nil {
		return nil, err
	}

	t, err := s.Repository.FindThreadByID(c, id)
	if err != nil {
		return nil, notFoundAs(err, ErrThreadNotFound)
	}

	now := time.Now()
	var moved bool
	var auditAction string

	switch action {
	case "hide":
		if !CanTransitionThread(t.Status, StatusHidden) {
			return t, nil
		}
		moved, err = s.Repository.TransitionThread(c, id, t.Status, StatusHidden, map[string]any{
			"hidden_by":       currentUserID(c),
			"hidden_at":       now,
			"decision_reason": reason,
		})
		auditAction = actionThreadHide

	case "restore":
		if !CanTransitionThread(t.Status, StatusPublished) {
			return t, nil
		}
		moved, err = s.Repository.TransitionThread(c, id, t.Status, StatusPublished, map[string]any{
			"hidden_by":       nil,
			"hidden_at":       nil,
			"decision_reason": reason,
		})
		auditAction = actionThreadRestore

	case "delete":
		fields := map[string]any{
			"deleted_at":      now,
			"decision_reason": reason,
		}
		// Hapus bisa berangkat dari dua status, dan UPDATE bersyarat hanya
		// menerima satu. Dua percobaan berurutan aman: keduanya menuju status
		// yang sama, dan yang kedua tidak berpengaruh bila yang pertama berhasil.
		for _, from := range []string{StatusPublished, StatusHidden} {
			if !CanTransitionThread(t.Status, StatusDeleted) {
				break
			}
			moved, err = s.Repository.TransitionThread(c, id, from, StatusDeleted, fields)
			if err != nil || moved {
				break
			}
		}
		auditAction = actionThreadDelete

	default:
		return nil, invalid("Tindakan moderasi tidak dikenal: %s.", action)
	}

	if err != nil {
		return nil, err
	}
	if moved {
		s.writeAudit(c, auditAction, "thread", id, reason, map[string]any{
			"from": t.Status,
			"to":   targetStatus(action),
		})
	}
	return s.Repository.FindThreadByID(c, id)
}

// ModerateComment mengikuti aturan yang sama dengan ModerateThread, ditambah
// penyesuaian comment_count thread induknya.
//
// Counter itu menghitung komentar yang terlihat, sama seperti ListComments —
// jadi menyembunyikan atau menghapus komentar harus menurunkannya, dan
// memulihkannya harus menaikkannya. Tanpa itu jumlah di feed akan berbohong.
func (s *service) ModerateComment(c *gin.Context, id, action string, req *DecideContent) (*ThreadComment, error) {
	reason, err := s.decisionReasonForAction(action, req)
	if err != nil {
		return nil, err
	}

	cm, err := s.Repository.FindCommentByID(c, id)
	if err != nil {
		return nil, notFoundAs(err, ErrCommentNotFound)
	}

	now := time.Now()
	var moved bool
	var auditAction string
	var counterDelta int

	switch action {
	case "hide":
		if !CanTransitionThreadComment(cm.Status, StatusHidden) {
			return cm, nil
		}
		moved, err = s.Repository.TransitionComment(c, id, cm.Status, StatusHidden, map[string]any{
			"hidden_by":       currentUserID(c),
			"hidden_at":       now,
			"decision_reason": reason,
		})
		auditAction = actionCommentHide
		counterDelta = -1

	case "restore":
		if !CanTransitionThreadComment(cm.Status, StatusPublished) {
			return cm, nil
		}
		moved, err = s.Repository.TransitionComment(c, id, cm.Status, StatusPublished, map[string]any{
			"hidden_by":       nil,
			"hidden_at":       nil,
			"decision_reason": reason,
		})
		auditAction = actionCommentRestore
		counterDelta = 1

	case "delete":
		fields := map[string]any{
			"deleted_at":      now,
			"decision_reason": reason,
		}
		for _, from := range []string{StatusPublished, StatusHidden} {
			if !CanTransitionThreadComment(cm.Status, StatusDeleted) {
				break
			}
			moved, err = s.Repository.TransitionComment(c, id, from, StatusDeleted, fields)
			if err != nil || moved {
				break
			}
		}
		auditAction = actionCommentDelete
		counterDelta = -1

	default:
		return nil, invalid("Tindakan moderasi tidak dikenal: %s.", action)
	}

	if err != nil {
		return nil, err
	}
	if moved {
		if counterDelta != 0 {
			if err := s.Repository.AdjustThreadCounters(c, cm.ThreadID, counterDelta, 0); err != nil {
				log.Printf("[thread] gagal menyesuaikan comment_count thread %s: %v", cm.ThreadID, err)
			}
		}
		s.writeAudit(c, auditAction, "thread_comment", id, reason, map[string]any{
			"thread_id": cm.ThreadID,
			"from":      cm.Status,
			"to":        targetStatus(action),
		})
	}
	return s.Repository.FindCommentByID(c, id)
}

func targetStatus(action string) string {
	switch action {
	case "hide":
		return StatusHidden
	case "restore":
		return StatusPublished
	case "delete":
		return StatusDeleted
	}
	return ""
}

func (s *service) RestrictedAccounts(c *gin.Context, page, perPage int) ([]*community.PublicProfile, bool, error) {
	page, perPage = gamification.NormalizePagination(page, perPage)
	return s.authors.ListBlocked(c, page, perPage)
}

// RestrictAccount membatasi atau memulihkan akun.
//
// Alasan wajib saat membatasi. Saat mencabut, alasan opsional tetapi tetap
// harus memenuhi panjang minimum bila diisi — jejak "kenapa dibuka" sama
// pentingnya dengan jejak "kenapa ditutup".
func (s *service) RestrictAccount(c *gin.Context, publicID string, req *RestrictAccount) (*community.PublicProfile, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	if req.Blocked {
		if err := RequireReason(req.Reason); err != nil {
			return nil, err
		}
	} else if strings.TrimSpace(req.Reason) != "" {
		if err := RequireReason(req.Reason); err != nil {
			return nil, err
		}
	}

	profile, changed, err := s.authors.SetBlockedByPublicID(c, publicID, req.Blocked)
	if err != nil {
		if errors.Is(err, community.ErrProfileNotFound) {
			return nil, ErrAccountNotFound
		}
		return nil, err
	}

	if changed {
		action := actionAccountUnrestrict
		if req.Blocked {
			action = actionAccountRestrict
		}
		reason := strings.TrimSpace(req.Reason)
		var reasonPtr *string
		if reason != "" {
			reasonPtr = &reason
		}
		s.writeAudit(c, action, "account", publicID, reasonPtr, map[string]any{
			"blocked": req.Blocked,
		})
	}
	return profile, nil
}

// ModerationAudit menampilkan riwayat keputusan moderasi.
//
// Sumbernya audit_logs yang sama dengan jejak dana dan XP, disaring ke entity
// type milik modul ini — satu tabel jejak, beberapa pembaca.
func (s *service) ModerationAudit(c *gin.Context, page, perPage int) ([]*audit.Log, bool, error) {
	perPage, offset := pageWindow(page, perPage)

	rows, err := s.Repository.ListAudit(c, moderationEntityTypes, perPage+1, offset)
	if err != nil {
		return nil, false, err
	}
	hasMore := len(rows) > perPage
	if hasMore {
		rows = rows[:perPage]
	}
	return rows, hasMore, nil
}

// decisionReason memvalidasi alasan wajib pada keputusan laporan.
func (s *service) decisionReason(req *DecideReport) (*string, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	if err := RequireReason(req.Reason); err != nil {
		return nil, err
	}
	trimmed := strings.TrimSpace(req.Reason)
	return &trimmed, nil
}

// decisionReasonForAction memberlakukan aturan alasan per jenis tindakan.
func (s *service) decisionReasonForAction(action string, req *DecideContent) (*string, error) {
	if req == nil {
		return nil, apperr.ErrInvalidRequest
	}
	trimmed := strings.TrimSpace(req.Reason)

	if action == "restore" {
		if trimmed == "" {
			return nil, nil
		}
		if err := RequireReason(trimmed); err != nil {
			return nil, err
		}
		return &trimmed, nil
	}

	if err := RequireReason(trimmed); err != nil {
		return nil, err
	}
	return &trimmed, nil
}

// writeAudit menulis satu baris jejak. Kegagalannya dilaporkan ke log, bukan
// dikembalikan: keputusan moderator yang sudah tersimpan tidak boleh
// dibatalkan hanya karena jejaknya gagal ditulis.
func (s *service) writeAudit(c *gin.Context, action, entityType, entityID string, reason *string, detail map[string]any) {
	entry := audit.NewEntry(currentUserID(c), action, entityType, entityID, reason, detail)
	if err := s.Repository.WriteAudit(c, entry); err != nil {
		log.Printf("[audit] gagal menulis audit %s %s/%s: %v", action, entityType, entityID, err)
	}
}
