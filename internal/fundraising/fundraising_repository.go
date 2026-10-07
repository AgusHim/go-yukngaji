package fundraising

import (
	"errors"
	"time"

	"mainyuk/internal/audit"

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
	// audit adalah penulis bersama untuk tabel audit_logs. Recorder dibuat dari
	// *gorm.DB yang sama supaya tanda tangan NewRepository tidak perlu berubah
	// hanya karena penulisnya dipindahkan ke internal/audit.
	audit audit.Recorder
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{db: db, audit: audit.NewRecorder(db)}
}

// ---------------------------------------------------------------------------
// Campaign
// ---------------------------------------------------------------------------

// ListPublished mengembalikan campaign yang sedang menerima donasi.
//
// Predikatnya sama persis dengan DonationWindowOpen, hanya saja diterjemahkan
// ke SQL. Keduanya harus sepakat: daftar publik tidak boleh menampilkan
// campaign yang donasinya akan ditolak saat dikirim.
func (r *repository) ListPublished(c *gin.Context, now time.Time) ([]*Campaign, error) {
	campaigns := []*Campaign{}
	err := r.db.
		Where("deleted_at IS NULL").
		Where("status = ?", CampaignPublished).
		Where("(starts_at IS NULL OR starts_at <= ?)", now).
		Where("(ends_at IS NULL OR ends_at > ?)", now).
		Order("created_at DESC").
		Find(&campaigns).Error
	if err != nil {
		return nil, err
	}
	return campaigns, nil
}

func (r *repository) FindPublishedBySlug(c *gin.Context, slug string, now time.Time) (*Campaign, error) {
	campaign := &Campaign{}
	err := r.db.
		Where("slug = ?", slug).
		Where("deleted_at IS NULL").
		Where("status = ?", CampaignPublished).
		Where("(starts_at IS NULL OR starts_at <= ?)", now).
		Where("(ends_at IS NULL OR ends_at > ?)", now).
		First(campaign).Error
	if err != nil {
		return nil, err
	}
	return campaign, nil
}

func (r *repository) FindByID(c *gin.Context, id string) (*Campaign, error) {
	campaign := &Campaign{}
	err := r.db.Where("id = ?", id).Where("deleted_at IS NULL").First(campaign).Error
	if err != nil {
		return nil, err
	}
	return campaign, nil
}

func (r *repository) ListAll(c *gin.Context, status string) ([]*Campaign, error) {
	campaigns := []*Campaign{}

	query := r.db.Where("deleted_at IS NULL")
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Order("created_at DESC").Find(&campaigns).Error
	if err != nil {
		return nil, err
	}
	return campaigns, nil
}

func (r *repository) CreateCampaign(c *gin.Context, campaign *Campaign) error {
	return r.db.Create(campaign).Error
}

