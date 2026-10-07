package thread

import (
	"errors"
	"time"

	"mainyuk/internal/audit"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// repository adalah satu-satunya tempat paket ini menyentuh database.
//
// Dua hal yang diulang di hampir setiap metode dan sengaja tidak disingkat:
// setiap query baca menambahkan `deleted_at IS NULL` (model memakai
// *time.Time, bukan gorm.DeletedAt, sehingga hapus di sini benar-benar hapus
// bila tidak dijaga), dan setiap perpindahan status memakai UPDATE bersyarat
// `status = from` yang mengembalikan RowsAffected.
type repository struct {
	db    *gorm.DB
	audit audit.Recorder
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db, audit: audit.NewRecorder(db)}
}

// ---------------------------------------------------------------------------
// Thread
// ---------------------------------------------------------------------------

func (r *repository) CreateThread(c *gin.Context, t *Thread) error {
	return r.db.Create(t).Error
}

// insertThreadSQL menyisipkan thread auto-post dengan anti-ganda di tingkat
// database.
//
// ON CONFLICT DO NOTHING dipakai alih-alih menangkap SQLSTATE 23505, karena
// error PostgreSQL membatalkan transaksi yang menampungnya sehingga tidak bisa
// di-commit setelahnya. Dengan DO NOTHING, percobaan kedua hanya menghasilkan
// RowsAffected = 0 tanpa error.
const insertThreadSQL = `INSERT INTO threads
    (id, public_id, user_id, title, body, status, source_type, source_ref,
     comment_count, reaction_count, created_at, updated_at)
  VALUES (?, ?, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?)
  ON CONFLICT (source_type, source_ref) DO NOTHING`

