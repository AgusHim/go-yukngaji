package mission

import (
	"testing"
	"time"

	"mainyuk/internal/period"
)

func TestPeriodKeyHarianMemakaiZonaJakarta(t *testing.T) {
	loc := period.JakartaLocation

	// 2026-10-02 06:00 WIB masih tanggal 2; pada UTC ini masih tanggal 1,
	// jadi kunci harus mengikuti tanggal Jakarta.
	wib := time.Date(2026, 10, 2, 6, 0, 0, 0, loc)
	if got, want := PeriodKey(TypeDaily, wib), "daily:2026-10-02"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	// 2026-10-02 00:30 WIB = 2026-10-01 17:30 UTC.
	utcSame := time.Date(2026, 10, 1, 17, 30, 0, 0, time.UTC)
	if got, want := PeriodKey(TypeDaily, utcSame), "daily:2026-10-02"; got != want {
		t.Fatalf("instant UTC harus dinilai di zona Jakarta: got %q want %q", got, want)
	}
}

func TestPeriodKeyHarianBergantiDiTengahMalamJakarta(t *testing.T) {
	loc := period.JakartaLocation

	before := time.Date(2026, 10, 2, 23, 59, 59, 0, loc)
	after := time.Date(2026, 10, 3, 0, 0, 0, 0, loc)

	if a, b := PeriodKey(TypeDaily, before), PeriodKey(TypeDaily, after); a == b {
		t.Fatalf("kunci harian harus berganti saat tengah malam Jakarta: %q == %q", a, b)
	}
}

func TestPeriodKeyMingguanMemakaiTahunISO(t *testing.T) {
	loc := period.JakartaLocation

	// Dua tanggal di minggu ISO yang sama, tetapi berbeda tahun kalender.
	// Memakai t.Year() akan menghasilkan 2026-W53 dan 2027-W53 — keliru.
	endOf2026 := time.Date(2026, 12, 31, 12, 0, 0, 0, loc)
	startOf2027 := time.Date(2027, 1, 1, 12, 0, 0, 0, loc)

	if a, b := PeriodKey(TypeWeekly, endOf2026), PeriodKey(TypeWeekly, startOf2027); a != b {
		t.Fatalf("31 Des 2026 dan 1 Jan 2027 harus satu minggu ISO: %q != %q", a, b)
	}
	if got, want := PeriodKey(TypeWeekly, startOf2027), "weekly:2026-W53"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}

	// Senin berikutnya membuka minggu baru.
	nextMonday := time.Date(2027, 1, 4, 12, 0, 0, 0, loc)
	if got, want := PeriodKey(TypeWeekly, nextMonday), "weekly:2027-W01"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestPeriodKeyMingguanStabilSelamaSatuMinggu(t *testing.T) {
	loc := period.JakartaLocation
	want := PeriodKey(TypeWeekly, time.Date(2026, 10, 1, 0, 0, 0, 0, loc))

	// Senin 2026-09-28 s/d Minggu 2026-10-04 harus menghasilkan kunci sama.
	for day := 28; day <= 30; day++ {
		t.Helper()
		at := time.Date(2026, 9, day, 15, 0, 0, 0, loc)
		if got := PeriodKey(TypeWeekly, at); got != want {
			t.Fatalf("September %d: got %q want %q", day, got, want)
		}
	}
	for day := 1; day <= 4; day++ {
		at := time.Date(2026, 10, day, 15, 0, 0, 0, loc)
		if got := PeriodKey(TypeWeekly, at); got != want {
			t.Fatalf("Oktober %d: got %q want %q", day, got, want)
		}
	}

	// Senin 2026-10-05 sudah minggu berikutnya.
	if got := PeriodKey(TypeWeekly, time.Date(2026, 10, 5, 15, 0, 0, 0, loc)); got == want {
		t.Fatal("Senin berikutnya harus membuka periode mingguan baru")
	}
}

func TestPeriodKeySpecialTidakDipecah(t *testing.T) {
	loc := period.JakartaLocation
	a := PeriodKey(TypeSpecial, time.Date(2026, 10, 1, 8, 0, 0, 0, loc))
	b := PeriodKey(TypeSpecial, time.Date(2026, 12, 31, 20, 0, 0, 0, loc))
	if a != b {
		t.Fatalf("misi special harus satu periode untuk seluruh durasinya: %q != %q", a, b)
	}
	if a != "special" {
		t.Fatalf("kunci misi special tidak sesuai: %q", a)
	}
}

func TestPeriodKeyJenisTidakDikenalJatuhKeSpecial(t *testing.T) {
	// Jenis tidak dikenal tidak boleh menghasilkan kunci unik per hari yang
	// diam-diam melonggarkan batas klaim.
	if got := PeriodKey("tidak-ada", time.Now()); got != "special" {
		t.Fatalf("got %q want %q", got, "special")
	}
}

