// Package payment adalah satu-satunya pintu antara domain mana pun di aplikasi
// ini dan cara uang benar-benar berpindah.
//
// Paket ini lahir dari pemindahan seam yang sebelumnya tinggal di dalam
// internal/fundraising. Donasi adalah konsumen pertamanya, merchandise adalah
// konsumen keduanya, dan keduanya berbicara ke interface yang sama tanpa perlu
// saling mengenal. EntityType/EntityID hanya menjadi kunci korelasi provider;
// aturan domain tetap berada pada service pemilik transaksi.
package payment

import (
	"net/http"
	"time"

	"mainyuk/internal/payment_method"

	"github.com/gin-gonic/gin"
)

// Instruction adalah satu tujuan transfer yang ditampilkan ke pembayar.
type Instruction struct {
	MethodID      string  `json:"method_id"`
	Type          string  `json:"type"`
	Name          string  `json:"name"`
	AccountName   string  `json:"account_name"`
	AccountNumber string  `json:"account_number"`
	ImageURL      *string `json:"image_url"`
}

// Charge adalah instruksi pembayaran satu transaksi.
//
// ExternalID nil berarti pembayarannya tidak dikelola pihak ketiga — itulah
// keadaan provider manual, dan sebabnya pembayar menuliskan identitas
// transaksinya sebagai berita transfer.
type Charge struct {
	Provider      string        `json:"provider"`
	ExternalID    *string       `json:"external_id"`
	Amount        int           `json:"amount"`
	BaseAmount    int           `json:"base_amount,omitempty"`
	UniqueCode    int           `json:"unique_code,omitempty"`
	QRISPayload   string        `json:"qris_payload,omitempty"`
	PaymentStatus string        `json:"payment_status,omitempty"`
	Instructions  []Instruction `json:"instructions"`
	ExpiresAt     *time.Time    `json:"expires_at"`
}

// WebhookEvent adalah pembayaran yang dilaporkan penyedia pembayaran.
//
// Belum ada yang menghasilkannya sampai sekarang; bentuknya ada supaya gateway
// dapat ditambahkan tanpa mengubah tipe yang sudah beredar.
type WebhookEvent struct {
	ExternalID string
	DonationID string
	Status     string
	PaidAmount int
	Reference  *string
}

// ChargeRequest adalah keterangan minimum yang dibutuhkan provider untuk
// menyusun instruksi pembayaran.
//
// EntityType dan EntityID adalah identitas korelasi lintas domain. Provider
// tidak membaca tabel domain dan tidak menentukan status bisnisnya.
type ChargeRequest struct {
	Amount          int
	PaymentMethodID *string
	EntityType      string
	EntityID        string
	ExpiresAt       *time.Time
}

// MethodResolver mengambil metode pembayaran untuk menyusun instruksi transfer.
type MethodResolver interface {
	Show(ctx *gin.Context, id string) (*payment_method.PaymentMethod, error)
}

// Provider adalah satu-satunya pintu antara domain dan cara uang berpindah.
//
// Implementasi gabungan memilih QRIS Shopee untuk metode SHOPEE_QRIS dan
// mempertahankan metode pembayaran lain pada alur manual.
type Provider interface {
	// Name adalah pengenal provider, disimpan pada jejak audit.
	Name() string

	// CreateCharge menyusun instruksi pembayaran untuk satu transaksi.
	CreateCharge(ctx *gin.Context, req ChargeRequest) (*Charge, error)

	// ParseWebhook menerjemahkan panggilan balik penyedia pembayaran menjadi
	// peristiwa yang seragam. Provider manual mengembalikan
	// ErrWebhookUnsupported.
	ParseWebhook(ctx *gin.Context, body []byte, headers http.Header) (*WebhookEvent, error)
}
