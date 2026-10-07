package shop

import (
	"strings"
	"testing"
)

func TestNormalizeSlug(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"huruf besar diturunkan", "Kaos YukNgaji", "kaos-yukngaji"},
		{"spasi ganda jadi satu tanda hubung", "Kaos   YukNgaji", "kaos-yukngaji"},
		{"tanda baca dibuang", "Kaos (Edisi Spesial)!", "kaos-edisi-spesial"},
		{"spasi di ujung dibuang", "   kaos   ", "kaos"},
		{"tanda hubung di ujung dibuang", "--kaos--", "kaos"},
		{"angka dipertahankan", "Kaos 2026", "kaos-2026"},
		{"string kosong", "", ""},
		{"hanya tanda baca", "!!!", ""},
		{"hanya spasi", "   ", ""},
		{"garis bawah dan titik jadi tanda hubung", "kaos_premium.v2", "kaos-premium-v2"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeSlug(tc.input); got != tc.want {
				t.Errorf("NormalizeSlug(%q) = %q, ingin %q", tc.input, got, tc.want)
			}
		})
	}

	t.Run("hasilnya selalu lolos ValidateSlug", func(t *testing.T) {
		for _, input := range []string{
			"Kaos YukNgaji", "  --Aneh--  ", "kaos_premium.v2", "A", "2026",
		} {
			got := NormalizeSlug(input)
			if got == "" {
				continue
			}
			if err := ValidateSlug(got); err != nil {
				t.Errorf("ValidateSlug(NormalizeSlug(%q)) = %v, ingin nil (slug %q)", input, err, got)
			}
		}
	})

	t.Run("slug sangat panjang dipotong tanpa tanda hubung di ujung", func(t *testing.T) {
		got := NormalizeSlug(strings.Repeat("a", 200))
		if len(got) > slugMax {
			t.Errorf("panjang = %d, ingin maksimum %d", len(got), slugMax)
		}
		if strings.HasSuffix(got, "-") {
			t.Errorf("slug = %q, tidak boleh berakhir dengan tanda hubung", got)
		}
	})
}

func TestValidateSlug(t *testing.T) {
	cases := []struct {
		name    string
		slug    string
		wantErr bool
	}{
		{"slug sah", "kaos-yukngaji", false},
		{"slug satu kata", "kaos", false},
		{"slug berangka", "kaos-2026", false},
		{"kosong", "", true},
		{"hanya spasi", "   ", true},
		{"huruf besar", "Kaos", true},
		{"tanda hubung ganda", "kaos--yukngaji", true},
		{"tanda hubung di depan", "-kaos", true},
		{"tanda hubung di ujung", "kaos-", true},
		{"spasi di tengah", "kaos yukngaji", true},
		{"garis bawah", "kaos_yukngaji", true},
		{"terlalu panjang", strings.Repeat("a", slugMax+1), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateSlug(tc.slug)
			if tc.wantErr && err == nil {
				t.Fatal("error = nil, ingin galat validasi")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("error = %v, ingin nil", err)
			}
		})
	}
}

func TestValidateProduct(t *testing.T) {
	cases := []struct {
		name      string
		product   string
		slug      string
		imageURLs []string
		wantErr   bool
	}{
		{"sah", "Kaos YukNgaji", "kaos-yukngaji", nil, false},
		{"nama 2 karakter", "Ka", "kaos", nil, true},
		{"nama 3 karakter", "Kao", "kaos", nil, false},
		{"nama 140 karakter", strings.Repeat("a", 140), "kaos", nil, false},
		{"nama 141 karakter", strings.Repeat("a", 141), "kaos", nil, true},
		{"nama hanya spasi", "   ", "kaos", nil, true},
		{"slug kosong", "Kaos", "", nil, true},
		{"slug tidak sah", "Kaos", "Kaos Besar", nil, true},
		{"foto kosong", "Kaos", "kaos", []string{""}, true},
		{"foto hanya spasi", "Kaos", "kaos", []string{"   "}, true},
		{"foto sah", "Kaos", "kaos", []string{"https://example.test/a.png"}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateProduct(tc.product, tc.slug, tc.imageURLs)
			if tc.wantErr && err == nil {
				t.Fatal("error = nil, ingin galat validasi")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("error = %v, ingin nil", err)
			}
		})
	}
}

func TestValidateVariant(t *testing.T) {
	cases := []struct {
		name    string
		size    string
		color   string
		price   int64
		stock   int
		wantErr bool
	}{
		{"ukuran saja", "L", "", 75_000, 10, false},
		{"warna saja", "", "Merah", 75_000, 10, false},
		{"keduanya", "L", "Merah", 75_000, 10, false},
		{"harga nol", "L", "", 0, 10, false},
		{"stok nol", "L", "", 75_000, 0, false},
		{"tanpa pembeda", "", "", 75_000, 10, true},
		{"pembeda hanya spasi", "  ", "   ", 75_000, 10, true},
		{"harga negatif", "L", "", -1, 10, true},
		{"stok negatif", "L", "", 75_000, -1, true},
		{"ukuran terlalu panjang", strings.Repeat("a", 33), "", 75_000, 10, true},
		{"warna terlalu panjang", "", strings.Repeat("a", 33), 75_000, 10, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateVariant(tc.size, tc.color, tc.price, tc.stock)
			if tc.wantErr && err == nil {
				t.Fatal("error = nil, ingin galat validasi")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("error = %v, ingin nil", err)
			}
		})
	}
}

func TestValidateVariantStatus(t *testing.T) {
	if err := ValidateVariantStatus(VariantActive); err != nil {
		t.Errorf("status active = %v, ingin nil", err)
	}
	if err := ValidateVariantStatus(VariantInactive); err != nil {
		t.Errorf("status inactive = %v, ingin nil", err)
	}
	if err := ValidateVariantStatus(""); err == nil {
		t.Error("status kosong = nil, ingin galat")
	}
	if err := ValidateVariantStatus("entah"); err == nil {
		t.Error("status tak dikenal = nil, ingin galat")
	}
}

func TestVariantLabel(t *testing.T) {
	cases := []struct {
		name  string
		size  string
		color string
		want  string
	}{
		{"ukuran saja", "L", "", "L"},
		{"warna saja", "", "Merah", "Merah"},
		{"keduanya", "L", "Merah", "L / Merah"},
		{"keduanya kosong", "", "", ""},
		{"spasi berlebih dirapikan", "  L  ", "  Merah  ", "L / Merah"},
		{"hanya spasi", "   ", "   ", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := VariantLabel(tc.size, tc.color); got != tc.want {
				t.Errorf("VariantLabel(%q, %q) = %q, ingin %q", tc.size, tc.color, got, tc.want)
			}
		})
	}
}

func TestNormalizeImageURLs(t *testing.T) {
	got := NormalizeImageURLs([]string{"  https://a.test/1.png  ", "", "   ", "https://a.test/2.png"})
	assertStrings(t, got, []string{"https://a.test/1.png", "https://a.test/2.png"})

	if got := NormalizeImageURLs(nil); len(got) != 0 {
		t.Errorf("hasil = %v, ingin kosong", got)
	}
}
