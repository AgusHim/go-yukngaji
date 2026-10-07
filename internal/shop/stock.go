package shop

import (
	"sort"
	"strings"
	"time"
)

// Perhitungan stok dan keranjang — seluruhnya fungsi murni, diuji tanpa
// database. Inilah tempat aturan overselling dapat dibuktikan tanpa
// menjalankan Postgres sekalipun.

// AvailableStock menghitung stok yang benar-benar bisa dijual.
//
// Hasilnya tidak pernah negatif. Stok yang ditahan lebih banyak dari stok fisik
// berarti ada ketidakcocokan, dan menampilkan angka negatif ke pembeli lebih
// buruk daripada menampilkan nol.
func AvailableStock(stock, reserved int) int {
	available := stock - reserved
	if available < 0 {
		return 0
	}
	return available
}

// NormalizeCart menggabungkan baris varian yang sama dan membuang qty nol.
//
// Ini bukan kerapian belaka: tanpa penggabungan, dua baris varian identik
// dengan qty 6 masing-masing akan lolos dari batas MaxQtyPerLine, dan
// perhitungan stok menjadi bergantung pada bentuk keranjang alih-alih pada
// jumlah yang benar-benar diminta.
//
// Urutan kemunculan pertama dipertahankan supaya pesan galat menyebut varian
// yang sama dengan yang dilihat pembeli.
func NormalizeCart(lines []CartLine) []CartLine {
	merged := make([]CartLine, 0, len(lines))
	index := map[string]int{}

	for _, line := range lines {
		variantID := strings.TrimSpace(line.VariantID)
		if variantID == "" || line.Qty <= 0 {
			continue
		}
		if at, ok := index[variantID]; ok {
			merged[at].Qty += line.Qty
			continue
		}
		index[variantID] = len(merged)
		merged = append(merged, CartLine{VariantID: variantID, Qty: line.Qty})
	}
	return merged
}

// ValidateCartLines memeriksa keranjang terhadap ketersediaan tiap varian.
//
// Dijalankan DI DALAM kunci advisory per varian: available harus dibaca setelah
// kuncinya dipegang, karena di luar kunci angka itu sudah basi sebelum
// sempat dipakai.
func ValidateCartLines(lines []CartLine, available map[string]int) error {
	if len(lines) == 0 {
		return invalid("keranjang kosong")
	}
	for _, line := range lines {
		if line.Qty <= 0 {
			return invalid("jumlah varian harus lebih dari nol")
		}
		if line.Qty > MaxQtyPerLine {
			return invalid("jumlah maksimum %d per varian", MaxQtyPerLine)
		}
		stock, known := available[line.VariantID]
		if !known {
			return invalid("varian tidak tersedia")
		}
		if stock < line.Qty {
			return invalid("stok tidak mencukupi untuk salah satu varian (tersisa %d, diminta %d)", stock, line.Qty)
		}
	}
	return nil
}

// CartTotal menghitung subtotal dari harga yang diambil server.
//
// prices dipetakan per varian. Varian yang tidak ada di peta dihitung nol —
// bukan karena itu keadaan yang sah (ValidateCartLines sudah menolaknya lebih
// dulu), melainkan supaya fungsi ini tidak pernah mengembalikan angka negatif
// atau panik karena peta yang tidak lengkap.
func CartTotal(lines []CartLine, prices map[string]int64) int64 {
	var total int64
	for _, line := range lines {
		if line.Qty <= 0 {
			continue
		}
		total += prices[line.VariantID] * int64(line.Qty)
	}
	return total
}

// ReservationOrdering mengurutkan varian untuk pengambilan kunci advisory.
//
// Urutannya menaik dan tidak pernah bergantung pada urutan baris keranjang.
// Itu bukan hiasan: dua checkout bersamaan yang berbagi dua varian dengan
// urutan berbeda akan saling menunggu secara melingkar bila kuncinya diambil
// menurut urutan keranjang — dan deadlock di jalur checkout berarti pembeli
// melihat kegagalan yang tidak dapat mereka pahami.
func ReservationOrdering(lines []CartLine) []string {
	ids := make([]string, 0, len(lines))
	seen := map[string]bool{}
	for _, line := range lines {
		if seen[line.VariantID] {
			continue
		}
		seen[line.VariantID] = true
		ids = append(ids, line.VariantID)
	}
	sort.Strings(ids)
	return ids
}

// OrderExpiry mengembalikan batas waktu tahanan stok untuk pesanan yang baru
// dibuat.
func OrderExpiry(now time.Time) time.Time {
	return now.Add(ReservationTTL)
}

// ShouldExpireOrder melaporkan apakah pesanan menunggu pembayaran sudah lewat
// batas waktunya.
//
// Hanya pesanan yang masih pending yang bisa kedaluwarsa. Pesanan yang sudah
// dibayar tidak pernah kedaluwarsa — uangnya sudah masuk, dan membatalkannya
// otomatis akan menghapus pesanan yang sah.
func ShouldExpireOrder(paymentStatus string, expiresAt, now time.Time) bool {
	if paymentStatus != PaymentPending {
		return false
	}
	return !now.Before(expiresAt)
}

// ValidateFulfillmentInput memeriksa isian penerima untuk cara pemenuhan yang
// dipilih.
//
// Dipanggil sebelum barisnya disimpan, supaya pesan galatnya jelas. Constraint
// chk_shop_orders_penerima tetap penjaga terakhirnya.
func ValidateFulfillmentInput(method, recipientName, recipientPhone, recipientAddress string) error {
	switch method {
	case MethodPickup:
		return nil
	case MethodShipping:
		if strings.TrimSpace(recipientName) == "" {
			return invalid("nama penerima wajib diisi untuk pengiriman")
		}
		if strings.TrimSpace(recipientPhone) == "" {
			return invalid("nomor telepon penerima wajib diisi untuk pengiriman")
		}
		if strings.TrimSpace(recipientAddress) == "" {
			return invalid("alamat pengiriman wajib diisi")
		}
		return nil
	default:
		return invalid("cara pemenuhan tidak dikenal: %s", method)
	}
}

// TrimOrNil mengubah string kosong menjadi nil, supaya klien dapat membedakan
// "tidak ada isi" dari "ada isi yang kebetulan kosong".
func TrimOrNil(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
