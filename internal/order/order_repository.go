package order

import (
	"fmt"
	"strings"
	"time"

	"mainyuk/internal/user_ticket"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type voucherRedemption struct {
	ID, VoucherID, OrderID, UserID string
	DiscountAmount                 int
	ReleasedAt                     *time.Time
	CreatedAt                      time.Time
}

func (voucherRedemption) TableName() string { return "voucher_redemptions" }

type repository struct {
	db *gorm.DB
}

func (r *repository) RedeemVoucher(c *gin.Context, orderID, eventID, userID, code string, subtotal int, now time.Time) (int, *string, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return 0, nil, nil
	}
	var discount int
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var v Voucher
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("target_type = ? AND target_id = ? AND code = ? AND is_active = true", "event", eventID, code).First(&v).Error; err != nil {
			return fmt.Errorf("voucher tidak ditemukan atau tidak aktif")
		}
		if (v.StartsAt != nil && now.Before(*v.StartsAt)) || (v.EndsAt != nil && !now.Before(*v.EndsAt)) {
			return fmt.Errorf("voucher belum atau sudah tidak berlaku")
		}
		if subtotal < v.MinOrderAmount {
			return fmt.Errorf("minimum transaksi voucher adalah Rp%d", v.MinOrderAmount)
		}
		if v.UsageLimit > 0 && v.UsedCount >= v.UsageLimit {
			return fmt.Errorf("kuota voucher sudah habis")
		}
		var used int64
		if err := tx.Model(&voucherRedemption{}).Where("voucher_id = ? AND user_id = ? AND released_at IS NULL", v.ID, userID).Count(&used).Error; err != nil {
			return err
		}
		if v.UsagePerUser > 0 && used >= int64(v.UsagePerUser) {
			return fmt.Errorf("batas penggunaan voucher sudah tercapai")
		}
		if v.DiscountType == "percentage" {
			discount = subtotal * v.DiscountValue / 100
			if v.MaxDiscount > 0 && discount > v.MaxDiscount {
				discount = v.MaxDiscount
			}
		} else {
			discount = v.DiscountValue
		}
		if discount > subtotal {
			discount = subtotal
		}
		if discount <= 0 {
			return fmt.Errorf("nilai voucher tidak valid")
		}
		if err := tx.Model(&Voucher{}).Where("id = ?", v.ID).Updates(map[string]interface{}{"used_count": v.UsedCount + 1}).Error; err != nil {
			return err
		}
		return tx.Create(&voucherRedemption{ID: uuid.NewString(), VoucherID: v.ID, OrderID: orderID, UserID: userID, DiscountAmount: discount, CreatedAt: now}).Error
	})
	if err != nil {
		return 0, nil, err
	}
	return discount, &code, nil
}
func (r *repository) ReleaseVoucher(c *gin.Context, orderID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var red voucherRedemption
		if err := tx.Where("order_id = ? AND released_at IS NULL", orderID).First(&red).Error; err != nil {
			return nil
		}
		now := time.Now()
		if err := tx.Model(&voucherRedemption{}).Where("id = ?", red.ID).Updates(map[string]interface{}{"released_at": now}).Error; err != nil {
			return err
		}
		return tx.Model(&Voucher{}).Where("id = ? AND used_count > 0", red.VoucherID).Update("used_count", gorm.Expr("used_count - 1")).Error
	})
}

func (r *repository) ListVouchers(c *gin.Context) ([]*Voucher, error) {
	var items []*Voucher
	query := r.db.Order("created_at DESC")
	if targetType := c.Query("target_type"); targetType != "" {
		query = query.Where("target_type = ?", targetType)
	}
	if err := query.Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *repository) CreateVoucher(c *gin.Context, input *VoucherInput) (*Voucher, error) {
	v := &Voucher{ID: uuid.NewString(), TargetType: input.TargetType, TargetID: input.TargetID, Code: strings.ToUpper(strings.TrimSpace(input.Code)), DiscountType: input.DiscountType, DiscountValue: input.DiscountValue, MaxDiscount: input.MaxDiscount, MinOrderAmount: input.MinOrderAmount, UsageLimit: input.UsageLimit, UsagePerUser: input.UsagePerUser, StartsAt: input.StartsAt, EndsAt: input.EndsAt, IsActive: input.IsActive}
	if err := r.db.Create(v).Error; err != nil {
		return nil, err
	}
	return v, nil
}

func (r *repository) UpdateVoucher(c *gin.Context, id string, input *VoucherInput) (*Voucher, error) {
	var v Voucher
	if err := r.db.Where("id = ?", id).First(&v).Error; err != nil {
		return nil, err
	}
	updates := map[string]interface{}{"target_type": input.TargetType, "target_id": input.TargetID, "code": strings.ToUpper(strings.TrimSpace(input.Code)), "discount_type": input.DiscountType, "discount_value": input.DiscountValue, "max_discount": input.MaxDiscount, "min_order_amount": input.MinOrderAmount, "usage_limit": input.UsageLimit, "usage_per_user": input.UsagePerUser, "starts_at": input.StartsAt, "ends_at": input.EndsAt, "is_active": input.IsActive, "updated_at": time.Now()}
	if err := r.db.Model(&v).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := r.db.Where("id = ?", id).First(&v).Error; err != nil {
		return nil, err
	}
	return &v, nil
}

