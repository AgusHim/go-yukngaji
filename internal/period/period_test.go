package period

import (
	"testing"
	"time"
)

// jkt membangun waktu pada zona Asia/Jakarta.
func jkt(year int, month time.Month, day, hour int) time.Time {
	return time.Date(year, month, day, hour, 0, 0, 0, JakartaLocation)
}

func TestDayRange(t *testing.T) {
	tests := []struct {
		name      string
		in        time.Time
		wantStart string
		wantEnd   string
	}{
		{
			name: "tengah hari tetap tanggal yang sama",
			in:   jkt(2026, time.October, 2, 12), wantStart: "2026-10-02", wantEnd: "2026-10-03",
		},
		{
			name: "menjelang tengah malam belum pindah tanggal",
			in:   jkt(2026, time.October, 2, 23), wantStart: "2026-10-02", wantEnd: "2026-10-03",
		},
		{
			// 2026-10-01 17:00 UTC = 2026-10-02 00:00 WIB.
			name:      "instant UTC dipetakan ke tanggal Jakarta",
			in:        time.Date(2026, time.October, 1, 17, 0, 0, 0, time.UTC),
			wantStart: "2026-10-02", wantEnd: "2026-10-03",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := DayRange(tt.in, JakartaLocation)
			if got := start.Format("2006-01-02"); got != tt.wantStart {
				t.Errorf("start = %s, want %s", got, tt.wantStart)
			}
			if got := end.Format("2006-01-02"); got != tt.wantEnd {
				t.Errorf("end = %s, want %s", got, tt.wantEnd)
			}
		})
	}
}

func TestISOWeekRange(t *testing.T) {
	tests := []struct {
		name        string
		in          time.Time
		wantStart   string
		wantEnd     string
		wantISOYear int
		wantISOWeek int
	}{
		{
			name:      "tengah minggu",
			in:        jkt(2026, time.October, 2, 10), // Jumat
			wantStart: "2026-09-28", wantEnd: "2026-10-05", wantISOYear: 2026, wantISOWeek: 40,
		},
		{
			name:      "hari Senin adalah awal minggunya sendiri",
			in:        jkt(2026, time.September, 28, 0),
			wantStart: "2026-09-28", wantEnd: "2026-10-05", wantISOYear: 2026, wantISOWeek: 40,
		},
		{
			name:      "hari Minggu masih minggu yang sama",
			in:        jkt(2026, time.October, 4, 23),
			wantStart: "2026-09-28", wantEnd: "2026-10-05", wantISOYear: 2026, wantISOWeek: 40,
		},
		{
			// 31 Des 2026 (Kamis) masih minggu ke-53 tahun 2026.
			name:      "31 Desember 2026 masuk minggu ke-53 tahun 2026",
			in:        jkt(2026, time.December, 31, 12),
			wantStart: "2026-12-28", wantEnd: "2027-01-04", wantISOYear: 2026, wantISOWeek: 53,
		},
		{
			// 1 Jan 2027 (Jumat) masih minggu yang sama dengan 31 Des 2026.
			name:      "1 Januari 2027 masih minggu ke-53 tahun 2026",
			in:        jkt(2027, time.January, 1, 12),
			wantStart: "2026-12-28", wantEnd: "2027-01-04", wantISOYear: 2026, wantISOWeek: 53,
		},
		{
			name:      "4 Januari 2027 mulai minggu pertama 2027",
			in:        jkt(2027, time.January, 4, 0),
			wantStart: "2027-01-04", wantEnd: "2027-01-11", wantISOYear: 2027, wantISOWeek: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end, isoYear, isoWeek := ISOWeekRange(tt.in, JakartaLocation)
			if got := start.Format("2006-01-02"); got != tt.wantStart {
				t.Errorf("start = %s, want %s", got, tt.wantStart)
			}
			if got := end.Format("2006-01-02"); got != tt.wantEnd {
				t.Errorf("end = %s, want %s", got, tt.wantEnd)
			}
			if isoYear != tt.wantISOYear {
				t.Errorf("isoYear = %d, want %d", isoYear, tt.wantISOYear)
			}
			if isoWeek != tt.wantISOWeek {
				t.Errorf("isoWeek = %d, want %d", isoWeek, tt.wantISOWeek)
			}
		})
	}
}