func (r *repository) UpdateCampaign(c *gin.Context, id string, fields map[string]interface{}) error {
	fields["updated_at"] = time.Now()
	return r.db.Model(&Campaign{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Updates(fields).Error
}

// TransitionCampaign memindahkan status campaign secara atomik.
//
// Guard status=from adalah inti idempotensinya: dua pengurus yang mengubah
// status bersamaan diserialkan oleh lock baris, dan yang kedua mengevaluasi
// ulang terhadap baris yang sudah ter-commit sehingga tidak menemukan status
// asalnya. RowsAffected = 0 karena itu berarti "sudah berpindah", bukan
// kegagalan.
func (r *repository) TransitionCampaign(c *gin.Context, id, from, to string, fields map[string]interface{}) (bool, error) {
	if fields == nil {
		fields = map[string]interface{}{}
	}
	fields["status"] = to
	fields["updated_at"] = time.Now()

	res := r.db.Model(&Campaign{}).
		Where("id = ?", id).
		Where("status = ?", from).
		Where("deleted_at IS NULL").
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *repository) SoftDeleteCampaign(c *gin.Context, id string) error {
	now := time.Now()
	return r.db.Model(&Campaign{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"updated_at": now,
		}).Error
}

// ---------------------------------------------------------------------------
// Agregat
// ---------------------------------------------------------------------------

// CampaignTotals menghitung dana terkumpul dan jumlah donatur satu campaign.
//
// Progres selalu dijumlahkan saat dibaca, tidak pernah disimpan sebagai
// counter. Itulah sebabnya "konfirmasi berulang tidak menggandakan progres"
// benar secara struktural: satu donasi hanya menyumbang sekali, karena
// statusnya hanya bisa berpindah sekali.
//
// SUM(bigint) mengembalikan numeric di Postgres, jadi hasilnya di-cast
// ::bigint di sini — tanpa cast itu pemindaian ke int Go bisa gagal.
func (r *repository) CampaignTotals(c *gin.Context, campaignID string) (*CampaignTotals, error) {
	row := struct {
		RaisedAmount int
		DonorCount   int
	}{}
	err := r.db.Model(&Donation{}).
		Where("campaign_id = ?", campaignID).
		Where("status = ?", DonationConfirmed).
		Where("deleted_at IS NULL").
		Select("COALESCE(SUM(amount), 0)::bigint AS raised_amount, COUNT(*) AS donor_count").
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &CampaignTotals{RaisedAmount: row.RaisedAmount, DonorCount: row.DonorCount}, nil
}

// CampaignTotalsBatch menghitung progres banyak campaign sekaligus, supaya
// daftar campaign tidak menghasilkan satu query per baris.
func (r *repository) CampaignTotalsBatch(c *gin.Context, campaignIDs []string) (map[string]*CampaignTotals, error) {
	result := map[string]*CampaignTotals{}
	if len(campaignIDs) == 0 {
		return result, nil
	}

	rows := []struct {
		CampaignID   string
		RaisedAmount int
		DonorCount   int
	}{}
	err := r.db.Model(&Donation{}).
		Where("campaign_id IN ?", campaignIDs).
		Where("status = ?", DonationConfirmed).
		Where("deleted_at IS NULL").
		Select("campaign_id, COALESCE(SUM(amount), 0)::bigint AS raised_amount, COUNT(*) AS donor_count").
		Group("campaign_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		result[row.CampaignID] = &CampaignTotals{
			RaisedAmount: row.RaisedAmount,
			DonorCount:   row.DonorCount,
		}
	}
	return result, nil
}

// UpdateTotals menjumlahkan baris penggunaan dana dan biaya.
//
// Keduanya dicatat terpisah dan TIDAK mengurangi progres; ia hanya muncul
// sebagai baris di laporan supaya tidak ada uang yang hilang dari catatan.
func (r *repository) UpdateTotals(c *gin.Context, campaignID string) (*UpdateTotals, error) {
	rows := []struct {
		Kind   string
		Amount int
	}{}
	err := r.db.Model(&CampaignUpdate{}).
		Where("campaign_id = ?", campaignID).
		Where("kind IN ?", []string{UpdateKindUsage, UpdateKindFee}).
		Where("deleted_at IS NULL").
		Select("kind, COALESCE(SUM(amount), 0)::bigint AS amount").
		Group("kind").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	totals := &UpdateTotals{}
	for _, row := range rows {
		switch row.Kind {
		case UpdateKindUsage:
			totals.UsageAmount = row.Amount
		case UpdateKindFee:
			totals.FeeAmount = row.Amount
		}
	}
	return totals, nil
}

// DonationAggregates merangkum donasi satu campaign per status.
//
// Donasi 'cancelled' sengaja tidak dilaporkan: donatur membatalkan sebelum
// transfer, jadi tidak ada uang yang pernah masuk maupun gagal masuk.
func (r *repository) DonationAggregates(c *gin.Context, campaignID string) (*ReportInput, error) {
	rows := []struct {
		Status string
		Amount int
		Count  int
	}{}
	err := r.db.Model(&Donation{}).
		Where("campaign_id = ?", campaignID).
		Where("deleted_at IS NULL").
		Select("status, COALESCE(SUM(amount), 0)::bigint AS amount, COUNT(*) AS count").
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	in := &ReportInput{}
	for _, row := range rows {
		switch row.Status {
		case DonationConfirmed:
			in.ConfirmedAmount, in.ConfirmedCount = row.Amount, row.Count
		case DonationPending:
			in.PendingAmount, in.PendingCount = row.Amount, row.Count
		case DonationRejected:
			in.RejectedAmount, in.RejectedCount = row.Amount, row.Count
		case DonationRefunded:
			in.RefundedAmount, in.RefundedCount = row.Amount, row.Count
		}
	}

	totals, err := r.UpdateTotals(c, campaignID)
	if err != nil {
		return nil, err
	}
	in.FeeAmount = totals.FeeAmount
	in.UsageAmount = totals.UsageAmount
	return in, nil
}

// ---------------------------------------------------------------------------
// Donasi
// ---------------------------------------------------------------------------

// insertDonationSQL menyisipkan donasi dengan anti-ganda di tingkat database.
//
// ON CONFLICT menyebut predikat index parsialnya, karena Postgres menolak
// inferensi terhadap index parsial yang predikatnya tidak disertakan.
const insertDonationSQL = `INSERT INTO donations
    (id, public_id, campaign_id, user_id, amount, status, is_anonymous, show_amount,
     message, message_status, payment_method_id, client_token, rewarded_xp, created_at, updated_at)
  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
  ON CONFLICT (user_id, client_token)
    WHERE client_token IS NOT NULL AND deleted_at IS NULL
  DO NOTHING`

// CreateDonationGuarded menyisipkan donasi, atau mengembalikan donasi yang
// sudah ada bila token idempotensinya sudah terpakai.
//
// ON CONFLICT DO NOTHING dipakai alih-alih menangkap SQLSTATE 23505, karena
// error PostgreSQL membatalkan transaksi yang menampungnya sehingga tidak bisa
// di-commit setelahnya. Dengan DO NOTHING, permintaan kedua hanya menghasilkan
// RowsAffected = 0 tanpa error.
func (r *repository) CreateDonationGuarded(c *gin.Context, donation *Donation) (*Donation, bool, error) {
	existing := &Donation{}
	created := false

	err := r.db.Transaction(func(tx *gorm.DB) error {
		res := tx.Exec(
			insertDonationSQL,
			donation.ID, donation.PublicID, donation.CampaignID, donation.UserID,
			donation.Amount, donation.Status, donation.IsAnonymous, donation.ShowAmount,
			donation.Message, donation.MessageStatus, donation.PaymentMethodID,
			donation.ClientToken, donation.RewardedXP, donation.CreatedAt, donation.UpdatedAt,
		)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected > 0 {
			created = true
			return nil
		}

		// Token sudah dipakai: kembalikan donasi yang sudah ada, bukan galat.
		return tx.
			Where("user_id = ?", donation.UserID).
			Where("client_token = ?", donation.ClientToken).
			Where("deleted_at IS NULL").
			First(existing).Error
	})

	if err != nil {
		return nil, false, err
	}
	if created {
		return donation, true, nil
	}
	return existing, false, nil
}

func (r *repository) FindDonationByPublicID(c *gin.Context, publicID string) (*Donation, error) {
	donation := &Donation{}
	err := r.db.
		Where("public_id = ?", publicID).
		Where("deleted_at IS NULL").
		First(donation).Error
	if err != nil {
		return nil, err
	}
	return donation, nil
}

func (r *repository) FindDonationByID(c *gin.Context, id string) (*Donation, error) {
	donation := &Donation{}
	err := r.db.Where("id = ?", id).Where("deleted_at IS NULL").First(donation).Error
	if err != nil {
		return nil, err
	}
	return donation, nil
}

func (r *repository) ListDonationsByUser(c *gin.Context, userID string, limit, offset int) ([]*Donation, error) {
	donations := []*Donation{}
	err := r.db.
		Where("user_id = ?", userID).
		Where("deleted_at IS NULL").
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&donations).Error
	if err != nil {
		return nil, err
	}
	return donations, nil
}

