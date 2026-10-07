package gamification

import (
	"testing"
	"time"
)

func TestCappedRewardDecision(t *testing.T) {
	cases := []struct {
		name        string
		ruleXP      int
		monthlyCap  int
		rewardedNow int
		want        int
	}{
		{"cap nol berarti tanpa batas", 50, 0, 999, 50},
		{"cap negatif berarti tanpa batas", 50, -1, 999, 50},
		{"belum ada sumber ber-XP", 50, 3, 0, 50},
		{"masih di bawah batas", 50, 3, 2, 50},
		{"tepat di batas", 50, 3, 3, 0},
		{"melewati batas", 50, 3, 4, 0},
		{"aturan nol", 0, 3, 0, 0},
		{"aturan negatif", -5, 3, 0, 0},
		{"batas satu dan sudah terpakai", 50, 1, 1, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := CappedRewardDecision(tc.ruleXP, tc.monthlyCap, tc.rewardedNow)
			if got != tc.want {
				t.Errorf("CappedRewardDecision(%d, %d, %d) = %d, ingin %d",
					tc.ruleXP, tc.monthlyCap, tc.rewardedNow, got, tc.want)
			}
		})
	}
}

// Batas bulan harus mengikuti kalender Asia/Jakarta, bukan zona server.
func TestMonthlyCapWindowJakarta(t *testing.T) {
	// 31 Januari 2026 pukul 18:00 UTC = 1 Februari 2026 pukul 01:00 WIB.
	// Pergantian bulan karena itu jatuh di Februari, bukan Januari.
	now := time.Date(2026, time.January, 31, 18, 0, 0, 0, time.UTC)

	from, to := MonthlyCapWindow(now)

	if got := from.In(time.FixedZone("WIB", 7*60*60)).Format("2006-01-02"); got != "2026-02-01" {
		t.Errorf("awal bulan = %s, ingin 2026-02-01", got)
	}
	if got := to.In(time.FixedZone("WIB", 7*60*60)).Format("2006-01-02"); got != "2026-03-01" {
		t.Errorf("akhir bulan = %s, ingin 2026-03-01", got)
	}
	if !to.After(from) {
		t.Errorf("akhir bulan (%v) harus setelah awal bulan (%v)", to, from)
	}
}
