package fundraising

import (
	"testing"
	"time"
)

func TestCanTransitionCampaign(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"draft ke published", CampaignDraft, CampaignPublished, true},
		{"draft ke closed", CampaignDraft, CampaignClosed, true},
		{"published ke draft", CampaignPublished, CampaignDraft, true},
		{"published ke closed", CampaignPublished, CampaignClosed, true},
		{"closed tidak bisa dibuka lagi", CampaignClosed, CampaignPublished, false},
		{"closed ke draft ditolak", CampaignClosed, CampaignDraft, false},
		{"draft ke draft ditolak", CampaignDraft, CampaignDraft, false},
		{"status tak dikenal ditolak", "archived", CampaignPublished, false},
		{"tujuan tak dikenal ditolak", CampaignDraft, "archived", false},
		{"dua status kosong ditolak", "", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanTransitionCampaign(tc.from, tc.to); got != tc.want {
				t.Errorf("CanTransitionCampaign(%q, %q) = %v, ingin %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestDonationWindowOpen(t *testing.T) {
	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)
	ptr := func(t time.Time) *time.Time { return &t }

	cases := []struct {
		name   string
		now    time.Time
		status string
		starts *time.Time
		ends   *time.Time
		want   bool
	}{
		{"draft selalu tertutup", start.Add(24 * time.Hour), CampaignDraft, ptr(start), ptr(end), false},
		{"closed selalu tertutup", start.Add(24 * time.Hour), CampaignClosed, ptr(start), ptr(end), false},
		{"sebelum starts_at", start.Add(-time.Hour), CampaignPublished, ptr(start), ptr(end), false},
		{"tepat di starts_at sudah terbuka", start, CampaignPublished, ptr(start), ptr(end), true},
		{"di tengah rentang", start.Add(10 * 24 * time.Hour), CampaignPublished, ptr(start), ptr(end), true},
		{"tepat di ends_at tertutup (eksklusif)", end, CampaignPublished, ptr(start), ptr(end), false},
		{"setelah ends_at", end.Add(time.Hour), CampaignPublished, ptr(start), ptr(end), false},
		{"tanpa batas waktu", start.Add(10 * 24 * time.Hour), CampaignPublished, nil, nil, true},
		{"hanya ends_at dan belum lewat", start, CampaignPublished, nil, ptr(end), true},
		{"hanya ends_at dan sudah lewat", end, CampaignPublished, nil, ptr(end), false},
		{"hanya starts_at dan belum mulai", start.Add(-time.Hour), CampaignPublished, ptr(start), nil, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := DonationWindowOpen(tc.now, tc.status, tc.starts, tc.ends)
			if got != tc.want {
				t.Errorf("DonationWindowOpen(%v, %q) = %v, ingin %v", tc.now, tc.status, got, tc.want)
			}
		})
	}
}

func TestProgressPercent(t *testing.T) {
	cases := []struct {
		name   string
		raised int
		target int
		want   int
	}{
		{"tanpa target", 500_000, 0, 0},
		{"target negatif", 500_000, -1, 0},
		{"belum terkumpul", 0, 1_000_000, 0},
		{"seperempat", 250_000, 1_000_000, 25},
		{"tepat penuh", 1_000_000, 1_000_000, 100},
		{"melebihi target dipagari 100", 5_000_000, 1_000_000, 100},
		{"dibulatkan ke bawah", 999, 1_000, 99},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ProgressPercent(tc.raised, tc.target); got != tc.want {
				t.Errorf("ProgressPercent(%d, %d) = %d, ingin %d", tc.raised, tc.target, got, tc.want)
			}
		})
	}
}

func TestNormalizeSlug(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{"huruf kecil", "Bantu Yatim", "bantu-yatim"},
		{"spasi berulang jadi satu hubung", "Bantu   Yatim", "bantu-yatim"},
		{"garis bawah jadi hubung", "bantu_yatim", "bantu-yatim"},
		{"huruf besar", "WAKAF MASJID", "wakaf-masjid"},
		{"simbol dibuang", "wakaf/masjid#2026", "wakafmasjid2026"},
		{"hubung di tepi dibuang", "  -bantu-  ", "bantu"},
		{"angka dipertahankan", "campaign 2026", "campaign-2026"},
		{"kosong", "   ", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeSlug(tc.input); got != tc.want {
				t.Errorf("NormalizeSlug(%q) = %q, ingin %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsValidFundTypeDanCampaignStatus(t *testing.T) {
	for _, valid := range []string{FundOperasional, FundDakwah, FundSosial, FundPendidikan, FundLainnya} {
		if !IsValidFundType(valid) {
			t.Errorf("IsValidFundType(%q) = false, ingin true", valid)
		}
	}
	for _, invalid := range []string{"", "zakat", "Dakwah", "lain-lain"} {
		if IsValidFundType(invalid) {
			t.Errorf("IsValidFundType(%q) = true, ingin false", invalid)
		}
	}

	for _, valid := range []string{CampaignDraft, CampaignPublished, CampaignClosed} {
		if !IsValidCampaignStatus(valid) {
			t.Errorf("IsValidCampaignStatus(%q) = false, ingin true", valid)
		}
	}
	for _, invalid := range []string{"", "archived", "Published"} {
		if IsValidCampaignStatus(invalid) {
			t.Errorf("IsValidCampaignStatus(%q) = true, ingin false", invalid)
		}
	}
}
