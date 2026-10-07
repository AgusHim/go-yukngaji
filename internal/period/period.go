// Package period menghitung periode waktu di zona Asia/Jakarta.
//
// Dipakai bersama oleh gamification (leaderboard mingguan/bulanan) dan
// mission (periode klaim). Dipisah ke paket sendiri supaya tidak ada impor
// melingkar: mission bergantung pada gamification untuk pemberian XP, jadi
// aritmetika tanggal tidak bisa tinggal di salah satunya.
package period

import "time"

// JakartaLocation adalah zona waktu tetap WIB (UTC+7). Jakarta tidak memakai
// daylight saving, sehingga offset tetap sudah tepat sepanjang tahun.
//
// Catatan: presence/date.go punya definisi sendiri dengan nilai yang sama.
// Keduanya sengaja dibiarkan terpisah agar perubahan di sini tidak menyentuh
// jalur check-in yang sudah terverifikasi.
var JakartaLocation = time.FixedZone("Asia/Jakarta", 7*60*60)

// Awal minggu mengikuti ISO-8601: Senin.
const WeekStartsOn = time.Monday

func locOrDefault(loc *time.Location) *time.Location {
	if loc == nil {
		return JakartaLocation
	}
	return loc
}

// DayRange mengembalikan rentang satu hari kalender di loc: [00:00, besok 00:00).
func DayRange(t time.Time, loc *time.Location) (time.Time, time.Time) {
	loc = locOrDefault(loc)
	local := t.In(loc)
	start := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 0, 1)
}

// ISOWeekRange mengembalikan rentang minggu ISO-8601 (Senin s.d. Minggu)
// beserta nomor tahun-minggu ISO-nya.
//
// Memakai t.ISOWeek(), bukan t.Year(): 1 Januari 2027 masih termasuk minggu
// ke-53 tahun 2026, dan t.Year() akan salah melaporkannya sebagai 2027-W53.
func ISOWeekRange(t time.Time, loc *time.Location) (start, end time.Time, isoYear, isoWeek int) {
	loc = locOrDefault(loc)
	local := t.In(loc)
	isoYear, isoWeek = local.ISOWeek()

	// Weekday() memakai Minggu = 0; ubah ke konvensi ISO (Senin = 1).
	weekday := int(local.Weekday())
	if weekday == 0 {
		weekday = 7
	}

	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, loc)
	start = dayStart.AddDate(0, 0, -(weekday - 1))
	return start, start.AddDate(0, 0, 7), isoYear, isoWeek
}

// MonthRange mengembalikan rentang satu bulan kalender di loc.
func MonthRange(t time.Time, loc *time.Location) (time.Time, time.Time) {
	loc = locOrDefault(loc)
	local := t.In(loc)
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, loc)
	return start, start.AddDate(0, 1, 0)
}
