package fundraising

import (
	"strings"
	"time"
)

// campaignTransitions adalah satu-satunya definisi perpindahan status campaign.
//
// draft -> published: menerbitkan.
// draft -> closed:    menutup tanpa pernah menerbitkan.
// published -> draft: menarik kembali ke draf (mis. salah terbit).
// published -> closed: menutup.
// closed  -> (tidak ada): terminal, supaya campaign yang sudah ditutup tidak
//
//	diam-diam menerima donasi lagi hanya karena statusnya diubah.
var campaignTransitions = map[string][]string{
	CampaignDraft:     {CampaignPublished, CampaignClosed},
	CampaignPublished: {CampaignDraft, CampaignClosed},
	CampaignClosed:    {},
}

// CanTransitionCampaign melaporkan apakah perpindahan status campaign sah.
// Status yang tidak dikenal selalu ditolak.
func CanTransitionCampaign(from, to string) bool {
	if from == to {
		return false
	}
	for _, allowed := range campaignTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// DonationWindowOpen melaporkan apakah campaign sedang menerima donasi.
//
// Donasi hanya diterima pada status 'published' dan di dalam rentang waktunya.
// Batas akhir bersifat eksklusif: tepat pada ends_at, jendela sudah tertutup.
//
// Jendela diperiksa di Go pada saat donasi dibuat, bukan dengan menulis status
// 'closed' saat tenggat lewat. Tanpa scheduler, campaign yang tenggatnya lewat
// harus tetap menolak donasi tanpa menunggu pengurus menutupnya.
func DonationWindowOpen(now time.Time, status string, startsAt, endsAt *time.Time) bool {
	if status != CampaignPublished {
		return false
	}
	if startsAt != nil && now.Before(*startsAt) {
		return false
	}
	if endsAt != nil && !now.Before(*endsAt) {
		return false
	}
	return true
}

// IsValidFundType melaporkan apakah jenis dana dikenal.
func IsValidFundType(fundType string) bool {
	switch fundType {
	case FundOperasional, FundDakwah, FundSosial, FundPendidikan, FundLainnya:
		return true
	}
	return false
}

// IsValidCampaignStatus melaporkan apakah status campaign dikenal.
func IsValidCampaignStatus(status string) bool {
	switch status {
	case CampaignDraft, CampaignPublished, CampaignClosed:
		return true
	}
	return false
}

// ProgressPercent menghitung persentase progres terhadap target.
//
// Target 0 berarti tanpa target: hasilnya 0, bukan pembagian dengan nol.
// Hasilnya dipagari 100 supaya bilah progres tidak pernah melebihi lebarnya,
// dan dipagari 0 untuk nilai negatif.
func ProgressPercent(raised, target int) int {
	if target <= 0 || raised <= 0 {
		return 0
	}
	if raised >= target {
		return 100
	}
	return raised * 100 / target
}

// NormalizeSlug merapikan slug campaign menjadi huruf kecil dengan tanda hubung.
func NormalizeSlug(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	lastDash := false
	for _, r := range trimmed {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ':
			if !lastDash && b.Len() > 0 {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
