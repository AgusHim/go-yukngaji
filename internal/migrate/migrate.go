// Package migrate menjalankan migrasi SQL berversi di atas koneksi GORM yang
// sudah dipakai aplikasi.
//
// Sebelumnya skema hanya dibuat lewat sql/migration.sql yang dijalankan manual
// dan tidak punya catatan versi. Runner ini menambahkan tabel schema_migrations
// supaya perubahan skema berikutnya bisa diterapkan berurutan, diulang dengan
// aman (semua migrasi memakai IF NOT EXISTS), dan dibalik bila perlu.
package migrate

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"
)

// Direction menentukan arah migrasi.
type Direction string

const (
	Up   Direction = "up"
	Down Direction = "down"
)

// Migration adalah satu berkas migrasi yang sudah dibaca dari disk.
type Migration struct {
	Version string
	Name    string
	Path    string
	SQL     string
}

// migrationsTable adalah tabel pencatat versi yang sudah diterapkan.
const migrationsTable = "schema_migrations"

// Load membaca berkas <version>_<name>.<direction>.sql dari dir, terurut naik
// berdasarkan versi.
func Load(dir string, direction Direction) ([]Migration, error) {
	pattern := filepath.Join(dir, "*."+string(direction)+".sql")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)

	migrations := make([]Migration, 0, len(paths))
	seen := make(map[string]string, len(paths))

	for _, path := range paths {
		base := filepath.Base(path)
		stem := strings.TrimSuffix(base, "."+string(direction)+".sql")

		version, name := stem, ""
		if idx := strings.Index(stem, "_"); idx >= 0 {
			version, name = stem[:idx], stem[idx+1:]
		}
		if version == "" {
			return nil, fmt.Errorf("migrate: nama berkas %q tidak memuat versi", base)
		}
		if prev, dup := seen[version]; dup {
			return nil, fmt.Errorf("migrate: versi %q dipakai dua kali (%s dan %s)", version, prev, base)
		}
		seen[version] = base

		content, errRead := os.ReadFile(path)
		if errRead != nil {
			return nil, errRead
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			Path:    path,
			SQL:     string(content),
		})
	}

	return migrations, nil
}

