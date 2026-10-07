package migrate

import (
	"strings"
	"unicode"
)

// SplitStatements memecah skrip SQL menjadi pernyataan-pernyataan terpisah.
//
// Pemecahan dilakukan dengan sadar konteks supaya titik koma di dalam string,
// komentar, atau dollar-quoted block ($$ ... $$) tidak dianggap pemisah:
//
//   - komentar baris  -- ...
//   - komentar blok   /* ... */
//   - string literal  '...'  dengan escape ”
//   - identifier      "..."  dengan escape ""
//   - dollar quote    $$ ... $$, $tag$ ... $tag$
//
// Komentar dibuang dari hasil. String dan identifier disalin apa adanya.
func SplitStatements(script string) []string {
	var (
		out []string
		buf strings.Builder
	)
	runes := []rune(script)
	n := len(runes)

	flush := func() {
		if stmt := strings.TrimSpace(buf.String()); stmt != "" {
			out = append(out, stmt)
		}
		buf.Reset()
	}

	for i := 0; i < n; {
		ch := runes[i]

		switch {
		case ch == '-' && i+1 < n && runes[i+1] == '-':
			// Komentar baris: sisanya sampai newline dibuang, newline tetap
			// ditulis supaya dua token tidak menempel.
			for i < n && runes[i] != '\n' {
				i++
			}

		case ch == '/' && i+1 < n && runes[i+1] == '*':
			i += 2
			for i+1 < n && !(runes[i] == '*' && runes[i+1] == '/') {
				i++
			}
			if i+1 < n {
				i += 2
			} else {
				i = n
			}
			buf.WriteRune(' ')

		case ch == '\'':
			i = copyQuoted(runes, i, '\'', &buf)

		case ch == '"':
			i = copyQuoted(runes, i, '"', &buf)

		case ch == '$':
			tag, ok := dollarTag(runes, i)
			if !ok {
				buf.WriteRune(ch)
				i++
				break
			}
			buf.WriteString(tag)
			i += len(tag)
			for i < n {
				if runes[i] == '$' {
					if end, okEnd := dollarTag(runes, i); okEnd && end == tag {
						buf.WriteString(tag)
						i += len(tag)
						break
					}
				}
				buf.WriteRune(runes[i])
				i++
			}

		case ch == ';':
			flush()
			i++

		default:
			buf.WriteRune(ch)
			i++
		}
	}

	flush()
	return out
}

// copyQuoted menyalin literal/identifier yang dimulai pada index i (yang
// berisi quote) sampai quote penutup, lalu mengembalikan index berikutnya.
// Dua quote berturut-turut di dalam dianggap satu karakter quote.
func copyQuoted(runes []rune, i int, quote rune, buf *strings.Builder) int {
	n := len(runes)
	buf.WriteRune(quote)
	i++
	for i < n {
		if runes[i] == quote {
			if i+1 < n && runes[i+1] == quote {
				buf.WriteRune(quote)
				buf.WriteRune(quote)
				i += 2
				continue
			}
			buf.WriteRune(quote)
			return i + 1
		}
		buf.WriteRune(runes[i])
		i++
	}
	return i
}

// dollarTag membaca tag dollar-quote pada posisi i ("$$", "$tag$").
// Mengembalikan ok=false untuk placeholder seperti $1 atau tanda $ biasa.
func dollarTag(runes []rune, i int) (string, bool) {
	if i >= len(runes) || runes[i] != '$' {
		return "", false
	}
	j := i + 1
	for j < len(runes) && (unicode.IsLetter(runes[j]) || unicode.IsDigit(runes[j]) || runes[j] == '_') {
		j++
	}
	if j < len(runes) && runes[j] == '$' {
		return string(runes[i : j+1]), true
	}
	return "", false
}
