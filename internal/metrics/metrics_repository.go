package metrics

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db}
}

// activeAccountsQuery menghitung akun unik yang berbuat sesuatu dalam rentang.
//
// Aksi dari sembilan modul disatukan lewat UNION ALL, lalu dihitung DISTINCT.
// Yang di-UNION hanya user_id dan created_at — kolom lain tidak dibutuhkan dan
// menyertakannya hanya memperbesar kerja pemindai.
//
// Tabel yang punya deleted_at menyaringnya; thread_reactions dan xp_ledger
// memang tidak punya kolom itu, karena keduanya append-only. presence.user_id
// boleh NULL (baris yang dibuat admin tanpa peserta), jadi ia disaring
// tersendiri — tanpa itu COUNT(DISTINCT) tetap benar, tetapi pemindaiannya
// ikut membawa baris yang tidak mungkin menyumbang akun.
//
// Rentangnya [mulai, selesai). CTE satu baris dipakai supaya dua parameter
// waktu tidak perlu diulang sembilan kali.
const activeAccountsQuery = `
WITH jendela AS (SELECT ?::timestamptz AS mulai, ?::timestamptz AS selesai)
SELECT COUNT(DISTINCT user_id) AS jumlah
FROM (
    SELECT user_id FROM presence, jendela
        WHERE user_id IS NOT NULL AND deleted_at IS NULL
          AND created_at >= jendela.mulai AND created_at < jendela.selesai
    UNION ALL
    SELECT user_id FROM user_tickets, jendela
        WHERE deleted_at IS NULL
          AND created_at >= jendela.mulai AND created_at < jendela.selesai
    UNION ALL
    SELECT user_id FROM donations, jendela
        WHERE deleted_at IS NULL
          AND created_at >= jendela.mulai AND created_at < jendela.selesai
    UNION ALL
    SELECT user_id FROM mission_claims, jendela
        WHERE deleted_at IS NULL
          AND created_at >= jendela.mulai AND created_at < jendela.selesai
    UNION ALL
    SELECT user_id FROM shop_orders, jendela
        WHERE deleted_at IS NULL
          AND created_at >= jendela.mulai AND created_at < jendela.selesai
    UNION ALL
    SELECT user_id FROM threads, jendela
        WHERE deleted_at IS NULL
          AND created_at >= jendela.mulai AND created_at < jendela.selesai
    UNION ALL
    SELECT user_id FROM thread_comments, jendela
        WHERE deleted_at IS NULL
          AND created_at >= jendela.mulai AND created_at < jendela.selesai
    UNION ALL
    SELECT user_id FROM thread_reactions, jendela
        WHERE created_at >= jendela.mulai AND created_at < jendela.selesai
    UNION ALL
    SELECT user_id FROM xp_ledger, jendela
        WHERE created_at >= jendela.mulai AND created_at < jendela.selesai
) AS aktivitas
`

// ActiveAccounts menjalankan query gabungan di atas.
//
// Bebannya nyata: sembilan pemindaian tabel dengan DISTINCT. Endpoint ini
// hanya dipanggil pengurus saat membuka halaman laporan, bukan dari jalur
// panas, dan sengaja tidak diberi cache supaya tidak ada infrastruktur baru
// yang perlu dijaga.
func (r *repository) ActiveAccounts(c *gin.Context, from, to time.Time) (int64, error) {
	var total int64
	if err := r.db.Raw(activeAccountsQuery, from, to).Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (r *repository) ApprovedClaims(c *gin.Context, from, to time.Time) (int64, error) {
	return r.count(`SELECT COUNT(*) FROM mission_claims
		WHERE deleted_at IS NULL AND status = 'approved'
		  AND reviewed_at >= ? AND reviewed_at < ?`, from, to)
}

func (r *repository) ApprovedClaimsTotal(c *gin.Context) (int64, error) {
	return r.count(`SELECT COUNT(*) FROM mission_claims
		WHERE deleted_at IS NULL AND status = 'approved'`)
}

func (r *repository) ConfirmedDonations(c *gin.Context, from, to time.Time) (int64, error) {
	return r.count(`SELECT COUNT(*) FROM donations
		WHERE deleted_at IS NULL AND status = 'confirmed'
		  AND confirmed_at >= ? AND confirmed_at < ?`, from, to)
}

func (r *repository) ConfirmedDonationsTotal(c *gin.Context) (int64, error) {
	return r.count(`SELECT COUNT(*) FROM donations
		WHERE deleted_at IS NULL AND status = 'confirmed'`)
}

func (r *repository) ShopOrders(c *gin.Context, from, to time.Time) (int64, error) {
	return r.count(`SELECT COUNT(*) FROM shop_orders
		WHERE deleted_at IS NULL AND created_at >= ? AND created_at < ?`, from, to)
}

func (r *repository) ShopOrdersByPaymentStatus(c *gin.Context) (map[string]int64, error) {
	return r.groupCount("shop_orders", "payment_status", true)
}

func (r *repository) ShopOrdersByFulfillmentStatus(c *gin.Context) (map[string]int64, error) {
	return r.groupCount("shop_orders", "fulfillment_status", true)
}

func (r *repository) Threads(c *gin.Context, from, to time.Time) (int64, error) {
	return r.count(`SELECT COUNT(*) FROM threads
		WHERE deleted_at IS NULL AND created_at >= ? AND created_at < ?`, from, to)
}

func (r *repository) ThreadComments(c *gin.Context, from, to time.Time) (int64, error) {
	return r.count(`SELECT COUNT(*) FROM thread_comments
		WHERE deleted_at IS NULL AND created_at >= ? AND created_at < ?`, from, to)
}

// ThreadReactions tidak menyaring deleted_at: tabelnya append-only dan tidak
// punya kolom itu.
func (r *repository) ThreadReactions(c *gin.Context, from, to time.Time) (int64, error) {
	return r.count(`SELECT COUNT(*) FROM thread_reactions
		WHERE created_at >= ? AND created_at < ?`, from, to)
}

func (r *repository) ThreadReports(c *gin.Context, from, to time.Time) (int64, error) {
	return r.count(`SELECT COUNT(*) FROM thread_reports
		WHERE created_at >= ? AND created_at < ?`, from, to)
}

func (r *repository) ThreadReportsByStatus(c *gin.Context) (map[string]int64, error) {
	return r.groupCount("thread_reports", "status", false)
}

// count menjalankan satu COUNT(*) dengan parameter yang diberikan.
//
// Query-nya adalah konstanta di berkas ini, tidak pernah dirakit dari masukan
// pemanggil.
func (r *repository) count(query string, args ...any) (int64, error) {
	var total int64
	if err := r.db.Raw(query, args...).Scan(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// groupCount menghitung baris per nilai satu kolom.
//
// table dan column hanya diisi konstanta dari berkas ini; nilainya tidak
// pernah berasal dari permintaan HTTP. Nilai yang belum pernah muncul tidak
// diisi nol — pemanggil yang memutuskan kelengkapan daftarnya.
func (r *repository) groupCount(table, column string, softDeleted bool) (map[string]int64, error) {
	query := "SELECT " + column + " AS kunci, COUNT(*) AS jumlah FROM " + table
	if softDeleted {
		query += " WHERE deleted_at IS NULL"
	}
	query += " GROUP BY " + column

	rows := []struct {
		Kunci  string
		Jumlah int64
	}{}
	if err := r.db.Raw(query).Scan(&rows).Error; err != nil {
		return nil, err
	}

	counts := make(map[string]int64, len(rows))
	for _, row := range rows {
		counts[row.Kunci] = row.Jumlah
	}
	return counts, nil
}
