package fundraising

import "testing"

func TestCanTransitionMessage(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"pending ke approved", MessagePending, MessageApproved, true},
		{"pending ke hidden", MessagePending, MessageHidden, true},
		{"approved ke hidden", MessageApproved, MessageHidden, true},
		{"hidden ke approved", MessageHidden, MessageApproved, true},
		{"approved ke approved ditolak", MessageApproved, MessageApproved, false},
		{"pending ke none ditolak", MessagePending, MessageNone, false},
		{"none tidak bisa dimoderasi", MessageNone, MessageApproved, false},
		{"hidden ke pending ditolak", MessageHidden, MessagePending, false},
		{"status tak dikenal ditolak", "rejected", MessageApproved, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanTransitionMessage(tc.from, tc.to); got != tc.want {
				t.Errorf("CanTransitionMessage(%q, %q) = %v, ingin %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}
