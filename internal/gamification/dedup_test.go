package gamification

import (
	"strings"
	"testing"
)

func TestProfileDedupKeyStabil(t *testing.T) {
	// Kunci yang memuat waktu atau referensi akan membuat reward "profil
	// lengkap" bisa dipanen berulang hanya dengan mengubah field profil.
	first := ProfileDedupKey("user-1")
	if second := ProfileDedupKey("user-1"); first != second {
		t.Fatalf("kunci harus stabil untuk user yang sama: %q != %q", first, second)
	}
	if want := "profile_complete:user-1"; first != want {
		t.Fatalf("kunci tidak sesuai: got %q want %q", first, want)
	}
}

func TestCheckInDedupKeyTerpisahPerAkunDanEvent(t *testing.T) {
	a := CheckInDedupKey("user-1", "event-a")
	if b := CheckInDedupKey("user-1", "event-b"); a == b {
		t.Fatal("event berbeda harus menghasilkan kunci berbeda")
	}
	if c := CheckInDedupKey("user-2", "event-a"); a == c {
		t.Fatal("akun berbeda harus menghasilkan kunci berbeda")
	}
	if want := "checkin:user-1:event-a"; a != want {
		t.Fatalf("kunci tidak sesuai: got %q want %q", a, want)
	}
}

func TestMissionDedupKeyTerpisahPerKlaim(t *testing.T) {
	a := MissionDedupKey("user-1", "claim-1")
	if b := MissionDedupKey("user-1", "claim-2"); a == b {
		t.Fatal("klaim berbeda harus menghasilkan kunci berbeda")
	}
	if want := "mission:user-1:claim-1"; a != want {
		t.Fatalf("kunci tidak sesuai: got %q want %q", a, want)
	}
}

func TestAdjustmentDedupKeySelaluUnik(t *testing.T) {
	// Koreksi admin tidak boleh saling menimpa: dua koreksi dengan nilai
	// identik harus tetap tercatat sebagai dua baris ledger.
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		key := AdjustmentDedupKey()
		if seen[key] {
			t.Fatalf("kunci koreksi terulang: %q", key)
		}
		seen[key] = true
		if !strings.HasPrefix(key, SourceAdjustment+":") {
			t.Fatalf("kunci koreksi harus berawalan %q: %q", SourceAdjustment, key)
		}
	}
}

func TestKunciMemuatAwalanSumberYangBerbeda(t *testing.T) {
	// Awalan source_type yang berbeda memastikan satu akun tidak bisa
	// menabrakkan kunci antar-sumber.
	prefixes := map[string]string{
		ProfileDedupKey("u"):      SourceProfileComplete,
		CheckInDedupKey("u", "e"): SourceCheckIn,
		MissionDedupKey("u", "c"): SourceMission,
	}
	for key, source := range prefixes {
		if !strings.HasPrefix(key, source+":") {
			t.Fatalf("kunci %q harus berawalan %q", key, source+":")
		}
	}
}
