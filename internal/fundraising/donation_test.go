package fundraising

import "testing"

func TestCanTransitionDonation(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"pending ke confirmed", DonationPending, DonationConfirmed, true},
		{"pending ke rejected", DonationPending, DonationRejected, true},
		{"pending ke cancelled", DonationPending, DonationCancelled, true},
		{"pending tidak bisa langsung refunded", DonationPending, DonationRefunded, false},
		{"confirmed ke refunded", DonationConfirmed, DonationRefunded, true},
		{"confirmed tidak bisa kembali pending", DonationConfirmed, DonationPending, false},
		{"confirmed tidak bisa rejected", DonationConfirmed, DonationRejected, false},
		{"refunded terminal", DonationRefunded, DonationConfirmed, false},
		{"rejected terminal", DonationRejected, DonationConfirmed, false},
		{"cancelled terminal", DonationCancelled, DonationConfirmed, false},
		{"status sama ditolak", DonationConfirmed, DonationConfirmed, false},
		{"status tak dikenal ditolak", "expired", DonationConfirmed, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanTransitionDonation(tc.from, tc.to); got != tc.want {
				t.Errorf("CanTransitionDonation(%q, %q) = %v, ingin %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestValidateDonationAmount(t *testing.T) {
	cases := []struct {
		name    string
		amount  int
		wantErr bool
	}{
		{"nol ditolak", 0, true},
		{"negatif ditolak", -50_000, true},
		{"di bawah minimum ditolak", MinDonationAmount - 1, true},
		{"tepat minimum diterima", MinDonationAmount, false},
		{"nominal wajar diterima", 100_000, false},
		{"tepat batas atas diterima", MaxDonationAmount, false},
		{"melebihi batas atas ditolak", MaxDonationAmount + 1, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateDonationAmount(tc.amount)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateDonationAmount(%d) error = %v, inginErr %v", tc.amount, err, tc.wantErr)
			}
		})
	}
}

func TestNormalizeMessage(t *testing.T) {
	cases := []struct {
		name       string
		input      *string
		wantMsg    *string
		wantStatus string
	}{
		{"nil menjadi none", nil, nil, MessageNone},
		{"kosong menjadi none", strPtr("   "), nil, MessageNone},
		{"pesan dirapikan dan pending", strPtr("  semoga berkah  "), strPtr("semoga berkah"), MessagePending},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg, status := NormalizeMessage(tc.input)
			if status != tc.wantStatus {
				t.Errorf("status = %q, ingin %q", status, tc.wantStatus)
			}
			if (msg == nil) != (tc.wantMsg == nil) {
				t.Fatalf("pesan = %v, ingin %v", deref(msg), deref(tc.wantMsg))
			}
			if msg != nil && *msg != *tc.wantMsg {
				t.Errorf("pesan = %q, ingin %q", *msg, *tc.wantMsg)
			}
		})
	}
}

func TestIsValidMessageStatus(t *testing.T) {
	for _, valid := range []string{MessageNone, MessagePending, MessageApproved, MessageHidden} {
		if !IsValidMessageStatus(valid) {
			t.Errorf("IsValidMessageStatus(%q) = false, ingin true", valid)
		}
	}
	for _, invalid := range []string{"", "rejected", "Approved"} {
		if IsValidMessageStatus(invalid) {
			t.Errorf("IsValidMessageStatus(%q) = true, ingin false", invalid)
		}
	}
}

func strPtr(s string) *string { return &s }

func deref(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return *s
}
