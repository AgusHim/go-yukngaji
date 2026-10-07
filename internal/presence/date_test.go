package presence

import (
	"testing"
	"time"
)

func TestCheckInDate(t *testing.T) {
	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{
			name: "tengah hari UTC masih tanggal yang sama di Jakarta",
			in:   time.Date(2026, time.October, 2, 5, 0, 0, 0, time.UTC),
			want: "2026-10-02",
		},
		{
			name: "malam UTC sudah ganti hari di Jakarta",
			in:   time.Date(2026, time.October, 1, 18, 30, 0, 0, time.UTC),
			want: "2026-10-02",
		},
		{
			name: "dini hari Jakarta belum ganti hari",
			in:   time.Date(2026, time.October, 2, 16, 59, 0, 0, time.UTC),
			want: "2026-10-02",
		},
		{
			name: "tepat tengah malam Jakarta",
			in:   time.Date(2026, time.October, 1, 17, 0, 0, 0, time.UTC),
			want: "2026-10-02",
		},
		{
			name: "waktu sudah berzona Jakarta",
			in:   time.Date(2026, time.October, 2, 23, 59, 0, 0, JakartaLocation),
			want: "2026-10-02",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckInDate(tt.in, JakartaLocation)
			if got.Format("2006-01-02") != tt.want {
				t.Errorf("CheckInDate(%s) = %s, want %s",
					tt.in.Format(time.RFC3339), got.Format("2006-01-02"), tt.want)
			}
			if got.Location() != time.UTC {
				t.Errorf("hasil harus UTC agar aman ditulis ke kolom date, dapat %s", got.Location())
			}
			if h, m, s := got.Clock(); h != 0 || m != 0 || s != 0 {
				t.Errorf("hasil harus tengah malam, dapat %02d:%02d:%02d", h, m, s)
			}
		})
	}
}

func TestCheckInDateSameDayIsStable(t *testing.T) {
	// Dua pemindaian berbeda jam pada hari Jakarta yang sama harus
	// menghasilkan nilai identik, karena nilai itulah kunci idempotensi.
	morning := time.Date(2026, time.October, 2, 1, 15, 0, 0, JakartaLocation)
	night := time.Date(2026, time.October, 2, 22, 45, 0, 0, JakartaLocation)

	if !CheckInDate(morning, JakartaLocation).Equal(CheckInDate(night, JakartaLocation)) {
		t.Error("check-in pagi dan malam di hari yang sama harus punya tanggal yang sama")
	}
}

func TestCheckInDateNilLocationFallsBack(t *testing.T) {
	in := time.Date(2026, time.October, 1, 18, 0, 0, 0, time.UTC)
	if got := CheckInDate(in, nil); got.Format("2006-01-02") != "2026-10-02" {
		t.Errorf("lokasi nil harus jatuh ke Asia/Jakarta, dapat %s", got.Format("2006-01-02"))
	}
}