// DeleteVoucher hanya menghapus voucher yang belum pernah dipakai agar jejak
// diskon pada order tetap utuh. Voucher yang sudah dipakai dapat dinonaktifkan.
func (r *repository) DeleteVoucher(c *gin.Context, id string) error {
	var used int64
	if err := r.db.Model(&voucherRedemption{}).Where("voucher_id = ?", id).Count(&used).Error; err != nil {
		return err
	}
	if used > 0 {
		return fmt.Errorf("voucher yang sudah digunakan tidak dapat dihapus; nonaktifkan voucher ini")
	}
	return r.db.Delete(&Voucher{}, "id = ?", id).Error
}

func NewRepository(db *gorm.DB) Repository {
	return &repository{
		db: db,
	}
}

// CreateWithTickets menyimpan order dan tiket-tiketnya secara atomik.
// Bila salah satu tiket gagal disimpan, order pun dibatalkan sehingga tidak
// ada order yang tersisa tanpa tiket.
func (r *repository) CreateWithTickets(c *gin.Context, order *Order, tickets []*user_ticket.UserTicket) (*Order, error) {
	err := r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		for _, t := range tickets {
			if err := tx.Create(t).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *repository) Show(c *gin.Context, id string) (*Order, error) {
	order := &Order{}
	tx := r.db
	query := tx.Model(&order)
	err := query.Preload("User").Preload("Event").Preload("Event.Divisi").Preload("PaymentMethod").Where("id = ?", id).First(&order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *repository) ShowByPublicID(c *gin.Context, public_id string, user_id *string) (*Order, error) {
	order := &Order{}
	tx := r.db
	query := tx.Model(&order)
	if user_id != nil {
		query.Where("user_id = ?", user_id)
	}
	err := query.Preload("User").Preload("Event").Preload("Event.Divisi").Preload("PaymentMethod").Where("public_id = ?", public_id).First(&order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

func (r *repository) Index(c *gin.Context, user_id *string) ([]*Order, error) {
	var order []*Order
	tx := r.db
	query := tx.Model(&Order{})
	if user_id != nil && *user_id != "" {
		query.Where("user_id = ?", user_id)
	}
	// Filter status id
	status := c.Query("status")
	if status != "" {
		query.Where("status = ?", strings.ToLower(status))
	}
	// Filter event_id
	event_id := c.Query("event_id")
	if event_id != "" {
		query.Where("event_id = ?", event_id)
	}

	err := query.Preload("PaymentMethod").Preload("UserTickets").Preload("User").Preload("User.Province").Preload("User.District").Preload("User.SubDistrict").Preload("Event").Preload("Event.Divisi").Preload("PaymentMethod").Order("created_at DESC").Find(&order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

// Update hanya menulis kolom status. Sebelumnya dipakai Save yang menulis
// seluruh baris dan ikut menyentuh asosiasi (user/event) yang di-preload,
// sehingga berisiko menimpa data yang tidak dimaksud.
func (r *repository) Update(c *gin.Context, order *Order) (*Order, error) {
	err := r.db.Model(&Order{}).Where("id = ?", order.ID).Updates(map[string]interface{}{
		"status":     order.Status,
		"updated_at": time.Now(),
	}).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}

// TransitionStatus memindahkan status hanya bila status saat ini masih `from`.
//
// Syarat itu ada di klausa WHERE, bukan di kode Go, sehingga dua permintaan
// verifikasi yang datang bersamaan tidak bisa sama-sama lolos: yang kedua
// tidak menemukan baris dan melaporkan false.
func (r *repository) TransitionStatus(c *gin.Context, id string, from string, to string) (bool, error) {
	res := r.db.Model(&Order{}).
		Where("id = ?", id).
		Where("status = ?", from).
		Updates(map[string]interface{}{
			"status":     to,
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

// ExpirePending melakukan transisi pending -> expired langsung di database.
// Kondisi batas waktu berada di WHERE agar pembayaran yang sudah dikonfirmasi
// bersamaan tidak pernah tertimpa oleh worker kedaluwarsa.
func (r *repository) ExpirePending(c *gin.Context, now time.Time) (int64, error) {
	if err := r.db.Exec(`WITH released AS (UPDATE voucher_redemptions vr SET released_at = ? FROM orders o WHERE vr.order_id = o.id AND vr.released_at IS NULL AND o.status = ? AND o.expired_at <= ? RETURNING vr.voucher_id) UPDATE vouchers v SET used_count = GREATEST(0, v.used_count - counts.total) FROM (SELECT voucher_id, COUNT(*)::integer AS total FROM released GROUP BY voucher_id) counts WHERE v.id = counts.voucher_id`, now, StatusPending, now).Error; err != nil {
		return 0, err
	}
	res := r.db.Model(&Order{}).
		Where("status = ?", StatusPending).
		Where("expired_at IS NOT NULL AND expired_at <= ?", now).
		Updates(map[string]interface{}{
			"status":     StatusExpired,
			"updated_at": now,
		})
	if res.Error != nil {
		return 0, res.Error
	}
	return res.RowsAffected, nil
}

func (r *repository) Participants(c *gin.Context, event_id string) ([]*Order, error) {
	var order []*Order
	tx := r.db
	query := tx.Model(&Order{})
	if event_id != "" {
		query.Where("event_id = ?", event_id)
	}
	err := query.Preload("UserTickets").Preload("User").Preload("User.Province").Preload("User.District").Preload("User.SubDistrict").Preload("Event").Preload("PaymentMethod").Where("status = ?", "paid").Order("created_at DESC").Find(&order).Error
	if err != nil {
		return nil, err
	}
	return order, nil
}
