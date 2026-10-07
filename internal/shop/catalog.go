package shop

import (
	"strings"
	"unicode"
)

// Validasi katalog — seluruhnya fungsi murni, diuji tanpa database.
//
// Batas panjang ditegakkan dua kali: di sini untuk pesan galat yang ramah, dan
// di CHECK constraint migrasi 0012 sebagai jaminan terakhir yang tetap berlaku
// walau ada jalur tulis baru kelak.

// Batas panjang yang sama dengan CHECK constraint pada tabelnya.
const (
	nameMin = 3
	nameMax = 140
	slugMax = 160
)

// NormalizeSlug merapikan judul menjadi slug yang aman dipakai di URL.
//
// Huruf besar diturunkan, setiap deret karakter yang bukan huruf/angka menjadi
// satu tanda hubung, dan tanda hubung di ujung dibuang. Hasilnya selalu
// memenuhi pola yang ditegakkan CHECK constraint products.slug.
func NormalizeSlug(value string) string {
	var b strings.Builder
	lastDash := true // true di awal: tanda hubung di depan langsung dibuang

	for _, r := range strings.ToLower(strings.TrimSpace(value)) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}

	slug := b.String()
	slug = strings.Trim(slug, "-")
	if len(slug) > slugMax {
		// Pemotongan bisa menyisakan tanda hubung di ujung, jadi dibersihkan
		// lagi — slug yang berakhir dengan '-' tidak lolos constraint.
		slug = strings.Trim(slug[:slugMax], "-")
	}
	return slug
}

// ValidateSlug memeriksa bentuk slug yang sudah dinormalkan.
func ValidateSlug(slug string) error {
	if strings.TrimSpace(slug) == "" {
		return invalid("slug wajib diisi")
	}
	if len(slug) > slugMax {
		return invalid("slug maksimum %d karakter", slugMax)
	}
	if strings.HasPrefix(slug, "-") || strings.HasSuffix(slug, "-") || strings.Contains(slug, "--") {
		return invalid("slug tidak boleh mengandung tanda hubung ganda atau di ujung")
	}
	for _, r := range slug {
		if r == '-' || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			continue
		}
		return invalid("slug hanya boleh berisi huruf kecil, angka, dan tanda hubung")
	}
	return nil
}

// ValidateProduct memeriksa nama, slug, dan daftar foto produk.
func ValidateProduct(name, slug string, imageURLs []string) error {
	trimmed := strings.TrimSpace(name)
	if len([]rune(trimmed)) < nameMin {
		return invalid("nama produk minimum %d karakter", nameMin)
	}
	if len([]rune(trimmed)) > nameMax {
		return invalid("nama produk maksimum %d karakter", nameMax)
	}
	if err := ValidateSlug(slug); err != nil {
		return err
	}
	for _, url := range imageURLs {
		if strings.TrimSpace(url) == "" {
			return invalid("url foto tidak boleh kosong")
		}
	}
	return nil
}

// ValidateVariant memeriksa satu varian.
//
// Varian tanpa pembeda apa pun ditolak. Bukan karena teknis — melainkan karena
// pembeli tidak dapat membedakannya dari varian lain, dan unique index
// (product_id, size, color) akan menolaknya juga dengan pesan yang jauh lebih
// membingungkan.
func ValidateVariant(size, color string, price int64, stock int) error {
	if strings.TrimSpace(size) == "" && strings.TrimSpace(color) == "" {
		return invalid("varian harus punya ukuran atau warna")
	}
	if len([]rune(strings.TrimSpace(size))) > 32 {
		return invalid("ukuran maksimum 32 karakter")
	}
	if len([]rune(strings.TrimSpace(color))) > 32 {
		return invalid("warna maksimum 32 karakter")
	}
	if price < 0 {
		return invalid("harga tidak boleh negatif")
	}
	if stock < 0 {
		return invalid("stok tidak boleh negatif")
	}
	return nil
}

// ValidateVariantStatus memastikan status varian dikenal.
func ValidateVariantStatus(status string) error {
	if status != VariantActive && status != VariantInactive {
		return invalid("status varian tidak dikenal: %s", status)
	}
	return nil
}

// VariantLabel menyusun label varian yang terbaca manusia.
//
// Hasilnya deterministik dan sengaja sederhana: "L", "Merah", "L / Merah",
// atau string kosong bila keduanya kosong. Label ini juga yang disimpan sebagai
// snapshot di shop_order_items, sehingga riwayat pesanan tetap terbaca walau
// variannya sudah dihapus.
func VariantLabel(size, color string) string {
	size = strings.TrimSpace(size)
	color = strings.TrimSpace(color)

	switch {
	case size == "" && color == "":
		return ""
	case size == "":
		return color
	case color == "":
		return size
	default:
		return size + " / " + color
	}
}

// NormalizeImageURLs membuang url kosong dan merapikan spasi, supaya galeri
// tidak menyimpan entri yang tidak dapat ditampilkan.
func NormalizeImageURLs(urls []string) []string {
	out := make([]string, 0, len(urls))
	for _, url := range urls {
		trimmed := strings.TrimSpace(url)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}
