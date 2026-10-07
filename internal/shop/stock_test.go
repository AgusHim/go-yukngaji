package shop

import (
	"testing"
	"time"
)

func TestAvailableStock(t *testing.T) {
	cases := []struct {
		name     string
		stock    int
		reserved int
		want     int
	}{
		{"stok lebih banyak dari tahanan", 10, 3, 7},
		{"stok sama dengan tahanan", 5, 5, 0},
		{"tahanan melebihi stok tidak pernah negatif", 5, 8, 0},
		{"tanpa tahanan", 4, 0, 4},
		{"keduanya nol", 0, 0, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := AvailableStock(tc.stock, tc.reserved); got != tc.want {
				t.Errorf("AvailableStock(%d, %d) = %d, ingin %d", tc.stock, tc.reserved, got, tc.want)
			}
		})
	}
}

func TestNormalizeCart(t *testing.T) {
	t.Run("menggabungkan varian yang sama", func(t *testing.T) {
		got := NormalizeCart([]CartLine{
			{VariantID: "v1", Qty: 2},
			{VariantID: "v2", Qty: 1},
			{VariantID: "v1", Qty: 3},
		})
		want := []CartLine{{VariantID: "v1", Qty: 5}, {VariantID: "v2", Qty: 1}}
		assertCart(t, got, want)
	})

	t.Run("membuang qty nol dan negatif", func(t *testing.T) {
		got := NormalizeCart([]CartLine{
			{VariantID: "v1", Qty: 0},
			{VariantID: "v2", Qty: -3},
			{VariantID: "v3", Qty: 1},
		})
		assertCart(t, got, []CartLine{{VariantID: "v3", Qty: 1}})
	})

	t.Run("membuang varian kosong", func(t *testing.T) {
		got := NormalizeCart([]CartLine{
			{VariantID: "", Qty: 2},
			{VariantID: "   ", Qty: 2},
		})
		if len(got) != 0 {
			t.Errorf("hasil = %v, ingin kosong", got)
		}
	})

	t.Run("merapikan spasi di sekitar id", func(t *testing.T) {
		got := NormalizeCart([]CartLine{
			{VariantID: " v1 ", Qty: 1},
			{VariantID: "v1", Qty: 2},
		})
		assertCart(t, got, []CartLine{{VariantID: "v1", Qty: 3}})
	})

	t.Run("urutan kemunculan pertama dipertahankan", func(t *testing.T) {
		got := NormalizeCart([]CartLine{
			{VariantID: "v3", Qty: 1},
			{VariantID: "v1", Qty: 1},
			{VariantID: "v3", Qty: 1},
		})
		assertCart(t, got, []CartLine{{VariantID: "v3", Qty: 2}, {VariantID: "v1", Qty: 1}})
	})

	t.Run("keranjang kosong", func(t *testing.T) {
		if got := NormalizeCart(nil); len(got) != 0 {
			t.Errorf("hasil = %v, ingin kosong", got)
		}
	})
}

func TestValidateCartLines(t *testing.T) {
	available := map[string]int{"v1": 5, "v2": 1}

	cases := []struct {
		name    string
		lines   []CartLine
		wantErr bool
	}{
		{"keranjang sah", []CartLine{{VariantID: "v1", Qty: 5}}, false},
		{"tepat di batas ketersediaan", []CartLine{{VariantID: "v2", Qty: 1}}, false},
		{"keranjang kosong", nil, true},
		{"qty nol", []CartLine{{VariantID: "v1", Qty: 0}}, true},
		{"qty negatif", []CartLine{{VariantID: "v1", Qty: -1}}, true},
		{"qty di atas batas baris", []CartLine{{VariantID: "v1", Qty: MaxQtyPerLine + 1}}, true},
		{"varian tak dikenal", []CartLine{{VariantID: "v9", Qty: 1}}, true},
		{"ketersediaan kurang pada varian pertama", []CartLine{{VariantID: "v1", Qty: 6}}, true},
		{"ketersediaan kurang pada varian kedua", []CartLine{
			{VariantID: "v1", Qty: 1},
			{VariantID: "v2", Qty: 2},
		}, true},
		{"dua varian yang keduanya cukup", []CartLine{
			{VariantID: "v1", Qty: 5},
			{VariantID: "v2", Qty: 1},
		}, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateCartLines(tc.lines, available)
			if tc.wantErr && err == nil {
				t.Fatal("error = nil, ingin galat validasi")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("error = %v, ingin nil", err)
			}
			if err != nil && !IsValidationError(err) {
				t.Errorf("error = %v, ingin memenuhi IsValidationError", err)
			}
		})
	}

	t.Run("varian dengan ketersediaan nol selalu ditolak", func(t *testing.T) {
		err := ValidateCartLines([]CartLine{{VariantID: "v3", Qty: 1}}, map[string]int{"v3": 0})
		if err == nil {
			t.Fatal("error = nil, ingin galat validasi")
		}
	})
}

