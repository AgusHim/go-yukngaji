package shop

import "testing"

func TestCanTransitionProduct(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"draft ke published", ProductDraft, ProductPublished, true},
		{"draft ke archived", ProductDraft, ProductArchived, true},
		{"published ke draft", ProductPublished, ProductDraft, true},
		{"published ke archived", ProductPublished, ProductArchived, true},
		{"archived ke draft", ProductArchived, ProductDraft, true},
		{"archived ke published ditolak", ProductArchived, ProductPublished, false},
		{"draft ke draft ditolak", ProductDraft, ProductDraft, false},
		{"published ke published ditolak", ProductPublished, ProductPublished, false},
		{"status tak dikenal", "entah", ProductDraft, false},
		{"tujuan tak dikenal", ProductDraft, "entah", false},
		{"dari kosong", "", ProductDraft, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanTransitionProduct(tc.from, tc.to); got != tc.want {
				t.Errorf("CanTransitionProduct(%q, %q) = %v, ingin %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestCanTransitionPayment(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"pending ke paid", PaymentPending, PaymentPaid, true},
		{"pending ke rejected", PaymentPending, PaymentRejected, true},
		{"pending ke refunded ditolak", PaymentPending, PaymentRefunded, false},
		{"paid ke refunded", PaymentPaid, PaymentRefunded, true},
		{"paid ke pending ditolak", PaymentPaid, PaymentPending, false},
		{"paid ke rejected ditolak", PaymentPaid, PaymentRejected, false},
		{"rejected terminal", PaymentRejected, PaymentPaid, false},
		{"refunded terminal", PaymentRefunded, PaymentPaid, false},
		{"refunded ke rejected ditolak", PaymentRefunded, PaymentRejected, false},
		{"from sama dengan to ditolak", PaymentPending, PaymentPending, false},
		{"status tak dikenal", "entah", PaymentPaid, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanTransitionPayment(tc.from, tc.to); got != tc.want {
				t.Errorf("CanTransitionPayment(%q, %q) = %v, ingin %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestCanTransitionFulfillment(t *testing.T) {
	cases := []struct {
		name   string
		method string
		from   string
		to     string
		want   bool
	}{
		{"pickup belum siap ke siap", MethodPickup, FulfillmentUnfulfilled, FulfillmentReadyForPickup, true},
		{"pickup siap ke selesai", MethodPickup, FulfillmentReadyForPickup, FulfillmentCompleted, true},
		{"pickup tidak pernah dikirim", MethodPickup, FulfillmentUnfulfilled, FulfillmentShipped, false},
		{"pickup tidak langsung selesai", MethodPickup, FulfillmentUnfulfilled, FulfillmentCompleted, false},
		{"kirim belum kirim ke dikirim", MethodShipping, FulfillmentUnfulfilled, FulfillmentShipped, true},
		{"kirim dikirim ke selesai", MethodShipping, FulfillmentShipped, FulfillmentCompleted, true},
		{"kirim tidak pernah siap diambil", MethodShipping, FulfillmentUnfulfilled, FulfillmentReadyForPickup, false},
		{"kirim tidak langsung selesai", MethodShipping, FulfillmentUnfulfilled, FulfillmentCompleted, false},
		{"batal dari belum diproses", MethodPickup, FulfillmentUnfulfilled, FulfillmentCancelled, true},
		{"batal dari siap diambil", MethodShipping, FulfillmentReadyForPickup, FulfillmentCancelled, true},
		{"batal setelah dikirim ditolak", MethodShipping, FulfillmentShipped, FulfillmentCancelled, false},
		{"batal setelah selesai ditolak", MethodPickup, FulfillmentCompleted, FulfillmentCancelled, false},
		{"selesai terminal", MethodPickup, FulfillmentCompleted, FulfillmentReadyForPickup, false},
		{"dibatalkan terminal", MethodShipping, FulfillmentCancelled, FulfillmentShipped, false},
		{"cara tak dikenal", "entah", FulfillmentUnfulfilled, FulfillmentShipped, false},
		{"from sama dengan to ditolak", MethodPickup, FulfillmentUnfulfilled, FulfillmentUnfulfilled, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanTransitionFulfillment(tc.method, tc.from, tc.to); got != tc.want {
				t.Errorf("CanTransitionFulfillment(%q, %q, %q) = %v, ingin %v",
					tc.method, tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestAllowedFulfillmentTargets(t *testing.T) {
	t.Run("pickup dari belum diproses", func(t *testing.T) {
		got := AllowedFulfillmentTargets(MethodPickup, FulfillmentUnfulfilled)
		want := []string{FulfillmentReadyForPickup, FulfillmentCancelled}
		assertStrings(t, got, want)
	})

	t.Run("kirim dari belum diproses", func(t *testing.T) {
		got := AllowedFulfillmentTargets(MethodShipping, FulfillmentUnfulfilled)
		want := []string{FulfillmentShipped, FulfillmentCancelled}
		assertStrings(t, got, want)
	})

	t.Run("terminal tidak punya tujuan", func(t *testing.T) {
		if got := AllowedFulfillmentTargets(MethodPickup, FulfillmentCompleted); len(got) != 0 {
			t.Errorf("tujuan = %v, ingin kosong", got)
		}
	})
}

func TestIsCancellableFulfillment(t *testing.T) {
	cases := []struct {
		from string
		want bool
	}{
		{FulfillmentUnfulfilled, true},
		{FulfillmentReadyForPickup, true},
		{FulfillmentShipped, false},
		{FulfillmentCompleted, false},
		{FulfillmentCancelled, false},
		{"entah", false},
	}
	for _, tc := range cases {
		if got := IsCancellableFulfillment(tc.from); got != tc.want {
			t.Errorf("IsCancellableFulfillment(%q) = %v, ingin %v", tc.from, got, tc.want)
		}
	}
}

func TestRequireReason(t *testing.T) {
	cases := []struct {
		name    string
		reason  string
		wantErr bool
	}{
		{"terisi", "stok rusak", false},
		{"kosong", "", true},
		{"hanya spasi", "   ", true},
		{"hanya tab dan baris baru", "\t\n", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := RequireReason(tc.reason)
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
}

func assertStrings(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("panjang = %d (%v), ingin %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("elemen %d = %q, ingin %q (seluruhnya %v)", i, got[i], want[i], got)
		}
	}
}
