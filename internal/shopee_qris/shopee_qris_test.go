package shopee_qris

import (
	"testing"
	"time"

	"mainyuk/internal/shopee"
)

func TestMatchesMutationRequiresExactUniqueAmountAndWindow(t *testing.T) {
	created := time.Date(2026, 10, 22, 0, 0, 0, 0, time.UTC)
	charge := &Charge{PayableAmount: 100123, CreatedAt: created, ExpiresAt: created.Add(30 * time.Minute)}

	tests := []struct {
		name     string
		mutation shopee.SuccessfulTransaction
		want     bool
	}{
		{"exact", shopee.SuccessfulTransaction{ID: "tx-1", Amount: 100123, CreateTime: created.Add(time.Minute)}, true},
		{"amount berbeda", shopee.SuccessfulTransaction{ID: "tx-2", Amount: 100124, CreateTime: created.Add(time.Minute)}, false},
		{"terlalu awal", shopee.SuccessfulTransaction{ID: "tx-3", Amount: 100123, CreateTime: created.Add(-2*time.Minute - time.Second)}, false},
		{"terlalu akhir", shopee.SuccessfulTransaction{ID: "tx-4", Amount: 100123, CreateTime: charge.ExpiresAt.Add(2*time.Minute + time.Second)}, false},
		{"id kosong", shopee.SuccessfulTransaction{Amount: 100123, CreateTime: created.Add(time.Minute)}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchesMutation(charge, tc.mutation); got != tc.want {
				t.Fatalf("matchesMutation() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestRandomCodeAlwaysWithinDatabaseConstraint(t *testing.T) {
	for i := 0; i < 1000; i++ {
		code, err := randomCode()
		if err != nil {
			t.Fatal(err)
		}
		if code < 1 || code > 999 {
			t.Fatalf("kode %d di luar rentang", code)
		}
	}
}
