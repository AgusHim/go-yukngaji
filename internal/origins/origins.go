// Package origins memusatkan daftar origin yang diizinkan mengakses API.
//
// Daftar dibaca sekali dari env CORS_ALLOWED_ORIGINS (dipisah koma). Nilai
// defaultnya adalah origin web yang dikenal, bukan "*", supaya cookie dan
// header Authorization tidak bisa dikirim dari situs sembarangan.
package origins

import (
	"os"
	"strings"
	"sync"
)

// DefaultAllowed dipakai bila CORS_ALLOWED_ORIGINS tidak diset.
var DefaultAllowed = []string{
	"http://localhost:3000",
	"https://ynsolo.id",
	"https://www.ynsolo.id",
}

var (
	once    sync.Once
	allowed []string
)

// Allowed mengembalikan daftar origin yang diizinkan.
func Allowed() []string {
	once.Do(func() {
		raw := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
		if raw == "" {
			allowed = append(allowed, DefaultAllowed...)
			return
		}
		for _, item := range strings.Split(raw, ",") {
			if item = normalize(item); item != "" {
				allowed = append(allowed, item)
			}
		}
		if len(allowed) == 0 {
			allowed = append(allowed, DefaultAllowed...)
		}
	})
	return allowed
}

// IsAllowed melaporkan apakah origin boleh mengakses API.
//
// Origin kosong (klien non-browser, mis. aplikasi ranger atau curl) diterima;
// yang dibatasi adalah permintaan lintas situs dari browser.
func IsAllowed(origin string) bool {
	origin = normalize(origin)
	if origin == "" {
		return true
	}
	for _, item := range Allowed() {
		if item == "*" || strings.EqualFold(item, origin) {
			return true
		}
	}
	return false
}

// AllowsCredentials melaporkan apakah daftar memakai wildcard. Kombinasi
// "*" + credentials tidak sah menurut spesifikasi CORS, jadi pemanggil harus
// mematikan credentials bila wildcard dipakai.
func AllowsCredentials() bool {
	for _, item := range Allowed() {
		if item == "*" {
			return false
		}
	}
	return true
}

func normalize(origin string) string {
	return strings.TrimSuffix(strings.TrimSpace(origin), "/")
}
