package gamification

import (
	"time"

	"mainyuk/internal/audit"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type repository struct {
	db    *gorm.DB
	audit audit.Recorder
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db:    db,
		audit: audit.NewRecorder(db),
	}
}

// WriteAudit menulis satu baris audit lewat penulis bersama di internal/audit.
// Ledger audit bersifat append-only, jadi tidak ada jalur update maupun hapus.
//
// Pemanggil hanya menulis audit bila penulisan terjaga yang mendahuluinya
// benar-benar mengubah baris. Itu yang membuat percobaan ulang tidak
// menggandakan baris audit — bukan index uniknya.
func (r *repository) WriteAudit(c *gin.Context, entry *audit.Log) error {
	return r.audit.Write(c, entry)
}

// ListAudit membaca riwayat modul ini dari tabel audit bersama.
//
// Penyaringnya adalah daftar entity_type, bukan kolom khusus: tabel audit
// dipakai bersama dana, moderasi, dan toko, dan tiap modul memilih barisnya
// sendiri lewat jenis entitas yang memang hanya ia tulis.
func (r *repository) ListAudit(c *gin.Context, entityTypes []string, entityID string, limit, offset int) ([]*audit.Log, error) {
	entries := []*audit.Log{}
	if len(entityTypes) == 0 {
		return entries, nil
	}

	query := r.db.Model(&audit.Log{}).Where("entity_type IN ?", entityTypes)
	if entityID != "" {
		query = query.Where("entity_id = ?", entityID)
	}

	err := query.
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&entries).Error
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// GrantOnce menulis satu entri ledger bila dedup_key-nya belum pernah ada.
//
// Prasyarat: baris community_profiles pemilik sudah ada. Itu dijamin oleh
// service.grant, yang memanggil ProfileEnsurer lebih dulu. Baris profil wajib
// ada karena leaderboard JOIN ke tabel itu — tanpa barisnya, XP yang tercatat
// tidak akan pernah muncul di papan peringkat.
//
// Anti-ganda bergantung sepenuhnya pada index unik xp_ledger.dedup_key.
// ON CONFLICT DO NOTHING dipakai alih-alih menangkap SQLSTATE 23505, karena
// error PostgreSQL membatalkan transaksi yang menampungnya sehingga tidak
// bisa di-commit setelahnya. Dengan DO NOTHING, permintaan kedua hanya
// menghasilkan RowsAffected = 0 tanpa error.
func (r *repository) GrantOnce(c *gin.Context, entry *LedgerEntry) (bool, error) {
	res := r.db.Exec(
		`INSERT INTO xp_ledger
		   (id, user_id, delta, source_type, ref_type, ref_id, dedup_key, note, actor_user_id, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (dedup_key) DO NOTHING`,
		entry.ID, entry.UserID, entry.Delta, entry.SourceType,
		entry.RefType, entry.RefID, entry.DedupKey, entry.Note, entry.ActorUserID,
		time.Now(),
	)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// GrantCappedOnce memberi reward bersumber tunggal di dalam satu transaksi.
//
// Berbeda dari GrantOnce, batas bulanan membuat keputusan bergantung pada isi
// ledger. Karena itu hitung-lalu-sisip di sini diserialkan dengan kunci
// advisory per akun; tanpa itu dua sumber yang dikonfirmasi bersamaan dapat
// sama-sama membaca "belum mencapai batas" lalu sama-sama menyisipkan. Index
// unik dedup_key tetap menjadi jaminan keras terakhir, sehingga reward satu
// sumber tidak mungkin dobel walaupun lock-nya dilewati.
//
// lockKey dipisah per sumber meskipun kuncinya per akun: donasi dan pesanan
// merchandise punya kuota bulanannya sendiri-sendiri, jadi keduanya tidak
// boleh saling memblokir.
//
// Mengembalikan XP yang berlaku untuk sumber ini: delta yang baru tersisip,
// atau delta yang sudah ada bila sumber ini pernah diberi reward. Nilai itu
// dipakai pemanggil sebagai snapshot pada baris sumbernya.
func (r *repository) GrantCappedOnce(c *gin.Context, entry *LedgerEntry, sourceType, lockKey string, from, to time.Time, ruleXP, monthlyCap int) (int, error) {
	granted := 0

	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(
			"SELECT pg_advisory_xact_lock(hashtext(?))",
			lockKey,
		).Error; err != nil {
			return err
		}

		// Batas dihitung sebagai jumlah sumber ber-XP bulan ini, bukan jumlah
		// XP-nya: reward besarnya tetap, sehingga batas dalam satuan XP akan
		// menghabiskan seluruh kuota pada sumber pertama.
		//
		// Sumber ini sendiri dikecualikan dari hitungan. Pada percobaan ulang
		// entrinya sudah ada, dan ikut menghitungnya bisa membuat batas tampak
		// tercapai padahal kuotanya masih tersedia.
		rewardedCount := int64(0)
		if err := tx.Model(&LedgerEntry{}).
			Where("user_id = ?", entry.UserID).
			Where("source_type = ?", sourceType).
			Where("dedup_key <> ?", entry.DedupKey).
			Where("created_at >= ? AND created_at < ?", from, to).
			Count(&rewardedCount).Error; err != nil {
			return err
		}

		decision := CappedRewardDecision(ruleXP, monthlyCap, int(rewardedCount))
		if decision <= 0 {
			return nil
		}
		entry.Delta = decision

		res := tx.Exec(
			`INSERT INTO xp_ledger
			   (id, user_id, delta, source_type, ref_type, ref_id, dedup_key, note, actor_user_id, created_at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
			 ON CONFLICT (dedup_key) DO NOTHING`,
			entry.ID, entry.UserID, entry.Delta, entry.SourceType,
			entry.RefType, entry.RefID, entry.DedupKey, entry.Note, entry.ActorUserID,
			time.Now(),
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			granted = decision
			return nil
		}

		// Entri sudah ada: donasi ini pernah diberi reward. Kembalikan delta
		// yang berlaku supaya snapshot rewarded_xp tidak ikut terhapus.
		var existing int
		row := tx.Raw("SELECT delta FROM xp_ledger WHERE dedup_key = ?", entry.DedupKey).Row()
		if err := row.Scan(&existing); err != nil {
			return err
		}
		granted = existing
		return nil
	})

	if err != nil {
		return 0, err
	}
	return granted, nil
}

