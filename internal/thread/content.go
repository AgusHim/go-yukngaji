package thread

import (
	"regexp"
	"strings"
	"unicode"
)

// Batas panjang. Angka-angka ini adalah sumber kebenaran di Go; CHECK
// constraint di 0011_threads mengulangnya sebagai jaminan terakhir, bukan
// sebagai sumber pertama.
const (
	ThreadTitleMin = 3
	ThreadTitleMax = 140
	ThreadBodyMax  = 5000
	CommentBodyMax = 1000

	// MaxThreadLinks dan MaxCommentLinks menahan spam tautan. Judul tidak
	// boleh memuat tautan sama sekali — judul yang penuh tautan hampir selalu
	// spam, dan tidak ada kiriman sah yang membutuhkannya.
	MaxThreadLinks  = 3
	MaxCommentLinks = 1

	// MaxCharRun menahan "baguuuuuussss" dan pembatas deretan titik.
	MaxCharRun = 12

	// ReasonMin dan ReasonMax berlaku untuk seluruh alasan tindakan moderator.
	ReasonMin = 3
	ReasonMax = 500
)

// bannedWords adalah daftar kata yang ditolak.
//
// Sengaja pendek dan spesifik. Daftar ini memblokir kiriman secara otomatis
// tanpa mata manusia, jadi setiap kata yang bisa muncul wajar di percakapan
// biasa adalah false positive yang menutup mulut anggota yang tidak bersalah.
// Karena itu kata kabur seperti "promo" atau "diskon" tidak ada di sini —
// spam jualan ditahan lewat batas tautan, bukan lewat kosakata.
//
// Daftar ini ada di kode, bukan di tabel: mengubahnya perlu deploy. Itu
// keterbatasan yang disadari, bukan kelalaian.
var bannedWords = []string{
	"judi", "togel", "kasino", "pinjol",
	"bokep", "porno", "openbo", "situsjudi", "bandarjudi",
}

var linkPattern = regexp.MustCompile(`(?i)(https?://|www\.)\S*`)

// NormalizeContent merapikan teks sebelum disimpan dan sebelum diperiksa:
// baris baru diseragamkan, karakter kendali dibuang, spasi berlebih
// dirapatkan, dan baris kosong beruntun dipadatkan.
//
// Karena pemeriksaan panjang di CheckThreadContent bekerja atas bentuk yang
// sudah dirapikan, dan yang disimpan juga bentuk itu, panjang yang lolos
// validasi selalu sama dengan panjang yang dilihat constraint database.
func NormalizeContent(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")

	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		switch {
		case r == '\n':
			b.WriteRune(r)
		case r == '\t':
			b.WriteRune(' ')
		case unicode.IsControl(r), isInvisible(r):
			// dibuang
		default:
			b.WriteRune(r)
		}
	}

	lines := strings.Split(b.String(), "\n")
	for i, line := range lines {
		lines[i] = strings.Join(strings.Fields(line), " ")
	}
	out := strings.Join(lines, "\n")
	for strings.Contains(out, "\n\n\n") {
		out = strings.ReplaceAll(out, "\n\n\n", "\n\n")
	}
	return strings.TrimSpace(out)
}

// isInvisible menandai karakter tak terlihat yang sering dipakai menyisipkan
// kata terlarang agar lolos pencocokan teks.
func isInvisible(r rune) bool {
	switch r {
	case '\u200b', '\u200c', '\u200d', '\u2060', '\ufeff':
		return true
	}
	return false
}

// ContainsBannedWord mengembalikan kata terlarang pertama yang cocok, tanpa
// peduli huruf besar/kecil dan tanpa peduli pemisah yang disisipkan.
//
// Pencocokan dilakukan atas dua bentuk: per kata (setelah tanda baca dibuang)
// dan atas seluruh teks yang sudah dirapatkan. Bentuk kedua yang menangkap
// "s.p.a.m" dan "s p a m"; bentuk pertama yang menjaga "konsol otomatis"
// tidak cocok dengan kata pendek yang kebetulan melintasi batas kata.
func ContainsBannedWord(text string) (string, bool) {
	if len(bannedWords) == 0 {
		return "", false
	}

	tokens := make([]string, 0, 16)
	for _, field := range strings.Fields(strings.ToLower(text)) {
		tokens = append(tokens, keepAlnum(field))
	}
	joined := strings.Join(tokens, "")

	for _, word := range bannedWords {
		for _, token := range tokens {
			if token != "" && strings.Contains(token, word) {
				return word, true
			}
		}
		if strings.Contains(joined, word) {
			return word, true
		}
	}
	return "", false
}