func TestCartTotal(t *testing.T) {
	prices := map[string]int64{"v1": 75_000, "v2": 120_000}

	cases := []struct {
		name  string
		lines []CartLine
		want  int64
	}{
		{"satu baris", []CartLine{{VariantID: "v1", Qty: 2}}, 150_000},
		{"dua baris", []CartLine{{VariantID: "v1", Qty: 1}, {VariantID: "v2", Qty: 2}}, 315_000},
		{"keranjang kosong", nil, 0},
		{"qty nol tidak menambah", []CartLine{{VariantID: "v1", Qty: 0}}, 0},
		{"harga tak dikenal dihitung nol", []CartLine{{VariantID: "v9", Qty: 3}}, 0},
		{"harga nol", []CartLine{{VariantID: "v3", Qty: 5}}, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CartTotal(tc.lines, prices)
			if got != tc.want {
				t.Errorf("CartTotal(%v) = %d, ingin %d", tc.lines, got, tc.want)
			}
			if got < 0 {
				t.Errorf("total = %d, tidak boleh negatif", got)
			}
		})
	}
}

func TestReservationOrdering(t *testing.T) {
	t.Run("selalu menaik tanpa memperhatikan urutan keranjang", func(t *testing.T) {
		got := ReservationOrdering([]CartLine{
			{VariantID: "v9", Qty: 1},
			{VariantID: "v2", Qty: 1},
			{VariantID: "v5", Qty: 1},
		})
		assertStrings(t, got, []string{"v2", "v5", "v9"})
	})

	t.Run("varian kembar hanya sekali", func(t *testing.T) {
		got := ReservationOrdering([]CartLine{
			{VariantID: "v2", Qty: 1},
			{VariantID: "v2", Qty: 1},
		})
		assertStrings(t, got, []string{"v2"})
	})

	t.Run("keranjang kosong", func(t *testing.T) {
		if got := ReservationOrdering(nil); len(got) != 0 {
			t.Errorf("hasil = %v, ingin kosong", got)
		}
	})
}

func TestShouldExpireOrder(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Minute)
	future := now.Add(time.Minute)

	cases := []struct {
		name          string
		paymentStatus string
		expiresAt     time.Time
		want          bool
	}{
		{"pending lewat batas", PaymentPending, past, true},
		{"pending belum lewat batas", PaymentPending, future, false},
		{"pending tepat pada batas", PaymentPending, now, true},
		{"paid lewat batas tidak pernah kedaluwarsa", PaymentPaid, past, false},
		{"rejected lewat batas", PaymentRejected, past, false},
		{"refunded lewat batas", PaymentRefunded, past, false},
		{"status tak dikenal", "entah", past, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ShouldExpireOrder(tc.paymentStatus, tc.expiresAt, now); got != tc.want {
				t.Errorf("ShouldExpireOrder(%q, %v, now) = %v, ingin %v",
					tc.paymentStatus, tc.expiresAt, got, tc.want)
			}
		})
	}
}

func TestOrderExpiry(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	if got := OrderExpiry(now); !got.Equal(now.Add(ReservationTTL)) {
		t.Errorf("OrderExpiry = %v, ingin %v", got, now.Add(ReservationTTL))
	}
}

func TestValidateFulfillmentInput(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		nama    string
		telepon string
		alamat  string
		wantErr bool
	}{
		{"pickup tanpa isian apa pun", MethodPickup, "", "", "", false},
		{"kirim lengkap", MethodShipping, "Aisyah", "0812345678", "Jl. Slamet Riyadi 1", false},
		{"kirim tanpa nama", MethodShipping, "", "0812345678", "Jl. Slamet Riyadi 1", true},
		{"kirim tanpa telepon", MethodShipping, "Aisyah", "", "Jl. Slamet Riyadi 1", true},
		{"kirim tanpa alamat", MethodShipping, "Aisyah", "0812345678", "", true},
		{"kirim dengan nama hanya spasi", MethodShipping, "   ", "0812345678", "Jl. 1", true},
		{"cara tak dikenal", "entah", "Aisyah", "0812", "Jl. 1", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateFulfillmentInput(tc.method, tc.nama, tc.telepon, tc.alamat)
			if tc.wantErr && err == nil {
				t.Fatal("error = nil, ingin galat validasi")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("error = %v, ingin nil", err)
			}
		})
	}
}

func TestTrimOrNil(t *testing.T) {
	if got := TrimOrNil("  "); got != nil {
		t.Errorf("TrimOrNil(\"  \") = %v, ingin nil", *got)
	}
	if got := TrimOrNil(""); got != nil {
		t.Errorf("TrimOrNil(\"\") = %v, ingin nil", *got)
	}
	got := TrimOrNil("  resi-123  ")
	if got == nil || *got != "resi-123" {
		t.Errorf("TrimOrNil = %v, ingin \"resi-123\"", got)
	}
}

func assertCart(t *testing.T, got, want []CartLine) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("panjang = %d (%v), ingin %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("elemen %d = %+v, ingin %+v (seluruhnya %v)", i, got[i], want[i], got)
		}
	}
}
