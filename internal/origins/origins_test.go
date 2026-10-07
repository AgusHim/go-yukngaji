package origins

import (
	"os"
	"testing"
)

// TestMain menetapkan daftar origin sebelum paket ini membacanya, supaya
// hasil uji tidak bergantung pada environment mesin yang menjalankannya.
func TestMain(m *testing.M) {
	os.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:3000, https://ynsolo.id/, https://www.ynsolo.id")
	os.Exit(m.Run())
}

func TestAllowedParsesEnv(t *testing.T) {
	got := Allowed()
	want := []string{"http://localhost:3000", "https://ynsolo.id", "https://www.ynsolo.id"}

	if len(got) != len(want) {
		t.Fatalf("Allowed() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Allowed()[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestIsAllowed(t *testing.T) {
	tests := []struct {
		name   string
		origin string
		want   bool
	}{
		{name: "origin kosong (klien non-browser) diterima", origin: "", want: true},
		{name: "origin terdaftar diterima", origin: "http://localhost:3000", want: true},
		{name: "trailing slash dinormalisasi", origin: "https://ynsolo.id/", want: true},
		{name: "huruf besar kecil diabaikan", origin: "HTTPS://YNSOLO.ID", want: true},
		{name: "spasi di sekitar diabaikan", origin: "  https://www.ynsolo.id  ", want: true},
		{name: "situs lain ditolak", origin: "https://situs-jahat.example", want: false},
		{name: "subdomain liar ditolak", origin: "https://evil.ynsolo.id", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAllowed(tt.origin); got != tt.want {
				t.Errorf("IsAllowed(%q) = %v, want %v", tt.origin, got, tt.want)
			}
		})
	}
}

func TestAllowsCredentials(t *testing.T) {
	// Daftar eksplisit (tanpa wildcard) boleh memakai credentials.
	if !AllowsCredentials() {
		t.Error("daftar origin eksplisit harus mengizinkan credentials")
	}
}
