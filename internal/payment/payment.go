// Package payment adalah satu-satunya pintu antara domain mana pun di aplikasi
// ini dan cara uang benar-benar berpindah.
//
// Paket ini lahir dari pemindahan seam yang sebelumnya tinggal di dalam
// internal/fundraising. Donasi adalah konsumen pertamanya, merchandise adalah
// konsumen keduanya, dan keduanya berbicara ke interface yang sama tanpa perlu
// saling mengenal — itulah sebabnya permintaan charge di sini tidak menyebut
// donasi maupun order, hanya nominal dan metode pembayaran.
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
	Provider     string        `json:"provider"`
	ExternalID   *string       `json:"external_id"`
	Amount       int           `json:"amount"`
	Instructions []Instruction `json:"instructions"`
	ExpiresAt    *time.Time    `json:"expires_at"`
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
// Bentuknya sengaja tidak menyebut donasi maupun order merchandise: itulah
// yang membuat satu adapter dapat dipakai kedua domain tanpa keduanya saling
// mengenal, dan tanpa provider perlu tahu apa pun tentang tabel pemanggilnya.
type ChargeRequest struct {
	Amount          int
	PaymentMethodID *string
}

// MethodResolver mengambil metode pembayaran untuk menyusun instruksi transfer.
type MethodResolver interface {
	Show(ctx *gin.Context, id string) (*payment_method.PaymentMethod, error)
}

// Provider adalah satu-satunya pintu antara domain dan cara uang berpindah.
//
// Yang dipasang saat ini hanya implementasi manual. Menambahkan gateway kelak
// berarti menambah satu berkas baru yang memenuhi interface ini, lalu
// mendaftarkannya di titik konstruksi — seluruh modul pemakainya tidak perlu
// berubah, karena semuanya hanya berbicara ke interface ini.
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
