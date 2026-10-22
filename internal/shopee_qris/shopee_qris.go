// Package shopee_qris menyimpan charge QRIS dan mencocokkannya dengan mutasi
// Shopee secara persisten. Database adalah sumber kebenaran untuk deduplikasi;
// state proses Go tidak pernah dipakai untuk menentukan pembayaran.
package shopee_qris

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"mainyuk/internal/shopee"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	EntityEventOrder = "event_order"
	EntityDonation   = "donation"
	EntityShopOrder  = "shop_order"

	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusExpired   = "expired"
	StatusCancelled = "cancelled"

	defaultLifetime = 30 * time.Minute
	amountHold      = 24 * time.Hour
	settlementLease = 90 * time.Second
)

// Charge adalah satu nominal QRIS unik untuk tepat satu transaksi domain.
type Charge struct {
	ID                    string     `gorm:"column:id;primaryKey"`
	EntityType            string     `gorm:"column:entity_type"`
	EntityID              string     `gorm:"column:entity_id"`
	BaseAmount            int64      `gorm:"column:base_amount"`
	UniqueCode            int        `gorm:"column:unique_code"`
	PayableAmount         int64      `gorm:"column:payable_amount"`
	QRISPayload           string     `gorm:"column:qris_payload"`
	Status                string     `gorm:"column:status"`
	ShopeeTransactionID   *string    `gorm:"column:shopee_transaction_id"`
	ExpiresAt             time.Time  `gorm:"column:expires_at"`
	PaidAt                *time.Time `gorm:"column:paid_at"`
	SettledAt             *time.Time `gorm:"column:settled_at"`
	SettlementAttempts    int        `gorm:"column:settlement_attempts"`
	SettlementError       *string    `gorm:"column:settlement_error"`
	SettlementLockedUntil *time.Time `gorm:"column:settlement_locked_until"`
	CreatedAt             time.Time  `gorm:"column:created_at"`
	UpdatedAt             time.Time  `gorm:"column:updated_at"`
}

func (Charge) TableName() string { return "shopee_qris_charges" }

// Settler menjalankan efek domain setelah mutasi sudah diklaim oleh charge.
// Implementasinya wajib idempoten karena worker akan mengulang kegagalan.
type Settler func(ctx *gin.Context, entityType, entityID string, paidAmount int64, reference string) error
type Eligibility func(ctx *gin.Context, entityType, entityID string) (bool, error)

type Service struct {
	db       *gorm.DB
	shopee   *shopee.Service
	settler  Settler
	eligible Eligibility
	interval time.Duration
}

func New(database *gorm.DB, provider *shopee.Service) *Service {
	return &Service{db: database, shopee: provider, interval: 15 * time.Second}
}

func (s *Service) SetSettler(settler Settler)          { s.settler = settler }
func (s *Service) SetEligibility(eligible Eligibility) { s.eligible = eligible }

