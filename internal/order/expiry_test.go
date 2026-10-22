package order

import (
	"testing"
	"time"
)

func TestShouldExpirePendingOrder(t *testing.T) {
	now := time.Date(2026, time.October, 22, 10, 0, 0, 0, time.UTC)
	tests := []struct {
		name      string
		status    string
		expiresAt time.Time
		want      bool
	}{
		{"pending yang lewat batas", StatusPending, now.Add(-time.Second), true},
		{"pending tepat di batas", StatusPending, now, true},
		{"pending yang masih berlaku", StatusPending, now.Add(time.Second), false},
		{"paid yang lewat batas", StatusPaid, now.Add(-time.Second), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldExpire(tt.status, tt.expiresAt, now); got != tt.want {
				t.Fatalf("ShouldExpire(%q, %v, %v) = %v, want %v", tt.status, tt.expiresAt, now, got, tt.want)
			}
		})
	}
}