// keepAlnum membuang semua karakter selain huruf dan angka.
func keepAlnum(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// CheckThreadContent memvalidasi judul dan badan thread.
//
// Menerima teks mentah: ia merapikannya lebih dulu, sehingga pemanggil boleh
// memeriksa input apa adanya. Yang tetap harus dilakukan pemanggil adalah
// menyimpan hasil NormalizeContent, supaya yang tersimpan sama dengan yang
// diperiksa.
func CheckThreadContent(title, body string) error {
	title = NormalizeContent(title)
	body = NormalizeContent(body)

	if err := checkLength("Judul", title, ThreadTitleMin, ThreadTitleMax); err != nil {
		return err
	}
	if err := checkLength("Isi", body, 1, ThreadBodyMax); err != nil {
		return err
	}
	if n := len(linkPattern.FindAllString(title, -1)); n > 0 {
		return invalid("Judul tidak boleh memuat tautan.")
	}
	if n := len(linkPattern.FindAllString(body, -1)); n > MaxThreadLinks {
		return invalid("Isi memuat terlalu banyak tautan (maksimal %d).", MaxThreadLinks)
	}
	if word, ok := ContainsBannedWord(title); ok {
		return invalid("Judul memuat kata yang tidak diperbolehkan: %q.", word)
	}
	if word, ok := ContainsBannedWord(body); ok {
		return invalid("Isi memuat kata yang tidak diperbolehkan: %q.", word)
	}
	if hasRepeatedRun(title) || hasRepeatedRun(body) {
		return invalid("Teks memuat pengulangan karakter yang berlebihan.")
	}
	return nil
}

// CheckCommentContent memvalidasi badan komentar dengan aturan yang sama,
// hanya batasnya yang berbeda.
func CheckCommentContent(body string) error {
	body = NormalizeContent(body)

	if err := checkLength("Komentar", body, 1, CommentBodyMax); err != nil {
		return err
	}
	if n := len(linkPattern.FindAllString(body, -1)); n > MaxCommentLinks {
		return invalid("Komentar memuat terlalu banyak tautan (maksimal %d).", MaxCommentLinks)
	}
	if word, ok := ContainsBannedWord(body); ok {
		return invalid("Komentar memuat kata yang tidak diperbolehkan: %q.", word)
	}
	if hasRepeatedRun(body) {
		return invalid("Teks memuat pengulangan karakter yang berlebihan.")
	}
	return nil
}

// RequireReason memvalidasi alasan tindakan moderator. Dipakai bersama oleh
// sembunyikan, hapus, dan batasi akun, supaya tidak ada satu pun keputusan yang
// bisa tercatat tanpa alasan.
func RequireReason(reason string) error {
	trimmed := strings.TrimSpace(reason)
	if trimmed == "" {
		return invalid("Alasan tindakan wajib diisi.")
	}
	if n := len([]rune(trimmed)); n < ReasonMin {
		return invalid("Alasan tindakan minimal %d karakter.", ReasonMin)
	}
	if n := len([]rune(trimmed)); n > ReasonMax {
		return invalid("Alasan tindakan maksimal %d karakter.", ReasonMax)
	}
	return nil
}

// Excerpt memotong teks untuk ditampilkan di daftar. Baris baru dirapatkan
// supaya ringkasan tetap satu paragraf.
func Excerpt(text string, max int) string {
	flat := strings.Join(strings.Fields(text), " ")
	if max <= 0 {
		return flat
	}
	runes := []rune(flat)
	if len(runes) <= max {
		return flat
	}
	return strings.TrimSpace(string(runes[:max])) + "…"
}

func checkLength(label, value string, min, max int) error {
	n := len([]rune(value))
	if n < min {
		if min == 1 {
			return invalid("%s tidak boleh kosong.", label)
		}
		return invalid("%s minimal %d karakter.", label, min)
	}
	if n > max {
		return invalid("%s maksimal %d karakter.", label, max)
	}
	return nil
}

// hasRepeatedRun melaporkan apakah ada satu karakter yang diulang melebihi
// MaxCharRun kali berturut-turut.
func hasRepeatedRun(text string) bool {
	var prev rune
	run := 0
	for _, r := range text {
		if r == prev {
			run++
			if run > MaxCharRun {
				return true
			}
			continue
		}
		prev = r
		run = 1
	}
	return false
}