// Validate memastikan migrasi yang dibutuhkan sudah diterapkan. Integrasi
// pembayaran tidak boleh tampak aktif ketika worker sebenarnya hanya akan
// gagal pada setiap tick karena tabel/kolom belum ada.
func (s *Service) Validate(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("database Shopee QRIS tidak dikonfigurasi")
	}
	var count int64
	err := s.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM information_schema.columns
		WHERE table_schema = current_schema()
		  AND table_name = 'shopee_qris_charges'
		  AND column_name IN ('settled_at', 'settlement_attempts', 'settlement_error', 'settlement_locked_until')
	`).Scan(&count).Error
	if err != nil {
		return err
	}
	if count != 4 {
		return fmt.Errorf("migrasi Shopee QRIS belum lengkap; jalankan go run ./cmd/migrate up")
	}
	return nil
}

func validEntity(value string) bool {
	switch value {
	case EntityEventOrder, EntityDonation, EntityShopOrder:
		return true
	}
	return false
}

// CreateCharge mengambil charge aktif yang sama atau mengalokasikan kode unik
// baru. Advisory lock per nominal dasar menutup celah antar proses, sedangkan
// index unik payable_amount menjadi pagar terakhir pada tingkat database.
func (s *Service) CreateCharge(c *gin.Context, entityType, entityID string, baseAmount int64, expiresAt time.Time) (*Charge, error) {
	if s == nil || s.db == nil || s.shopee == nil {
		return nil, fmt.Errorf("Shopee QRIS provider is not configured")
	}
	if !validEntity(entityType) || strings.TrimSpace(entityID) == "" || baseAmount <= 0 {
		return nil, fmt.Errorf("invalid Shopee QRIS charge")
	}
	now := time.Now().UTC()
	if expiresAt.IsZero() || !expiresAt.After(now) {
		expiresAt = now.Add(defaultLifetime)
	}

	var result Charge
	err := s.db.WithContext(requestContext(c)).Transaction(func(tx *gorm.DB) error {
		// Seluruh allocator untuk base amount yang sama harus berbaris, termasuk
		// bila aplikasi berjalan di beberapa instance.
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", baseAmount).Error; err != nil {
			return err
		}
		if err := tx.Model(&Charge{}).
			Where("status = ? AND expires_at <= ?", StatusPending, now).
			Updates(map[string]any{"status": StatusExpired, "updated_at": now}).Error; err != nil {
			return err
		}

		var existing Charge
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("entity_type = ? AND entity_id = ?", entityType, entityID).First(&existing).Error
		if err == nil && existing.Status == StatusPaid {
			result = existing
			return nil
		}
		if err == nil && existing.Status == StatusPending && existing.BaseAmount == baseAmount && existing.ExpiresAt.After(now) {
			result = existing
			return nil
		}
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		for attempt := 0; attempt < 128; attempt++ {
			code, errCode := randomCode()
			if errCode != nil {
				return errCode
			}
			payable := baseAmount + int64(code)
			var count int64
			if errCheck := tx.Model(&Charge{}).
				Where("payable_amount = ? AND (status = ? OR (status = ? AND paid_at > ?))", payable, StatusPending, StatusPaid, now.Add(-amountHold)).
				Count(&count).Error; errCheck != nil {
				return errCheck
			}
			if count > 0 {
				continue
			}
			payload, errQR := s.shopee.DynamicQRIS(payable)
			if errQR != nil {
				return errQR
			}

			candidate := Charge{
				ID: uuid.NewString(), EntityType: entityType, EntityID: entityID,
				BaseAmount: baseAmount, UniqueCode: code, PayableAmount: payable,
				QRISPayload: payload, Status: StatusPending, ExpiresAt: expiresAt,
				CreatedAt: now, UpdatedAt: now,
			}
			if err == nil { // baris entitas lama dipakai ulang setelah expired/cancelled
				candidate.ID = existing.ID
				candidate.CreatedAt = now
				update := map[string]any{
					"base_amount": baseAmount, "unique_code": code, "payable_amount": payable,
					"qris_payload": payload, "status": StatusPending, "expires_at": expiresAt,
					"shopee_transaction_id": nil, "paid_at": nil, "settled_at": nil,
					"settlement_attempts": 0, "settlement_error": nil,
					"settlement_locked_until": nil, "created_at": now, "updated_at": now,
				}
				if updateErr := tx.Model(&Charge{}).Where("id = ? AND status <> ?", existing.ID, StatusPaid).Updates(update).Error; updateErr != nil {
					if isUniqueViolation(updateErr) {
						continue
					}
					return updateErr
				}
			} else if createErr := tx.Create(&candidate).Error; createErr != nil {
				if isUniqueViolation(createErr) {
					continue
				}
				return createErr
			}
			result = candidate
			return nil
		}
		return fmt.Errorf("tidak ada kode unik QRIS yang tersedia")
	})
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// LookupCharge membaca charge yang sudah diterbitkan untuk satu transaksi.
// Nil mengartikan transaksi tersebut memang bukan charge QRIS Shopee.
func (s *Service) LookupCharge(c *gin.Context, entityType, entityID string) (*Charge, error) {
	if s == nil || s.db == nil || !validEntity(entityType) || strings.TrimSpace(entityID) == "" {
		return nil, nil
	}
	var charge Charge
	tx := s.db.WithContext(requestContext(c)).
		Where("entity_type = ? AND entity_id = ?", entityType, entityID).
		Limit(1).
		Find(&charge)
	if tx.Error != nil {
		return nil, tx.Error
	}
	if tx.RowsAffected == 0 {
		return nil, nil
	}
	return &charge, nil
}

func randomCode() (int, error) {
	var raw [8]byte
	if _, err := cryptorand.Read(raw[:]); err != nil {
		return 0, err
	}
	return int(binary.BigEndian.Uint64(raw[:])%999) + 1, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func requestContext(c *gin.Context) context.Context {
	if c != nil && c.Request != nil {
		return c.Request.Context()
	}
	return context.Background()
}

// Start menjalankan pemeriksaan segera lalu berkala sampai context dihentikan.
func (s *Service) Start(ctx context.Context) {
	if s == nil || s.db == nil || s.shopee == nil || s.settler == nil || s.eligible == nil {
		log.Printf("[shopee-qris] worker tidak aktif: konfigurasi belum lengkap")
		return
	}
	s.shopee.SetAutoValidation(s.shopee.Configured())
	go func() {
		defer s.shopee.SetAutoValidation(false)
		s.run(ctx)
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.run(ctx)
			}
		}
	}()
	log.Printf("[shopee-qris] auto-validasi aktif setiap %s", s.interval)
}

func (s *Service) run(ctx context.Context) {
	s.shopee.SetAutoValidation(s.shopee.Configured())
	if err := s.Reconcile(ctx); err != nil {
		log.Printf("[shopee-qris] rekonsiliasi gagal: %v", err)
	}
}

// Reconcile dapat dipanggil test/operasional dan merupakan satu putaran worker.
func (s *Service) Reconcile(ctx context.Context) error {
	now := time.Now().UTC()
	var pending []Charge
	// Grace lima menit membuat pembayaran yang masuk tepat sebelum kedaluwarsa
	// tetap ditemukan bila satu tick worker terlambat atau upstream sempat gagal.
	if err := s.db.WithContext(ctx).Where("status = ? AND expires_at > ?", StatusPending, now.Add(-5*time.Minute)).
		Order("created_at ASC").Find(&pending).Error; err != nil {
		return err
	}
	if len(pending) > 0 {
		start := pending[0].CreatedAt.Add(-2 * time.Minute)
		transactions, err := s.shopee.SuccessfulTransactions(ctx, start, now)
		if err != nil {
			return err
		}
		for index := range pending {
			charge := &pending[index]
			allowed, eligibleErr := s.eligible(workerGinContext(ctx), charge.EntityType, charge.EntityID)
			if eligibleErr != nil {
				return eligibleErr
			}
			if !allowed {
				if errCancel := s.db.WithContext(ctx).Model(&Charge{}).
					Where("id = ? AND status = ?", charge.ID, StatusPending).
					Updates(map[string]any{"status": StatusCancelled, "updated_at": now}).Error; errCancel != nil {
					return errCancel
				}
				continue
			}
			for _, mutation := range transactions {
				if !matchesMutation(charge, mutation) {
					continue
				}
				claimed, errClaim := s.claim(ctx, charge.ID, mutation.ID, mutation.CreateTime)
				if errClaim != nil {
					return errClaim
				}
				if claimed {
					break
				}
			}
		}
	}
	if err := s.db.WithContext(ctx).Model(&Charge{}).
		Where("status = ? AND expires_at <= ?", StatusPending, now).
		Updates(map[string]any{"status": StatusExpired, "updated_at": now}).Error; err != nil {
		return err
	}
	return s.settleReady(ctx)
}

func matchesMutation(charge *Charge, mutation shopee.SuccessfulTransaction) bool {
	if charge == nil || mutation.ID == "" || mutation.Amount != charge.PayableAmount {
		return false
	}
	return !mutation.CreateTime.Before(charge.CreatedAt.Add(-2*time.Minute)) &&
		!mutation.CreateTime.After(charge.ExpiresAt.Add(2*time.Minute))
}

func (s *Service) claim(ctx context.Context, chargeID, transactionID string, paidAt time.Time) (bool, error) {
	result := s.db.WithContext(ctx).Model(&Charge{}).
		Where("id = ? AND status = ? AND shopee_transaction_id IS NULL", chargeID, StatusPending).
		Where("NOT EXISTS (SELECT 1 FROM shopee_qris_charges used WHERE used.shopee_transaction_id = ?)", transactionID).
		Updates(map[string]any{
			"status": StatusPaid, "shopee_transaction_id": transactionID,
			"paid_at": paidAt, "updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		if isUniqueViolation(result.Error) {
			return false, nil
		}
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (s *Service) settleReady(ctx context.Context) error {
	for {
		charge, err := s.takeSettlement(ctx)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		if charge == nil || charge.ShopeeTransactionID == nil {
			return nil
		}
		ginCtx := workerGinContext(ctx)
		err = s.settler(ginCtx, charge.EntityType, charge.EntityID, charge.PayableAmount, *charge.ShopeeTransactionID)
		now := time.Now().UTC()
		updates := map[string]any{"settlement_locked_until": nil, "updated_at": now}
		if err == nil {
			updates["settled_at"] = now
			updates["settlement_error"] = nil
		} else {
			message := err.Error()
			updates["settlement_error"] = message
			// Jangan mengambil baris gagal yang sama terus-menerus dalam satu
			// putaran. Lease menjadi backoff dan percobaan berikutnya dilakukan
			// oleh tick worker selanjutnya.
			updates["settlement_locked_until"] = now.Add(30 * time.Second)
		}
		if updateErr := s.db.WithContext(ctx).Model(&Charge{}).Where("id = ?", charge.ID).Updates(updates).Error; updateErr != nil {
			return updateErr
		}
		if err != nil {
			log.Printf("[shopee-qris] settlement %s/%s gagal (akan dicoba ulang): %v", charge.EntityType, charge.EntityID, err)
		}
	}
}

// PaidTransaction dipakai endpoint legacy /check-payment. Hanya mutasi yang
// telah diklaim persisten oleh charge yang boleh dilaporkan sebagai lunas.
func (s *Service) PaidTransaction(ctx context.Context, amount int64, start time.Time) (*shopee.SuccessfulTransaction, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("Shopee QRIS provider is not configured")
	}
	var charge Charge
	err := s.db.WithContext(ctx).
		Where("status = ? AND payable_amount = ? AND paid_at >= ? AND shopee_transaction_id IS NOT NULL", StatusPaid, amount, start).
		Order("paid_at DESC").First(&charge).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	paidAt := charge.CreatedAt
	if charge.PaidAt != nil {
		paidAt = *charge.PaidAt
	}
	return &shopee.SuccessfulTransaction{
		ID: *charge.ShopeeTransactionID, DisplayID: *charge.ShopeeTransactionID,
		Amount: charge.PayableAmount, CreateTime: paidAt,
	}, nil
}

func (s *Service) takeSettlement(ctx context.Context) (*Charge, error) {
	now := time.Now().UTC()
	lockedUntil := now.Add(settlementLease)
	var charge Charge
	err := s.db.WithContext(ctx).Raw(`
		UPDATE shopee_qris_charges
		SET settlement_locked_until = ?, settlement_attempts = settlement_attempts + 1, updated_at = ?
		WHERE id = (
			SELECT id FROM shopee_qris_charges
			WHERE status = ? AND settled_at IS NULL
			  AND (settlement_locked_until IS NULL OR settlement_locked_until < ?)
			ORDER BY paid_at ASC
			FOR UPDATE SKIP LOCKED
			LIMIT 1
		)
		RETURNING *`, lockedUntil, now, StatusPaid, now).Scan(&charge).Error
	if err != nil {
		return nil, err
	}
	if charge.ID == "" {
		return nil, gorm.ErrRecordNotFound
	}
	return &charge, nil
}

func workerGinContext(ctx context.Context) *gin.Context {
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, "http://worker.local/", nil)
	return &gin.Context{Request: request}
}
