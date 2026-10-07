package shop

import "strings"

// Mesin status pesanan dan produk.
//
// Seluruhnya fungsi murni dari string supaya aturannya dapat diuji tanpa
// database, dan supaya handler maupun UI tidak perlu meniru tabelnya. Service
// tetap penentu akhir: transisi dijalankan sebagai conditional UPDATE, jadi
// status yang tidak sah ditolak DB juga — bukan hanya oleh tabel ini.

// CanTransitionProduct melaporkan apakah perpindahan status produk sah.
//
// draft -> published -> archived -> draft: produk yang diarsipkan dapat
// diterbitkan kembali tanpa membuat baris baru, sehingga riwayat penjualannya
// tetap tersambung.
func CanTransitionProduct(from, to string) bool {
	if from == to {
		return false
	}
	switch from {
	case ProductDraft:
		return to == ProductPublished || to == ProductArchived
	case ProductPublished:
		return to == ProductDraft || to == ProductArchived
	case ProductArchived:
		return to == ProductDraft
	default:
		return false
	}
}

// CanTransitionPayment melaporkan apakah perpindahan status pembayaran sah.
//
// pending -> {paid, rejected}; paid -> refunded. rejected dan refunded
// terminal: pesanan yang uangnya sudah dikembalikan tidak boleh dibayar lagi,
// dan penolakan yang sudah terjadi tidak boleh dibatalkan diam-diam.
func CanTransitionPayment(from, to string) bool {
	if from == to {
		return false
	}
	switch from {
	case PaymentPending:
		return to == PaymentPaid || to == PaymentRejected
	case PaymentPaid:
		return to == PaymentRefunded
	default:
		return false
	}
}

// CanTransitionFulfillment melaporkan apakah perpindahan status pemenuhan sah.
//
// Pemenuhannya bergantung pada cara pemenuhannya: pesanan kirim tidak pernah
// melewati ready_for_pickup, dan pesanan pickup tidak pernah shipped. Tanpa
// pemisahan itu, UI harus menebak mana yang berlaku, dan pengurus bisa
// menandai pesanan pickup sebagai "dikirim" tanpa ada yang mengirim apa pun.
func CanTransitionFulfillment(method, from, to string) bool {
	if from == to {
		return false
	}
	if to == FulfillmentCancelled {
		// Pembatalan hanya masuk akal sebelum barang berpindah tangan.
		return from == FulfillmentUnfulfilled || from == FulfillmentReadyForPickup
	}
	switch method {
	case MethodPickup:
		switch from {
		case FulfillmentUnfulfilled:
			return to == FulfillmentReadyForPickup
		case FulfillmentReadyForPickup:
			return to == FulfillmentCompleted
		}
		return false
	case MethodShipping:
		switch from {
		case FulfillmentUnfulfilled:
			return to == FulfillmentShipped
		case FulfillmentShipped:
			return to == FulfillmentCompleted
		}
		return false
	default:
		return false
	}
}

// AllowedFulfillmentTargets mengembalikan status pemenuhan yang sah dari
// status sekarang. Dipakai UI untuk menampilkan hanya tombol yang mungkin,
// bukan untuk menegakkan aturan — penegakannya tetap di service dan DB.
func AllowedFulfillmentTargets(method, from string) []string {
	targets := []string{}
	for _, candidate := range []string{
		FulfillmentUnfulfilled,
		FulfillmentReadyForPickup,
		FulfillmentShipped,
		FulfillmentCompleted,
		FulfillmentCancelled,
	} {
		if CanTransitionFulfillment(method, from, candidate) {
			targets = append(targets, candidate)
		}
	}
	return targets
}

// IsCancellableFulfillment melaporkan apakah pemenuhan pesanan masih dapat
// dibatalkan. Dipakai untuk memutuskan apakah pembatalan mengembalikan stok.
func IsCancellableFulfillment(from string) bool {
	return from == FulfillmentUnfulfilled || from == FulfillmentReadyForPickup
}

// RequireReason memastikan alasan keputusan terisi.
//
// Dipakai ulang oleh konfirmasi, penolakan, refund, pembatalan, dan
// penyesuaian stok: kelimanya adalah keputusan pengurus yang harus dapat
// diaudit dari dua arah — dari barisnya sendiri dan dari jejak audit.
func RequireReason(reason string) error {
	if strings.TrimSpace(reason) == "" {
		return invalid("alasan wajib diisi")
	}
	return nil
}