func (r *repository) TotalXP(c *gin.Context, userID string) (int, error) {
	total := 0
	err := r.db.Model(&LedgerEntry{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(delta), 0)").
		Scan(&total).Error
	if err != nil {
		return 0, err
	}
	return total, nil
}

func (r *repository) History(c *gin.Context, userID string, limit, offset int) ([]*LedgerEntry, error) {
	entries := []*LedgerEntry{}
	err := r.db.Where("user_id = ?", userID).
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&entries).Error
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func (r *repository) LevelRules(c *gin.Context) ([]*LevelRule, error) {
	rules := []*LevelRule{}
	err := r.db.Order("min_xp ASC").Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}

// UpdateLevelRules menimpa satu aturan level. Dipakai admin untuk menyesuaikan
// kurva tanpa deploy.
func (r *repository) UpdateLevelRules(c *gin.Context, rules []*LevelRule) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, rule := range rules {
			res := tx.Model(&LevelRule{}).
				Where("level = ?", rule.Level).
				Updates(map[string]interface{}{
					"name":           rule.Name,
					"min_xp":         rule.MinXP,
					"badge_label":    rule.BadgeLabel,
					"badge_icon_url": rule.BadgeIconURL,
					"updated_at":     time.Now(),
				})
			if res.Error != nil {
				return res.Error
			}
		}
		return nil
	})
}

func (r *repository) XPRules(c *gin.Context) ([]*XPRule, error) {
	rules := []*XPRule{}
	err := r.db.Order("source_type ASC").Find(&rules).Error
	if err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *repository) XPRule(c *gin.Context, sourceType string) (*XPRule, error) {
	rule := &XPRule{}
	err := r.db.Where("source_type = ?", sourceType).First(rule).Error
	if err != nil {
		return nil, err
	}
	return rule, nil
}

func (r *repository) UpdateXPRule(c *gin.Context, sourceType string, xp int, description *string, monthlyCap *int) error {
	fields := map[string]interface{}{
		"xp": xp,
		// monthlyCap nil berarti tanpa batas, bukan "jangan diubah": form admin
		// mengirim aturan secara utuh, sehingga mengosongkan kolomnya adalah
		// cara menghapus batas.
		"monthly_cap": monthlyCap,
		"updated_at":  time.Now(),
	}
	if description != nil {
		fields["description"] = *description
	}
	return r.db.Model(&XPRule{}).Where("source_type = ?", sourceType).Updates(fields).Error
}

// Leaderboard mengagregasi XP bersih per akun dalam rentang waktu tertentu.
//
// Akun tersembunyi dan terblokir dikecualikan di sini, begitu pula akun yang
// bukan anggota (staf/ranger) dan akun yang sudah dihapus. Tie-break dibuat
// total dan deterministik supaya urutan halaman stabil: XP tertinggi lebih
// dulu, lalu yang lebih awal mencapai XP-nya, lalu id akun sebagai pemutus
// terakhir yang selalu unik.
func (r *repository) Leaderboard(c *gin.Context, from, to *time.Time, limit, offset int) ([]*LeaderboardRow, error) {
	rows := []*LeaderboardRow{}

	query := r.db.Table("xp_ledger AS l").
		Select(`cp.public_id, cp.alias, cp.avatar_url, cp.show_badges,
		        SUM(l.delta) AS net_xp, MIN(l.created_at) AS first_earned_at`).
		Joins("JOIN users u ON u.id = l.user_id AND u.deleted_at IS NULL AND u.role IN ('user', 'jamaah')").
		Joins("JOIN community_profiles cp ON cp.user_id = l.user_id").
		Where("cp.leaderboard_opt_out = false AND cp.is_blocked = false").
		Group("cp.public_id, cp.alias, cp.avatar_url, cp.show_badges, l.user_id").
		Order("net_xp DESC, first_earned_at ASC, l.user_id ASC").
		Limit(limit).
		Offset(offset)

	if from != nil {
		query = query.Where("l.created_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("l.created_at < ?", *to)
	}

	if err := query.Scan(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
