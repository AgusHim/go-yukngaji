package metrics

import (
	"testing"
	"time"

	"mainyuk/internal/period"
)

// TestWindowsFor menguji perhitungan rentang laporan.
//
// Ini satu-satunya bagian modul ini yang dapat diuji tanpa database: seluruh
// query-nya hanya berjalan di atas PostgreSQL. Yang diuji di sini adalah
// batas-batas yang paling mudah salah — pergantian bulan, pergantian tahun,
// dan pengaruh zona waktu terhadap keduanya.
func TestWindowsFor(t *testing.T) {
	jakarta := period.JakartaLocation
	utc := time.UTC

	tests := []struct {
		name string
		now  time.Time
		loc  *time.Location
		want Windows
	}{
		{
			// 17:00 WIB, pertengahan bulan.
			name: "pertengahan bulan di Jakarta",
			now:  time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC),
			loc:  jakarta,
			want: Windows{
				Now:       time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC),
				MonthFrom: time.Date(2026, 10, 1, 0, 0, 0, 0, jakarta),
				MonthTo:   time.Date(2026, 11, 1, 0, 0, 0, 0, jakarta),
				Day7From:  time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC),
				Day30From: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC),
			},
		},
		{
			// 2026-12-31 20:00 UTC sudah 2027-01-01 03:00 WIB. Bulan berjalan
			// harus Januari, bukan Desember — inilah kesalahan yang paling
			// mudah terjadi bila zona waktunya diabaikan.
			name: "pergantian tahun menurut Jakarta",
			now:  time.Date(2026, 12, 31, 20, 0, 0, 0, time.UTC),
			loc:  jakarta,
			want: Windows{
				Now:       time.Date(2026, 12, 31, 20, 0, 0, 0, time.UTC),
				MonthFrom: time.Date(2027, 1, 1, 0, 0, 0, 0, jakarta),
				MonthTo:   time.Date(2027, 2, 1, 0, 0, 0, 0, jakarta),
				Day7From:  time.Date(2026, 12, 24, 20, 0, 0, 0, time.UTC),
				Day30From: time.Date(2026, 12, 1, 20, 0, 0, 0, time.UTC),
			},
		},
		{
			// 2026-10-31 20:00 UTC = 2026-11-01 03:00 WIB: bulan berganti di
			// tengah malam Jakarta, bukan tengah malam UTC.
			name: "pergantian bulan di tengah malam Jakarta",
			now:  time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC),
			loc:  jakarta,
			want: Windows{
				Now:       time.Date(2026, 10, 31, 20, 0, 0, 0, time.UTC),
				MonthFrom: time.Date(2026, 11, 1, 0, 0, 0, 0, jakarta),
				MonthTo:   time.Date(2026, 12, 1, 0, 0, 0, 0, jakarta),
				Day7From:  time.Date(2026, 10, 24, 20, 0, 0, 0, time.UTC),
				Day30From: time.Date(2026, 10, 1, 20, 0, 0, 0, time.UTC),
			},
		},
		{
			// Februari 2028 punya 29 hari; akhir bulannya 1 Maret.
			name: "tahun kabisat",
			now:  time.Date(2028, 2, 10, 0, 0, 0, 0, time.UTC),
			loc:  jakarta,
			want: Windows{
				Now:       time.Date(2028, 2, 10, 0, 0, 0, 0, time.UTC),
				MonthFrom: time.Date(2028, 2, 1, 0, 0, 0, 0, jakarta),
				MonthTo:   time.Date(2028, 3, 1, 0, 0, 0, 0, jakarta),
				Day7From:  time.Date(2028, 2, 3, 0, 0, 0, 0, time.UTC),
				Day30From: time.Date(2028, 1, 11, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			// loc nil berarti Jakarta, bukan UTC.
			name: "lokasi kosong berarti Jakarta",
			now:  time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC),
			loc:  nil,
			want: Windows{
				Now:       time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC),
				MonthFrom: time.Date(2026, 10, 1, 0, 0, 0, 0, jakarta),
				MonthTo:   time.Date(2026, 11, 1, 0, 0, 0, 0, jakarta),
				Day7From:  time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC),
				Day30From: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC),
			},
		},
		{
			// Zona yang diberikan benar-benar dipakai, bukan diabaikan.
			name: "zona waktu eksplisit ikut dipakai",
			now:  time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC),
			loc:  utc,
			want: Windows{
				Now:       time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC),
				MonthFrom: time.Date(2026, 10, 1, 0, 0, 0, 0, utc),
				MonthTo:   time.Date(2026, 11, 1, 0, 0, 0, 0, utc),
				Day7From:  time.Date(2026, 9, 29, 10, 0, 0, 0, time.UTC),
				Day30From: time.Date(2026, 9, 6, 10, 0, 0, 0, time.UTC),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WindowsFor(tt.now, tt.loc)

			if !got.Now.Equal(tt.want.Now) {
				t.Errorf("Now = %v, ingin %v", got.Now, tt.want.Now)
			}
			if !got.MonthFrom.Equal(tt.want.MonthFrom) {
				t.Errorf("MonthFrom = %v, ingin %v", got.MonthFrom, tt.want.MonthFrom)
			}
			if !got.MonthTo.Equal(tt.want.MonthTo) {
				t.Errorf("MonthTo = %v, ingin %v", got.MonthTo, tt.want.MonthTo)
			}
			if !got.Day7From.Equal(tt.want.Day7From) {
				t.Errorf("Day7From = %v, ingin %v", got.Day7From, tt.want.Day7From)
			}
			if !got.Day30From.Equal(tt.want.Day30From) {
				t.Errorf("Day30From = %v, ingin %v", got.Day30From, tt.want.Day30From)
			}
		})
	}
}

// TestWindowsMonthIsHalfOpen memastikan jendela bulan kalender bersifat
// [awal, akhir): akhir bulan sebelumnya tidak ikut, dan awal bulan berikutnya
// tidak ikut. Salah satu saja akan membuat MAU menghitung dua kali atau nol.
func TestWindowsMonthIsHalfOpen(t *testing.T) {
	now := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	w := WindowsFor(now, period.JakartaLocation)

	akhirBulanLalu := time.Date(2026, 10, 1, 0, 0, 0, 0, period.JakartaLocation).Add(-time.Nanosecond)
	if !akhirBulanLalu.Before(w.MonthFrom) {
		t.Errorf("akhir bulan lalu (%v) seharusnya sebelum MonthFrom (%v)", akhirBulanLalu, w.MonthFrom)
	}

	awalBulanDepan := w.MonthTo
	if !awalBulanDepan.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, period.JakartaLocation)) {
		t.Errorf("MonthTo = %v, ingin awal November", awalBulanDepan)
	}
}

// TestWindowsRollingAreExactlySevenAndThirtyDays memastikan D7 dan D30 adalah
// jendela bergulir, bukan minggu/bulan kalender.
func TestWindowsRollingAreExactlySevenAndThirtyDays(t *testing.T) {
	now := time.Date(2026, 10, 6, 10, 0, 0, 0, time.UTC)
	w := WindowsFor(now, period.JakartaLocation)

	if selisih := w.Now.Sub(w.Day7From); selisih != 7*24*time.Hour {
		t.Errorf("lebar jendela 7 hari = %v, ingin 168 jam", selisih)
	}
	if selisih := w.Now.Sub(w.Day30From); selisih != 30*24*time.Hour {
		t.Errorf("lebar jendela 30 hari = %v, ingin 720 jam", selisih)
	}
}
