package gamification

import (
	"time"

	"mainyuk/internal/period"
)

// LeaderboardRow adalah hasil agregasi mentah dari xp_ledger sebelum level
// diresolusi dan peringkat ditetapkan.
type LeaderboardRow struct {
	PublicID      string    `gorm:"column:public_id"`
	Alias         *string   `gorm:"column:alias"`
	AvatarURL     *string   `gorm:"column:avatar_url"`
	ShowBadges    bool      `gorm:"column:show_badges"`
	NetXP         int       `gorm:"column:net_xp"`
	FirstEarnedAt time.Time `gorm:"column:first_earned_at"`
}

// LeaderboardEntry adalah DTO publik. Sengaja tidak memuat id akun internal,
// email, telepon, maupun alamat — hanya identitas publik.
type LeaderboardEntry struct {
	Rank      int     `json:"rank"`
	PublicID  string  `json:"public_id"`
	Alias     string  `json:"alias"`
	AvatarURL *string `json:"avatar_url"`
	Level     int     `json:"level"`
	LevelName string  `json:"level_name"`
	Badge     string  `json:"badge"`
	XP        int     `json:"xp"`
}

// LeaderboardPage adalah satu halaman hasil leaderboard.
type LeaderboardPage struct {
	Period  string              `json:"period"`
	Page    int                 `json:"page"`
	PerPage int                 `json:"per_page"`
	HasMore bool                `json:"has_more"`
	Entries []*LeaderboardEntry `json:"entries"`
}

// Pagination bounds. per_page dibatasi agar satu permintaan tidak bisa
// memaksa agregasi seluruh ledger.
const (
	DefaultPerPage = 20
	MaxPerPage     = 100
)

// NormalizePagination merapikan page/per_page yang dikirim client.
func NormalizePagination(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}
	return page, perPage
}

// IsValidPeriod melaporkan apakah periodKey termasuk periode yang didukung.
func IsValidPeriod(periodKey string) bool {
	switch periodKey {
	case PeriodWeekly, PeriodMonthly, PeriodAllTime:
		return true
	default:
		return false
	}
}

// PeriodRange mengembalikan batas rentang [from, to) untuk periode tertentu.
// Untuk all_time keduanya nil, artinya tanpa batas waktu.
func PeriodRange(periodKey string, now time.Time, loc *time.Location) (from, to *time.Time) {
	switch periodKey {
	case PeriodWeekly:
		start, end, _, _ := period.ISOWeekRange(now, loc)
		return &start, &end
	case PeriodMonthly:
		start, end := period.MonthRange(now, loc)
		return &start, &end
	default:
		return nil, nil
	}
}
