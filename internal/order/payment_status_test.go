package order

import "testing"

func TestIsValidStatus(t *testing.T) {
	valid := []string{StatusPending, StatusPaid, StatusExpired, StatusCancelled, StatusFailed, StatusRefunded}
	for _, status := range valid {
		if !IsValidStatus(status) {
			t.Errorf("IsValidStatus(%q) harus true", status)
		}
	}
	for _, status := range []string{"", "PENDING", "done", "menunggu"} {
		if IsValidStatus(status) {
			t.Errorf("IsValidStatus(%q) harus false", status)
		}
	}
}

func TestCanTransition(t *testing.T) {
	tests := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"pending ke paid", StatusPending, StatusPaid, true},
		{"pending ke expired", StatusPending, StatusExpired, true},
		{"pending ke cancelled", StatusPending, StatusCancelled, true},
		{"pending ke failed", StatusPending, StatusFailed, true},
		{"paid ke refunded", StatusPaid, StatusRefunded, true},
		{"paid tidak bisa kembali pending", StatusPaid, StatusPending, false},
		{"expired bersifat final", StatusExpired, StatusPaid, false},
		{"cancelled bersifat final", StatusCancelled, StatusPaid, false},
		{"failed bersifat final", StatusFailed, StatusPaid, false},
		{"refunded bersifat final", StatusRefunded, StatusPaid, false},
		{"status sama bukan transisi", StatusPending, StatusPending, false},
		{"status asal tak dikenal", "unknown", StatusPaid, false},
		{"status tujuan tak dikenal", StatusPending, "unknown", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanTransition(tt.from, tt.to); got != tt.want {
				t.Errorf("CanTransition(%q, %q) = %v, want %v", tt.from, tt.to, got, tt.want)
			}
		})
	}
}
