package gamification

import (
	"testing"
	"time"

	"mainyuk/internal/period"
)

func TestNormalizePagination(t *testing.T) {
	cases := []struct {
		name          string
		page, perPage int
		wantPage      int
		wantPerPage   int
	}{
		{"kosong", 0, 0, 1, DefaultPerPage},
		{"negatif", -5, -1, 1, DefaultPerPage},
		{"wajar", 3, 25, 3, 25},
		{"per_page di atas batas", 1, 5000, 1, MaxPerPage},
		{"tepat di batas", 2, MaxPerPage, 2, MaxPerPage},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			page, perPage := NormalizePagination(tc.page, tc.perPage)
			if page != tc.wantPage || perPage != tc.wantPerPage {
				t.Fatalf("got (%d,%d) want (%d,%d)", page, perPage, tc.wantPage, tc.wantPerPage)
			}
		})
	}
}

func TestIsValidPeriod(t *testing.T) {
	for _, p := range []string{PeriodWeekly, PeriodMonthly, PeriodAllTime} {
		if !IsValidPeriod(p) {
			t.Fatalf("periode %q harus valid", p)
		}
	}
	for _, p := range []string{"", "daily", "yearly", "WEEKLY"} {
		if IsValidPeriod(p) {
			t.Fatalf("periode %q harus ditolak", p)
		}
	}
}

func TestPeriodRangeAllTimeTanpaBatas(t *testing.T) {
	from, to := PeriodRange(PeriodAllTime, time.Now(), period.JakartaLocation)
	if from != nil || to != nil {
		t.Fatalf("all_time harus tanpa batas, got from=%v to=%v", from, to)
	}
}

func TestPeriodRangeMingguanMemakaiAwalMingguSenin(t *testing.T) {
	loc := period.JakartaLocation
	// Kamis 2026-10-01 12:00 WIB.
	now := time.Date(2026, 10, 1, 12, 0, 0, 0, loc)

	from, to := PeriodRange(PeriodWeekly, now, loc)
	if from == nil || to == nil {
		t.Fatal("periode mingguan harus punya batas")
	}

	wantFrom := time.Date(2026, 9, 28, 0, 0, 0, 0, loc) // Senin
	wantTo := time.Date(2026, 10, 5, 0, 0, 0, 0, loc)   // Senin berikutnya
	if !from.Equal(wantFrom) {
		t.Fatalf("awal minggu: got %v want %v", from, wantFrom)
	}
	if !to.Equal(wantTo) {
		t.Fatalf("akhir minggu: got %v want %v", to, wantTo)
	}
}

func TestPeriodRangeBulanan(t *testing.T) {
	loc := period.JakartaLocation
	now := time.Date(2026, 10, 15, 9, 30, 0, 0, loc)

	from, to := PeriodRange(PeriodMonthly, now, loc)
	if from == nil || to == nil {
		t.Fatal("periode bulanan harus punya batas")
	}
	if want := time.Date(2026, 10, 1, 0, 0, 0, 0, loc); !from.Equal(want) {
		t.Fatalf("awal bulan: got %v want %v", from, want)
	}
	if want := time.Date(2026, 11, 1, 0, 0, 0, 0, loc); !to.Equal(want) {
		t.Fatalf("akhir bulan: got %v want %v", to, want)
	}
}

func TestDisplayAlias(t *testing.T) {
	str := func(s string) *string { return &s }

	if got := displayAlias(nil); got != "Anggota" {
		t.Fatalf("alias kosong harus netral, got %q", got)
	}
	if got := displayAlias(str("   ")); got != "Anggota" {
		t.Fatalf("alias berisi spasi harus netral, got %q", got)
	}
	if got := displayAlias(str("Rani")); got != "Rani" {
		t.Fatalf("alias terisi harus dipakai, got %q", got)
	}
}