func TestClaimWindowOpen(t *testing.T) {
	loc := period.JakartaLocation
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, loc)
	end := time.Date(2026, 10, 31, 23, 59, 0, 0, loc)

	cases := []struct {
		name        string
		now         time.Time
		isPublished bool
		want        bool
	}{
		{"sebelum mulai", start.Add(-time.Minute), true, false},
		{"tepat saat mulai", start, true, true},
		{"di tengah", time.Date(2026, 10, 15, 12, 0, 0, 0, loc), true, true},
		{"tepat saat berakhir", end, true, false},
		{"setelah berakhir", end.Add(time.Minute), true, false},
		{"belum dipublikasikan", time.Date(2026, 10, 15, 12, 0, 0, 0, loc), false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClaimWindowOpen(tc.now, start, end, tc.isPublished); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}

func TestCanClaim(t *testing.T) {
	cases := []struct {
		limit  int
		active int
		want   bool
	}{
		{1, 0, true},
		{1, 1, false},
		{1, 2, false},
		{3, 0, true},
		{3, 2, true},
		{3, 3, false},
		{3, 4, false},
		{0, 0, false},
		{-1, 0, false},
	}

	for _, tc := range cases {
		if got := CanClaim(tc.limit, tc.active); got != tc.want {
			t.Fatalf("limit %d aktif %d: got %v want %v", tc.limit, tc.active, got, tc.want)
		}
	}
}

func TestRequiredProofDanAutoApprove(t *testing.T) {
	if !RequiredProof(VerificationProofApproval) {
		t.Fatal("proof_approval wajib meminta bukti")
	}
	if RequiredProof(VerificationAuto) || RequiredProof(VerificationSelfClaim) {
		t.Fatal("mode selain proof_approval tidak boleh mewajibkan bukti")
	}

	if !IsAutoApproved(VerificationAuto) {
		t.Fatal("mode auto harus langsung disetujui")
	}
	// Klaim mandiri tetap berstatus pending supaya ada jejak yang dapat
	// diperiksa pengurus.
	if IsAutoApproved(VerificationSelfClaim) || IsAutoApproved(VerificationProofApproval) {
		t.Fatal("hanya mode auto yang boleh langsung disetujui")
	}
}

func TestValidateMissionTypeDanVerificationMode(t *testing.T) {
	for _, tipe := range []string{TypeDaily, TypeWeekly, TypeSpecial} {
		if err := ValidateMissionType(tipe); err != nil {
			t.Fatalf("jenis %q harus valid: %v", tipe, err)
		}
	}
	if err := ValidateMissionType("yearly"); err == nil {
		t.Fatal("jenis tidak dikenal harus ditolak")
	}

	for _, mode := range []string{VerificationAuto, VerificationSelfClaim, VerificationProofApproval} {
		if err := ValidateVerificationMode(mode); err != nil {
			t.Fatalf("mode %q harus valid: %v", mode, err)
		}
	}
	if err := ValidateVerificationMode("manual"); err == nil {
		t.Fatal("mode tidak dikenal harus ditolak")
	}

	for _, status := range []string{StatusPending, StatusApproved, StatusRejected, StatusCancelled} {
		if err := ValidateClaimStatus(status); err != nil {
			t.Fatalf("status %q harus valid: %v", status, err)
		}
	}
	if err := ValidateClaimStatus("expired"); err == nil {
		t.Fatal("status tidak dikenal harus ditolak")
	}
}

func TestParseWindowMenolakRentangTerbalik(t *testing.T) {
	if _, _, err := parseWindow(
		"2026-10-31T00:00:00+07:00",
		"2026-10-01T00:00:00+07:00",
	); err == nil {
		t.Fatal("rentang terbalik harus ditolak")
	}

	if _, _, err := parseWindow(
		"2026-10-01T00:00:00+07:00",
		"2026-10-01T00:00:00+07:00",
	); err == nil {
		t.Fatal("rentang kosong harus ditolak")
	}

	if _, _, err := parseWindow("2026-10-01", "2026-10-02"); err == nil {
		t.Fatal("waktu tanpa offset zona waktu harus ditolak")
	}

	start, end, err := parseWindow(
		"2026-10-01T00:00:00+07:00",
		"2026-10-31T23:59:00+07:00",
	)
	if err != nil {
		t.Fatalf("rentang valid ditolak: %v", err)
	}
	if !end.After(start) {
		t.Fatal("rentang valid harus menghasilkan end setelah start")
	}
}

func TestTrimPtr(t *testing.T) {
	str := func(s string) *string { return &s }

	if got := trimPtr(nil); got != nil {
		t.Fatalf("nil harus tetap nil, got %q", *got)
	}
	if got := trimPtr(str("   ")); got != nil {
		t.Fatalf("string kosong harus menjadi nil, got %q", *got)
	}
	if got := trimPtr(str("  bukti.jpg  ")); got == nil || *got != "bukti.jpg" {
		t.Fatalf("string harus dirapikan, got %v", got)
	}
}