// Dua tanggal di tepi pergantian tahun harus menghasilkan minggu yang identik;
// inilah yang rusak bila memakai t.Year() alih-alih t.ISOWeek().
func TestISOWeekRangeTepiTahunKonsisten(t *testing.T) {
	startA, endA, yearA, weekA := ISOWeekRange(jkt(2026, time.December, 31, 12), JakartaLocation)
	startB, endB, yearB, weekB := ISOWeekRange(jkt(2027, time.January, 1, 12), JakartaLocation)

	if !startA.Equal(startB) || !endA.Equal(endB) || yearA != yearB || weekA != weekB {
		t.Errorf("31 Des 2026 dan 1 Jan 2027 harus satu minggu yang sama, dapat (%s-%s, %d-W%d) vs (%s-%s, %d-W%d)",
			startA.Format("2006-01-02"), endA.Format("2006-01-02"), yearA, weekA,
			startB.Format("2006-01-02"), endB.Format("2006-01-02"), yearB, weekB)
	}
}

func TestISOWeekRangeSelaluMulaiHariSenin(t *testing.T) {
	// Uji sepanjang satu tahun penuh, termasuk pergantian tahun.
	day := jkt(2026, time.December, 1, 12)
	for i := 0; i < 90; i++ {
		start, end, _, _ := ISOWeekRange(day, JakartaLocation)
		if start.Weekday() != time.Monday {
			t.Fatalf("awal minggu untuk %s jatuh pada %s, bukan Senin",
				day.Format("2006-01-02"), start.Weekday())
		}
		if !end.Equal(start.AddDate(0, 0, 7)) {
			t.Fatalf("rentang minggu untuk %s tidak tepat 7 hari", day.Format("2006-01-02"))
		}
		day = day.AddDate(0, 0, 1)
	}
}

func TestMonthRange(t *testing.T) {
	tests := []struct {
		name      string
		in        time.Time
		wantStart string
		wantEnd   string
	}{
		{name: "pertengahan bulan", in: jkt(2026, time.October, 15, 10), wantStart: "2026-10-01", wantEnd: "2026-11-01"},
		{name: "hari terakhir bulan", in: jkt(2026, time.October, 31, 23), wantStart: "2026-10-01", wantEnd: "2026-11-01"},
		{name: "desember menyeberang tahun", in: jkt(2026, time.December, 31, 23), wantStart: "2026-12-01", wantEnd: "2027-01-01"},
		{name: "februari tahun kabisat", in: jkt(2028, time.February, 29, 12), wantStart: "2028-02-01", wantEnd: "2028-03-01"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := MonthRange(tt.in, JakartaLocation)
			if got := start.Format("2006-01-02"); got != tt.wantStart {
				t.Errorf("start = %s, want %s", got, tt.wantStart)
			}
			if got := end.Format("2006-01-02"); got != tt.wantEnd {
				t.Errorf("end = %s, want %s", got, tt.wantEnd)
			}
		})
	}
}

func TestRentangMemakaiZonaJakarta(t *testing.T) {
	// 2026-10-01 18:00 UTC = 2026-10-02 01:00 WIB, jadi sudah masuk hari
	// berikutnya di Jakarta walau masih 1 Oktober di UTC.
	in := time.Date(2026, time.October, 1, 18, 0, 0, 0, time.UTC)
	start, _ := DayRange(in, nil)
	if got := start.Format("2006-01-02"); got != "2026-10-02" {
		t.Errorf("start = %s, want 2026-10-02 (loc nil harus memakai Jakarta)", got)
	}
}
