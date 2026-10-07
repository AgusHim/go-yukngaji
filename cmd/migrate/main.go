// Command migrate menerapkan migrasi SQL berversi.
//
// Penggunaan:
//
//	go run ./cmd/migrate status
//	go run ./cmd/migrate up            # terapkan semua migrasi tertunda
//	go run ./cmd/migrate up -n 1       # terapkan satu migrasi saja
//	go run ./cmd/migrate down -n 1     # batalkan satu migrasi terakhir
//	go run ./cmd/migrate baseline      # catat migrasi sebagai sudah ada
//
// Jalankan `pg_dump` sebelum `up` pada database berisi data produksi
// (lihat docs/migrations.md).
package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"mainyuk/db"
	"mainyuk/internal/migrate"

	"github.com/joho/godotenv"
)

func main() {
	dir := flag.String("dir", "sql/migrations", "direktori berkas migrasi")
	steps := flag.Int("n", 0, "jumlah migrasi yang dijalankan (0 = semua untuk up, 1 untuk down)")
	flag.Parse()

	command := flag.Arg(0)
	if command == "" {
		command = "status"
	}

	// .env bersifat opsional: di CI/produksi variabel bisa datang dari
	// environment langsung.
	if err := godotenv.Load(); err != nil {
		log.Printf("migrate: .env tidak dimuat (%v), memakai environment yang ada", err)
	}

	database, err := db.NewDatabase()
	if err != nil {
		log.Fatalf("migrate: gagal konek database: %v", err)
	}

	switch command {
	case "status":
		statuses, errReport := migrate.Report(database, *dir)
		if errReport != nil {
			log.Fatalf("migrate: %v", errReport)
		}
		for _, s := range statuses {
			mark := "[ ]"
			if s.Applied {
				mark = "[x]"
			}
			fmt.Printf("%s %s %s\n", mark, s.Version, s.Name)
		}

	case "up", "down":
		direction := migrate.Direction(command)
		if command == "down" && *steps == 0 {
			*steps = 1
		}
		if _, errRun := migrate.Run(database, *dir, direction, *steps, os.Stdout); errRun != nil {
			log.Fatalf("migrate: %v", errRun)
		}

	case "baseline":
		if _, errBase := migrate.Baseline(database, *dir, os.Stdout); errBase != nil {
			log.Fatalf("migrate: %v", errBase)
		}

	default:
		log.Fatalf("migrate: perintah %q tidak dikenal (pakai: status|up|down|baseline)", command)
	}
}