// CreateThreadIfAbsent menyisipkan auto-post, atau melaporkan bahwa sumbernya
// sudah pernah dibagikan.
//
// Index uniknya sengaja tanpa predikat deleted_at, jadi pencabutan izin
// bersifat permanen: sumber yang postingannya sudah dihapus tidak akan pernah
// terbit lagi. Itu yang membuat "mematikan lalu menyalakan preferensi" tidak
// menghidupkan kiriman lama.
func (r *repository) CreateThreadIfAbsent(c *gin.Context, t *Thread) (bool, error) {
	res := r.db.Exec(
		insertThreadSQL,
		t.ID, t.PublicID, t.UserID, t.Title, t.Body, t.Status,
		t.SourceType, t.SourceRef, t.CreatedAt, t.UpdatedAt,
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *repository) FindThreadByPublicID(c *gin.Context, publicID string) (*Thread, error) {
	t := &Thread{}
	err := r.db.Where("public_id = ?", publicID).Where("deleted_at IS NULL").First(t).Error
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (r *repository) FindThreadByID(c *gin.Context, id string) (*Thread, error) {
	t := &Thread{}
	err := r.db.Where("id = ?", id).Where("deleted_at IS NULL").First(t).Error
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ListThreads adalah query feed.
//
// Subquery NOT EXISTS menyembunyikan kiriman penulis yang sedang dibatasi.
// Dipilih ketimbang join supaya penulis yang belum punya baris profil tetap
// ikut tampil — join inner akan membuang mereka tanpa alasan. Efek sampingnya
// yang diinginkan: blokir dan pencabutannya langsung berlaku pada seluruh
// kiriman lama tanpa perlu update massal.
func (r *repository) ListThreads(c *gin.Context, sort string, limit, offset int) ([]*Thread, error) {
	rows := []*Thread{}

	query := r.db.Model(&Thread{}).
		Where("threads.status = ?", StatusPublished).
		Where("threads.deleted_at IS NULL").
		Where(`NOT EXISTS (
			SELECT 1 FROM community_profiles cp
			WHERE cp.user_id = threads.user_id AND cp.is_blocked
		)`)

	if sort == SortPopuler {
		query = query.Order("threads.reaction_count DESC, threads.created_at DESC, threads.id DESC")
	} else {
		query = query.Order("threads.created_at DESC, threads.id DESC")
	}

	err := query.Limit(limit).Offset(offset).Find(&rows).Error
	return rows, err
}

// ListThreadsForReview dipakai moderator, jadi penulis yang dibatasi tidak
// disembunyikan: justru merekalah yang perlu diperiksa. Status kosong berarti
// semua status yang belum terhapus.
func (r *repository) ListThreadsForReview(c *gin.Context, status string, limit, offset int) ([]*Thread, error) {
	rows := []*Thread{}

	query := r.db.Model(&Thread{}).Where("deleted_at IS NULL")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.
		Order("created_at DESC, id DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error
	return rows, err
}

// TransitionThread memindahkan status thread secara atomik.
//
// Guard status = from adalah inti idempotensinya: dua moderator yang
// menyembunyikan bersamaan diserialkan oleh lock baris, dan yang kedua
// mengevaluasi ulang terhadap baris yang sudah ter-commit sehingga tidak
// menemukan status asalnya. RowsAffected = 0 karena itu berarti "sudah
// berpindah", bukan kegagalan.
func (r *repository) TransitionThread(c *gin.Context, id, from, to string, fields map[string]any) (bool, error) {
	if fields == nil {
		fields = map[string]any{}
	}
	fields["status"] = to
	fields["updated_at"] = time.Now()

	res := r.db.Model(&Thread{}).
		Where("id = ?", id).
		Where("status = ?", from).
		Where("deleted_at IS NULL").
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// SoftDeleteOwnThread menghapus thread milik pemanggil. Syarat user_id ada di
// dalam query, bukan di pemeriksaan terpisah, supaya tidak ada celah antara
// "memeriksa pemilik" dan "menghapus".
func (r *repository) SoftDeleteOwnThread(c *gin.Context, publicID, userID string) (bool, error) {
	now := time.Now()
	res := r.db.Model(&Thread{}).
		Where("public_id = ?", publicID).
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Where("status IN ?", []string{StatusPublished, StatusHidden}).
		Updates(map[string]any{
			"status":     StatusDeleted,
			"deleted_at": now,
			"updated_at": now,
		})
	return res.RowsAffected > 0, res.Error
}

// SoftDeleteThreadBySource mencabut satu auto-post berdasarkan rujukan
// sumbernya. Mengembalikan false bila tidak ada yang perlu dicabut — termasuk
// ketika postingannya sudah terhapus sebelumnya.
func (r *repository) SoftDeleteThreadBySource(c *gin.Context, sourceType, sourceRef string) (bool, error) {
	now := time.Now()
	res := r.db.Model(&Thread{}).
		Where("source_type = ?", sourceType).
		Where("source_ref = ?", sourceRef).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"status":     StatusDeleted,
			"deleted_at": now,
			"updated_at": now,
		})
	return res.RowsAffected > 0, res.Error
}

// SoftDeleteThreadsBySource mencabut seluruh auto-post satu jenis aktivitas
// milik satu akun. Dipakai ketika anggota mematikan preferensi berbaginya.
//
// Thread manual tidak pernah tersentuh: syarat source_type = ? hanya cocok
// untuk baris auto-post, karena thread manual menyimpan NULL di sana.
func (r *repository) SoftDeleteThreadsBySource(c *gin.Context, userID, sourceType string) (int64, error) {
	now := time.Now()
	res := r.db.Model(&Thread{}).
		Where("user_id = ?", userID).
		Where("source_type = ?", sourceType).
		Where("deleted_at IS NULL").
		Updates(map[string]any{
			"status":     StatusDeleted,
			"deleted_at": now,
			"updated_at": now,
		})
	return res.RowsAffected, res.Error
}

// AdjustThreadCounters memakai ekspresi SQL supaya dua komentar yang masuk
// bersamaan tidak saling menimpa, dan GREATEST menahan counter agar tidak
// pernah negatif bila ada penghapusan yang datang berurutan tidak seperti
// dugaan.
func (r *repository) AdjustThreadCounters(c *gin.Context, id string, commentDelta, reactionDelta int) error {
	return r.db.Model(&Thread{}).Where("id = ?", id).Updates(map[string]any{
		"comment_count":  gorm.Expr("GREATEST(comment_count + ?, 0)", commentDelta),
		"reaction_count": gorm.Expr("GREATEST(reaction_count + ?, 0)", reactionDelta),
		"updated_at":     time.Now(),
	}).Error
}

// ReactionsByUser melaporkan thread mana saja yang sudah direaksikan
// pemanggil, dalam satu query untuk satu halaman feed.
func (r *repository) ReactionsByUser(c *gin.Context, userID string, threadIDs []string) (map[string]bool, error) {
	reacted := make(map[string]bool, len(threadIDs))
	if userID == "" || len(threadIDs) == 0 {
		return reacted, nil
	}

	ids := []string{}
	err := r.db.Model(&ThreadReaction{}).
		Where("user_id = ?", userID).
		Where("thread_id IN ?", threadIDs).
		Pluck("thread_id", &ids).Error
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		reacted[id] = true
	}
	return reacted, nil
}

// ---------------------------------------------------------------------------
// Komentar
// ---------------------------------------------------------------------------

func (r *repository) CreateComment(c *gin.Context, cm *ThreadComment) error {
	return r.db.Create(cm).Error
}

func (r *repository) FindCommentByPublicID(c *gin.Context, publicID string) (*ThreadComment, error) {
	cm := &ThreadComment{}
	err := r.db.Where("public_id = ?", publicID).Where("deleted_at IS NULL").First(cm).Error
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func (r *repository) FindCommentByID(c *gin.Context, id string) (*ThreadComment, error) {
	cm := &ThreadComment{}
	err := r.db.Where("id = ?", id).Where("deleted_at IS NULL").First(cm).Error
	if err != nil {
		return nil, err
	}
	return cm, nil
}

// ListComments menyaring dengan aturan yang sama seperti feed: komentar dari
// penulis yang sedang dibatasi tidak ikut terkirim. Saringan ini ada di SQL,
// bukan di Go setelah barisnya diambil, supaya limit+1 tetap berarti dan
// halaman tidak pernah datang kosong padahal has_more bernilai benar.
func (r *repository) ListComments(c *gin.Context, threadID string, limit, offset int) ([]*ThreadComment, error) {
	rows := []*ThreadComment{}
	err := r.db.Model(&ThreadComment{}).
		Where("thread_comments.thread_id = ?", threadID).
		Where("thread_comments.status = ?", StatusPublished).
		Where("thread_comments.deleted_at IS NULL").
		Where(`NOT EXISTS (
			SELECT 1 FROM community_profiles cp
			WHERE cp.user_id = thread_comments.user_id AND cp.is_blocked
		)`).
		Order("thread_comments.created_at ASC, thread_comments.id ASC").
		Limit(limit).Offset(offset).
		Find(&rows).Error
	return rows, err
}

// TransitionComment memakai pola yang sama dengan TransitionThread.
func (r *repository) TransitionComment(c *gin.Context, id, from, to string, fields map[string]any) (bool, error) {
	if fields == nil {
		fields = map[string]any{}
	}
	fields["status"] = to
	fields["updated_at"] = time.Now()

	res := r.db.Model(&ThreadComment{}).
		Where("id = ?", id).
		Where("status = ?", from).
		Where("deleted_at IS NULL").
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *repository) SoftDeleteOwnComment(c *gin.Context, publicID, userID string) (bool, error) {
	now := time.Now()
	res := r.db.Model(&ThreadComment{}).
		Where("public_id = ?", publicID).
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Where("status IN ?", []string{StatusPublished, StatusHidden}).
		Updates(map[string]any{
			"status":     StatusDeleted,
			"deleted_at": now,
			"updated_at": now,
		})
	return res.RowsAffected > 0, res.Error
}

// ---------------------------------------------------------------------------
// Reaksi
// ---------------------------------------------------------------------------

// AddReaction mengembalikan false bila akun itu sudah pernah bereaksi pada
// thread yang sama. Index unik (thread_id, user_id) yang menjaminnya — inilah
// penjaga yang tidak dimiliki tabel likes.
func (r *repository) AddReaction(c *gin.Context, reaction *ThreadReaction) (bool, error) {
	res := r.db.Exec(
		`INSERT INTO thread_reactions (id, thread_id, user_id, kind, created_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (thread_id, user_id) DO NOTHING`,
		reaction.ID, reaction.ThreadID, reaction.UserID, reaction.Kind, reaction.CreatedAt,
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// RemoveReaction adalah hard delete: tabelnya tidak punya deleted_at, supaya
// bereaksi ulang setelah batal berfungsi bersih.
func (r *repository) RemoveReaction(c *gin.Context, threadID, userID string) (bool, error) {
	res := r.db.
		Where("thread_id = ?", threadID).
		Where("user_id = ?", userID).
		Delete(&ThreadReaction{})
	return res.RowsAffected > 0, res.Error
}

// ---------------------------------------------------------------------------
// Laporan
// ---------------------------------------------------------------------------

// CreateReport mengembalikan false bila akun itu sudah melaporkan target yang
// sama. Itu yang menahan spam laporan tanpa perlu pemeriksaan terpisah yang
// bisa balapan.
func (r *repository) CreateReport(c *gin.Context, report *Report) (bool, error) {
	res := r.db.Exec(
		`INSERT INTO thread_reports
		   (id, public_id, reporter_user_id, target_type, target_id, reason, note, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (reporter_user_id, target_type, target_id) DO NOTHING`,
		report.ID, report.PublicID, report.ReporterUserID, report.TargetType,
		report.TargetID, report.Reason, report.Note, report.Status,
		report.CreatedAt, report.UpdatedAt,
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *repository) FindReportByID(c *gin.Context, id string) (*Report, error) {
	report := &Report{}
	if err := r.db.Where("id = ?", id).First(report).Error; err != nil {
		return nil, err
	}
	return report, nil
}

func (r *repository) ListReports(c *gin.Context, status string, limit, offset int) ([]*Report, error) {
	rows := []*Report{}

	query := r.db.Model(&Report{})
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.
		Order("created_at DESC, id DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error
	return rows, err
}

func (r *repository) TransitionReport(c *gin.Context, id, from, to string, fields map[string]any) (bool, error) {
	if fields == nil {
		fields = map[string]any{}
	}
	fields["status"] = to
	fields["updated_at"] = time.Now()

	res := r.db.Model(&Report{}).
		Where("id = ?", id).
		Where("status = ?", from).
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ---------------------------------------------------------------------------
// Preferensi berbagi
// ---------------------------------------------------------------------------

// FindSharePrefs mengembalikan (nil, nil) ketika barisnya belum ada.
//
// Ketiadaan baris bukan kesalahan dan bukan pula "semua mati": default-nya
// adalah semua nyala, dan keputusan itu ada di ResolveSharePrefs, bukan di
// sini. Mengembalikan baris kosong buatan akan menyamarkan perbedaan antara
// "belum pernah diatur" dan "sengaja diatur".
func (r *repository) FindSharePrefs(c *gin.Context, userID string) (*SharePrefs, error) {
	prefs := &SharePrefs{}
	err := r.db.Where("user_id = ?", userID).First(prefs).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return prefs, nil
}

func (r *repository) UpsertSharePrefs(c *gin.Context, prefs *SharePrefs) error {
	return r.db.Exec(
		`INSERT INTO thread_share_prefs
		   (user_id, share_event_registration, share_donation, share_mission, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (user_id) DO UPDATE SET
		   share_event_registration = EXCLUDED.share_event_registration,
		   share_donation           = EXCLUDED.share_donation,
		   share_mission            = EXCLUDED.share_mission,
		   updated_at               = EXCLUDED.updated_at`,
		prefs.UserID, prefs.ShareEventRegistration, prefs.ShareDonation,
		prefs.ShareMission, prefs.CreatedAt, prefs.UpdatedAt,
	).Error
}

// ---------------------------------------------------------------------------
// Audit
// ---------------------------------------------------------------------------

func (r *repository) WriteAudit(c *gin.Context, entry *audit.Log) error {
	return r.audit.Write(c, entry)
}

func (r *repository) ListAudit(c *gin.Context, entityTypes []string, limit, offset int) ([]*audit.Log, error) {
	rows := []*audit.Log{}

	query := r.db.Model(&audit.Log{})
	if len(entityTypes) > 0 {
		query = query.Where("entity_type IN ?", entityTypes)
	}

	err := query.
		Order("created_at DESC, id DESC").
		Limit(limit).Offset(offset).
		Find(&rows).Error
	return rows, err
}