// EnsureTable membuat tabel pencatat versi bila belum ada.
func EnsureTable(db *gorm.DB) error {
	stmt := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
		version varchar PRIMARY KEY,
		name varchar NOT NULL DEFAULT '',
		applied_at timestamptz NOT NULL DEFAULT now()
	)`, migrationsTable)
	return db.Exec(stmt).Error
}

// AppliedVersions mengembalikan versi yang sudah tercatat, beserta urutannya.
func AppliedVersions(db *gorm.DB) ([]string, error) {
	var versions []string
	err := db.Raw(fmt.Sprintf("SELECT version FROM %s ORDER BY version ASC", migrationsTable)).Scan(&versions).Error
	if err != nil {
		return nil, err
	}
	return versions, nil
}

// Apply menjalankan seluruh pernyataan satu migrasi dan mencatat versinya
// dalam satu transaksi. Postgres mendukung DDL transaksional, jadi kegagalan
// di tengah migrasi tidak meninggalkan skema setengah jadi.
func Apply(db *gorm.DB, m Migration, direction Direction) error {
	statements := SplitStatements(m.SQL)
	if len(statements) == 0 {
		return fmt.Errorf("migrate: %s tidak berisi pernyataan", m.Path)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for idx, stmt := range statements {
			if err := tx.Exec(stmt).Error; err != nil {
				return fmt.Errorf("migrate: %s pernyataan #%d gagal: %w", m.Path, idx+1, err)
			}
		}

		if direction == Up {
			return tx.Exec(
				fmt.Sprintf("INSERT INTO %s (version, name) VALUES (?, ?) ON CONFLICT (version) DO NOTHING", migrationsTable),
				m.Version, m.Name,
			).Error
		}
		return tx.Exec(fmt.Sprintf("DELETE FROM %s WHERE version = ?", migrationsTable), m.Version).Error
	})
}

// Status melaporkan versi yang sudah dan belum diterapkan.
type Status struct {
	Version string
	Name    string
	Applied bool
}

// Run menerapkan migrasi ke arah direction.
//
// steps <= 0 berarti semua yang tertunda. Arah down selalu memakai urutan
// terbalik supaya migrasi terakhir dibalik lebih dulu.
func Run(db *gorm.DB, dir string, direction Direction, steps int, out io.Writer) (int, error) {
	if out == nil {
		out = io.Discard
	}
	if err := EnsureTable(db); err != nil {
		return 0, err
	}

	migrations, err := Load(dir, direction)
	if err != nil {
		return 0, err
	}
	applied, err := AppliedVersions(db)
	if err != nil {
		return 0, err
	}
	appliedSet := make(map[string]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	var todo []Migration
	switch direction {
	case Up:
		for _, m := range migrations {
			if !appliedSet[m.Version] {
				todo = append(todo, m)
			}
		}
	case Down:
		for i := len(migrations) - 1; i >= 0; i-- {
			if appliedSet[migrations[i].Version] {
				todo = append(todo, migrations[i])
			}
		}
	default:
		return 0, fmt.Errorf("migrate: arah %q tidak dikenal", direction)
	}

	if steps > 0 && len(todo) > steps {
		todo = todo[:steps]
	}
	if len(todo) == 0 {
		fmt.Fprintf(out, "migrate: tidak ada migrasi %s yang perlu dijalankan\n", direction)
		return 0, nil
	}

	for _, m := range todo {
		fmt.Fprintf(out, "migrate: %s %s %s\n", direction, m.Version, m.Name)
		if err := Apply(db, m, direction); err != nil {
			return 0, err
		}
	}
	fmt.Fprintf(out, "migrate: %d migrasi %s selesai\n", len(todo), direction)
	return len(todo), nil
}

// Baseline mencatat semua migrasi up sebagai sudah diterapkan tanpa
// menjalankannya.
//
// Dipakai sekali pada database produksi yang skemanya sudah dibuat lewat
// sql/migration.sql, supaya runner tidak mencoba menerapkan ulang perubahan
// yang sebenarnya sudah ada.
func Baseline(db *gorm.DB, dir string, out io.Writer) (int, error) {
	if out == nil {
		out = io.Discard
	}
	if err := EnsureTable(db); err != nil {
		return 0, err
	}

	migrations, err := Load(dir, Up)
	if err != nil {
		return 0, err
	}
	applied, err := AppliedVersions(db)
	if err != nil {
		return 0, err
	}
	appliedSet := make(map[string]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	count := 0
	for _, m := range migrations {
		if appliedSet[m.Version] {
			continue
		}
		fmt.Fprintf(out, "migrate: baseline %s %s\n", m.Version, m.Name)
		errInsert := db.Exec(
			fmt.Sprintf("INSERT INTO %s (version, name) VALUES (?, ?) ON CONFLICT (version) DO NOTHING", migrationsTable),
			m.Version, m.Name,
		).Error
		if errInsert != nil {
			return count, errInsert
		}
		count++
	}
	fmt.Fprintf(out, "migrate: %d migrasi dicatat sebagai baseline\n", count)
	return count, nil
}

// Report mengembalikan daftar migrasi beserta status penerapannya.
func Report(db *gorm.DB, dir string) ([]Status, error) {
	if err := EnsureTable(db); err != nil {
		return nil, err
	}
	migrations, err := Load(dir, Up)
	if err != nil {
		return nil, err
	}
	applied, err := AppliedVersions(db)
	if err != nil {
		return nil, err
	}
	appliedSet := make(map[string]bool, len(applied))
	for _, v := range applied {
		appliedSet[v] = true
	}

	statuses := make([]Status, 0, len(migrations))
	for _, m := range migrations {
		statuses = append(statuses, Status{
			Version: m.Version,
			Name:    m.Name,
			Applied: appliedSet[m.Version],
		})
	}
	return statuses, nil
}
