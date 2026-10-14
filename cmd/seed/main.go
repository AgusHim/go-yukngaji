// Command seed menjalankan berkas SQL di sql/seeds secara berurutan.
//
// Penggunaan:
//
//	go run ./cmd/seed                    # jalankan semua berkas di sql/seeds
//	go run ./cmd/seed -dir sql/seeds     # tentukan direktori lain
//
// Berbeda dari cmd/migrate, seed tidak mencatat apa pun di schema_migrations:
// tiap berkas wajib idempoten sendiri (mis. ON CONFLICT DO NOTHING), sehingga
// menjalankannya berulang kali aman. Karena itu seed juga tidak punya
// perintah "down" — untuk membatalkan, jalankan SQL pembaliknya sendiri
// (lihat komentar di kepala tiap berkas).
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"sort"

	"mainyuk/db"

	"github.com/joho/godotenv"
)

func main() {
	dir := flag.String("dir", "sql/seeds", "direktori berkas seed")
	flag.Parse()

	// .env bersifat opsional: di CI/produksi variabel bisa datang dari
	// environment langsung.
	if err := godotenv.Load(); err != nil {
		log.Printf("seed: .env tidak dimuat (%v), memakai environment yang ada", err)
	}

	files, err := filepath.Glob(filepath.Join(*dir, "*.sql"))
	if err != nil {
		log.Fatalf("seed: gagal membaca %s: %v", *dir, err)
	}
	if len(files) == 0 {
		log.Fatalf("seed: tidak ada berkas .sql di %s", *dir)
	}
	sort.Strings(files)

	database, err := db.NewDatabase()
	if err != nil {
		log.Fatalf("seed: gagal konek database: %v", err)
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("seed: gagal membaca %s: %v", file, err)
		}
		if err := database.Exec(string(content)).Error; err != nil {
			log.Fatalf("seed: %s gagal: %v", file, err)
		}
		log.Printf("seed: %s diterapkan", filepath.Base(file))
	}
}
