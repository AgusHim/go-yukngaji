package thread

import "testing"

func TestCanTransitionThread(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"published ke hidden", StatusPublished, StatusHidden, true},
		{"published ke deleted", StatusPublished, StatusDeleted, true},
		{"hidden ke published", StatusHidden, StatusPublished, true},
		{"hidden ke deleted", StatusHidden, StatusDeleted, true},
		{"published ke published ditolak", StatusPublished, StatusPublished, false},
		{"hidden ke hidden ditolak", StatusHidden, StatusHidden, false},
		{"deleted ke published ditolak", StatusDeleted, StatusPublished, false},
		{"deleted ke hidden ditolak", StatusDeleted, StatusHidden, false},
		{"deleted ke deleted ditolak", StatusDeleted, StatusDeleted, false},
		{"status asal tak dikenal", "draft", StatusHidden, false},
		{"status tujuan tak dikenal", StatusPublished, "draft", false},
		{"keduanya tak dikenal", "draft", "draft", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanTransitionThread(tc.from, tc.to); got != tc.want {
				t.Fatalf("CanTransitionThread(%q, %q) = %v, mau %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestCanTransitionThreadComment(t *testing.T) {
	// Komentar memakai himpunan status yang sama dengan thread; yang diuji di
	// sini adalah bahwa keduanya memang benar-benar berbagi aturan yang sama.
	for _, from := range []string{StatusPublished, StatusHidden, StatusDeleted} {
		for _, to := range []string{StatusPublished, StatusHidden, StatusDeleted} {
			want := CanTransitionThread(from, to)
			if got := CanTransitionThreadComment(from, to); got != want {
				t.Fatalf("komentar (%q, %q) = %v, thread = %v", from, to, got, want)
			}
		}
	}
}

func TestCanTransitionReport(t *testing.T) {
	cases := []struct {
		name string
		from string
		to   string
		want bool
	}{
		{"open ke actioned", ReportOpen, ReportActioned, true},
		{"open ke dismissed", ReportOpen, ReportDismissed, true},
		{"actioned ke dismissed ditolak", ReportActioned, ReportDismissed, false},
		{"dismissed ke actioned ditolak", ReportDismissed, ReportActioned, false},
		{"actioned ke open ditolak", ReportActioned, ReportOpen, false},
		{"dismissed ke dismissed ditolak", ReportDismissed, ReportDismissed, false},
		{"open ke open ditolak", ReportOpen, ReportOpen, false},
		{"status tak dikenal", "baru", ReportActioned, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := CanTransitionReport(tc.from, tc.to); got != tc.want {
				t.Fatalf("CanTransitionReport(%q, %q) = %v, mau %v", tc.from, tc.to, got, tc.want)
			}
		})
	}
}

func TestIsKnownStatus(t *testing.T) {
	for _, status := range []string{StatusPublished, StatusHidden, StatusDeleted} {
		if !IsKnownContentStatus(status) {
			t.Fatalf("%q seharusnya dikenal", status)
		}
	}
	if IsKnownContentStatus("draft") {
		t.Fatal("draft seharusnya tidak dikenal")
	}

	for _, status := range []string{ReportOpen, ReportActioned, ReportDismissed} {
		if !IsKnownReportStatus(status) {
			t.Fatalf("%q seharusnya dikenal", status)
		}
	}
	if IsKnownReportStatus("baru") {
		t.Fatal("baru seharusnya tidak dikenal")
	}
}
