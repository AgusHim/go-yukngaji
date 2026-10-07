package order

// Status order yang dikenal sistem.
const (
	StatusPending   = "pending"
	StatusPaid      = "paid"
	StatusExpired   = "expired"
	StatusCancelled = "cancelled"
	StatusFailed    = "failed"
	StatusRefunded  = "refunded"
)

// OrderAmounts memecah tagihan order menjadi komponennya.
//
// "TicketTotal" adalah subtotal tiket dan disimpan pada kolom orders.amount
// (semantik lama dipertahankan supaya tampilan pembayaran yang menjumlahkan
// amount + donasi + biaya tetap benar). "Total" adalah jumlah yang harus
// dibayar dan dipakai untuk menentukan status awal order.
type OrderAmounts struct {
	TicketTotal int
	Donation    int
	AdminFee    int
	Total       int
}

// ComputeOrderAmounts menghitung tagihan dari harga tiket, donasi, dan biaya
// admin. Komponen negatif diperlakukan sebagai nol. Fungsi murni: tidak
// menyentuh database sehingga bisa diuji langsung.
func ComputeOrderAmounts(ticketPrices []int, donation, adminFee int) OrderAmounts {
	ticketTotal := 0
	for _, price := range ticketPrices {
		if price > 0 {
			ticketTotal += price
		}
	}

	if donation < 0 {
		donation = 0
	}
	if adminFee < 0 {
		adminFee = 0
	}

	return OrderAmounts{
		TicketTotal: ticketTotal,
		Donation:    donation,
		AdminFee:    adminFee,
		Total:       ticketTotal + donation + adminFee,
	}
}

// DeriveStatus menentukan status awal sebuah order.
//
// Order hanya otomatis "paid" bila seluruh tagihan — tiket, donasi, dan biaya
// admin — benar-benar nol. Sebelumnya order bertiket gratis dengan donasi
// positif ikut ditandai "paid" sehingga donasinya tidak pernah tertagih.
func DeriveStatus(total int) string {
	if total <= 0 {
		return StatusPaid
	}
	return StatusPending
}
