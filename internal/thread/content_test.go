package thread

import (
	"strings"
	"testing"
)

func TestNormalizeContent(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"teks kosong", "", ""},
		{"spasi berlebih", "  banyak   spasi  ", "banyak spasi"},
		{"baris baru CRLF diseragamkan", "a\r\nb", "a\nb"},
		{"baris baru CR diseragamkan", "a\rb", "a\nb"},
		{"baris kosong beruntun dipadatkan", "a\n\n\n\nb", "a\n\nb"},
		{"tab menjadi spasi", "a\tb", "a b"},
		{"karakter kendali dibuang", "halo\x00dunia", "halodunia"},
		{"karakter tak terlihat dibuang", "to\u200bgel", "togel"},
		{"teks hanya spasi", "   \n\t ", ""},
		{"spasi tepi baris dibuang", "  a  \n  b  ", "a\nb"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := NormalizeContent(tc.in); got != tc.want {
				t.Fatalf("NormalizeContent(%q) = %q, mau %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestContainsBannedWord(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		wantWord string
		wantOK   bool
	}{
		{"teks bersih", "Ayo ikut kerja bakti minggu ini", "", false},
		{"kata terlarang utuh", "togel", "togel", true},
		{"huruf besar", "TOGEL", "togel", true},
		{"huruf campur", "ToGeL", "togel", true},
		{"sebagian kata", "menjudikan", "judi", true},
		{"pemisah titik", "t.o.g.e.l", "togel", true},
		{"pemisah spasi", "t o g e l", "togel", true},
		{"kata lain yang memuat potongan", "konsol otomatis", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			word, ok := ContainsBannedWord(tc.in)
			if ok != tc.wantOK {
				t.Fatalf("ContainsBannedWord(%q) ok = %v, mau %v", tc.in, ok, tc.wantOK)
			}
			if word != tc.wantWord {
				t.Fatalf("ContainsBannedWord(%q) = %q, mau %q", tc.in, word, tc.wantWord)
			}
		})
	}

	t.Run("daftar kosong tidak pernah cocok", func(t *testing.T) {
		saved := bannedWords
		bannedWords = nil
		defer func() { bannedWords = saved }()

		if _, ok := ContainsBannedWord("togel"); ok {
			t.Fatal("daftar kosong seharusnya tidak menemukan apa pun")
		}
	})
}

func TestCheckThreadContent(t *testing.T) {
	okTitle := strings.Repeat("ab", 70)  // 140 karakter
	longTitle := okTitle + "c"           // 141 karakter
	okBody := strings.Repeat("ab", 2500) // 5000 karakter
	longBody := okBody + "c"             // 5001 karakter

	cases := []struct {
		name    string
		title   string
		body    string
		wantErr bool
	}{
		{"judul dan isi wajar", "Kerja bakti", "Ayo ikut kerja bakti minggu ini.", false},
		{"judul 3 karakter", "abc", "isi", false},
		{"judul 2 karakter", "ab", "isi", true},
		{"judul 140 karakter", okTitle, "isi", false},
		{"judul 141 karakter", longTitle, "isi", true},
		{"judul hanya spasi", "   ", "isi", true},
		{"isi kosong", "Kerja bakti", "", true},
		{"isi 1 karakter", "Kerja bakti", "a", false},
		{"isi 5000 karakter", "Kerja bakti", okBody, false},
		{"isi 5001 karakter", "Kerja bakti", longBody, true},
		{"kata terlarang di judul", "Promo togel", "Isi biasa.", true},
		{"kata terlarang di isi", "Kerja bakti", "Ayo main togel.", true},
		{"kata terlarang disamarkan", "Kerja bakti", "Ayo main t.o.g.e.l.", true},
		{"tautan di judul", "Lihat https://contoh.id", "Isi biasa.", true},
		{"tiga tautan di isi", "Kerja bakti", "a https://a.id b https://b.id c https://c.id", false},
		{"empat tautan di isi", "Kerja bakti", "a https://a.id b https://b.id c https://c.id d https://d.id", true},
		{"karakter berulang", "Kerja bakti", strings.Repeat("a", 13), true},
		{"karakter kendali dibersihkan lebih dulu", "Kerja bakti", "halo\x00dunia", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckThreadContent(tc.title, tc.body)
			if tc.wantErr && err == nil {
				t.Fatal("mau galat, tidak ada")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("tidak mau galat, dapat: %v", err)
			}
			if err != nil && !IsValidationError(err) {
				t.Fatalf("galat validasi tidak ditandai: %v", err)
			}
		})
	}
}

func TestCheckCommentContent(t *testing.T) {
	okBody := strings.Repeat("ab", 500) // 1000 karakter
	longBody := okBody + "c"            // 1001 karakter

	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"komentar wajar", "Setuju sekali.", false},
		{"komentar kosong", "", true},
		{"komentar hanya spasi", "   ", true},
		{"komentar 1000 karakter", okBody, false},
		{"komentar 1001 karakter", longBody, true},
		{"kata terlarang", "ayo main togel", true},
		{"satu tautan", "lihat https://contoh.id", false},
		{"dua tautan", "lihat https://a.id dan https://b.id", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckCommentContent(tc.body)
			if tc.wantErr && err == nil {
				t.Fatal("mau galat, tidak ada")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("tidak mau galat, dapat: %v", err)
			}
		})
	}
}

func TestRequireReason(t *testing.T) {
	cases := []struct {
		name    string
		reason  string
		wantErr bool
	}{
		{"kosong", "", true},
		{"hanya spasi", "   \t ", true},
		{"terlalu pendek", "ab", true},
		{"pas batas bawah", "abc", false},
		{"wajar", "Mengandung ujaran kebencian", false},
		{"terlalu panjang", strings.Repeat("a", ReasonMax+1), true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := RequireReason(tc.reason)
			if tc.wantErr && err == nil {
				t.Fatal("mau galat, tidak ada")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("tidak mau galat, dapat: %v", err)
			}
		})
	}
}

func TestExcerpt(t *testing.T) {
	cases := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{"lebih pendek dari batas", "halo dunia", 50, "halo dunia"},
		{"baris baru dirapatkan", "halo\ndunia", 50, "halo dunia"},
		{"dipotong", "halo dunia", 4, "halo…"},
		{"batas nol berarti tanpa potongan", "halo dunia", 0, "halo dunia"},
		{"teks kosong", "", 10, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Excerpt(tc.in, tc.max); got != tc.want {
				t.Fatalf("Excerpt(%q, %d) = %q, mau %q", tc.in, tc.max, got, tc.want)
			}
		})
	}
}