func (r *repository) ListDonationsForReview(c *gin.Context, filter DonationFilter, limit, offset int) ([]*Donation, error) {
	donations := []*Donation{}

	query := r.db.Where("deleted_at IS NULL")
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.MessageStatus != "" {
		query = query.Where("message_status = ?", filter.MessageStatus)
	}
	if filter.CampaignID != "" {
		query = query.Where("campaign_id = ?", filter.CampaignID)
	}

	err := query.
		Order("created_at ASC, id ASC").
		Limit(limit).
		Offset(offset).
		Find(&donations).Error
	if err != nil {
		return nil, err
	}
	return donations, nil
}

// TransitionDonation memindahkan status donasi secara atomik.
//
// Sama seperti TransitionCampaign, guard status=from membuat konfirmasi ulang
// tidak berpengaruh apa-apa: RowsAffected = 0, bukan kegagalan.
func (r *repository) TransitionDonation(c *gin.Context, id, from, to string, fields map[string]interface{}) (bool, error) {
	if fields == nil {
		fields = map[string]interface{}{}
	}
	fields["status"] = to
	fields["updated_at"] = time.Now()

	res := r.db.Model(&Donation{}).
		Where("id = ?", id).
		Where("status = ?", from).
		Where("deleted_at IS NULL").
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// SetDonationReward menyimpan snapshot XP yang benar-benar diberikan.
//
// Nilai 0 tidak pernah menimpa snapshot yang sudah ada: bila pemberian reward
// gagal sebagian lalu diulang, snapshot lama tetap menjadi acuan pembalikan.
func (r *repository) SetDonationReward(c *gin.Context, id string, xp int) error {
	if xp <= 0 {
		return nil
	}
	return r.db.Model(&Donation{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Updates(map[string]interface{}{
			"rewarded_xp": xp,
			"updated_at":  time.Now(),
		}).Error
}

// ListDonors mengembalikan donasi terkonfirmasi satu campaign untuk daftar
// donatur publik.
func (r *repository) ListDonors(c *gin.Context, campaignID string, limit, offset int) ([]*Donation, error) {
	donations := []*Donation{}
	err := r.db.
		Where("campaign_id = ?", campaignID).
		Where("status = ?", DonationConfirmed).
		Where("deleted_at IS NULL").
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&donations).Error
	if err != nil {
		return nil, err
	}
	return donations, nil
}

// ListApprovedMessages mengembalikan donasi terkonfirmasi yang pesannya sudah
// disetujui. Ini satu-satunya jalur yang boleh menampilkan pesan donasi.
func (r *repository) ListApprovedMessages(c *gin.Context, campaignID string, limit, offset int) ([]*Donation, error) {
	donations := []*Donation{}
	err := r.db.
		Where("campaign_id = ?", campaignID).
		Where("status = ?", DonationConfirmed).
		Where("message_status = ?", MessageApproved).
		Where("message IS NOT NULL").
		Where("deleted_at IS NULL").
		Order("created_at DESC, id DESC").
		Limit(limit).
		Offset(offset).
		Find(&donations).Error
	if err != nil {
		return nil, err
	}
	return donations, nil
}

// TransitionMessage memindahkan status moderasi pesan secara atomik.
func (r *repository) TransitionMessage(c *gin.Context, id, from, to string, fields map[string]interface{}) (bool, error) {
	if fields == nil {
		fields = map[string]interface{}{}
	}
	fields["message_status"] = to
	fields["updated_at"] = time.Now()

	res := r.db.Model(&Donation{}).
		Where("id = ?", id).
		Where("message_status = ?", from).
		Where("deleted_at IS NULL").
		Updates(fields)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ---------------------------------------------------------------------------
// Update campaign
// ---------------------------------------------------------------------------

func (r *repository) ListUpdates(c *gin.Context, campaignID string, publishedOnly bool) ([]*CampaignUpdate, error) {
	updates := []*CampaignUpdate{}

	query := r.db.Where("campaign_id = ?", campaignID).Where("deleted_at IS NULL")
	if publishedOnly {
		query = query.Where("is_published = true")
	}

	err := query.Order("created_at DESC, id DESC").Find(&updates).Error
	if err != nil {
		return nil, err
	}
	return updates, nil
}

func (r *repository) FindUpdate(c *gin.Context, id string) (*CampaignUpdate, error) {
	update := &CampaignUpdate{}
	err := r.db.Where("id = ?", id).Where("deleted_at IS NULL").First(update).Error
	if err != nil {
		return nil, err
	}
	return update, nil
}

func (r *repository) CreateUpdate(c *gin.Context, update *CampaignUpdate) error {
	return r.db.Create(update).Error
}

func (r *repository) UpdateUpdate(c *gin.Context, id string, fields map[string]interface{}) error {
	fields["updated_at"] = time.Now()
	return r.db.Model(&CampaignUpdate{}).
		Where("id = ?", id).
		Where("deleted_at IS NULL").
		Updates(fields).Error
}

func (r *repository) SoftDeleteUpdate(c *gin.Context, id string) error {
	now := time.Now()
	return r.db.Model(&CampaignUpdate{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"deleted_at": now,
			"updated_at": now,
		}).Error
}

// ---------------------------------------------------------------------------
// Audit
// ---------------------------------------------------------------------------

// WriteAudit menulis satu baris audit lewat penulis bersama di internal/audit.
// Ledger audit bersifat append-only, jadi tidak ada jalur update maupun hapus.
//
// Pemanggil hanya menulis audit bila penulisan terjaga yang mendahuluinya
// benar-benar mengubah baris (RowsAffected > 0). Itu yang membuat percobaan
// ulang tidak menggandakan baris audit — bukan index uniknya. Karena itu
// dedup_key diisi ID entri (uuid unik per keputusan), sehingga
// ON CONFLICT DO NOTHING hanya berperan sebagai jaring pengaman untuk
// pemanggilan yang benar-benar identik, bukan sebagai mekanisme idempotensi
// utama.
func (r *repository) WriteAudit(c *gin.Context, entry *AuditLog) error {
	return r.audit.Write(c, entry)
}

func (r *repository) ListAudit(c *gin.Context, entityType, entityID string, limit, offset int) ([]*AuditLog, error) {
	entries := []*AuditLog{}

	query := r.db.Model(&AuditLog{})
	if entityType != "" {
		query = query.Where("entity_type = ?", entityType)
	}
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
