#!/usr/bin/env bash
#
# script.sh — pembantu operasi migrasi database.
#
# Membungkus `go run ./cmd/migrate` dengan langkah pengaman yang mudah
# terlupa bila diketik manual:
#
#   * `pg_dump` otomatis sebelum `up` (lewati dengan --no-backup)
#   * konfirmasi eksplisit sebelum `baseline`, `down`, dan `restore`
#   * flag `-n` dikirim SEBELUM subcommand
#
# Soal urutan flag: cmd/migrate memakai flag.Parse(), yang berhenti pada
# argumen pertama bukan-flag. Jadi `migrate up -n 1` mengabaikan `-n 1` dan
# menjalankan SEMUA migrasi tertunda, bukan satu. Urutan yang benar adalah
# `migrate -n 1 up`. Script ini selalu memakai urutan yang benar.
#
# Pemakaian:
#   ./script.sh status
#   ./script.sh up [--yes] [--no-backup]
#   ./script.sh down [-n N] [--yes]
#   ./script.sh baseline --yes
#   ./script.sh backup
#   ./script.sh restore <berkas.dump> [--yes]
#   ./script.sh help
#
# Prosedur lengkap: docs/migrations.md dan docs/deployment.md.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR" # cmd/migrate memuat .env dari direktori kerja

BACKUP_DIR="$SCRIPT_DIR/backups"
MIGRATE=(go run ./cmd/migrate)

ASSUME_YES=0
NO_BACKUP=0
STEPS=1

info() { printf '==> %s\n' "$*"; }
warn() { printf 'PERINGATAN: %s\n' "$*" >&2; }
die() { printf 'error: %s\n' "$*" >&2; exit 1; }

# trim membuang spasi di awal dan akhir sebuah nilai.
trim() {
	local s="$1"
	s="${s#"${s%%[![:space:]]*}"}"
	s="${s%"${s##*[![:space:]]}"}"
	printf '%s' "$s"
}

# load_env memuat .env dengan aturan yang sama seperti godotenv di sisi Go.
#
# Tidak memakai `source`: berkas .env proyek ini menulis `HOST =nilai` dengan
# spasi di sekitar `=`. godotenv menoleransi itu, sedangkan `source` bash
# memperlakukannya sebagai perintah `HOST` yang gagal. Parser ini juga:
#   * melewati baris kosong dan baris komentar penuh
#   * membuang tanda kutip pembungkus pada nilai
#   * TIDAK menimpa variabel yang sudah ada di environment, supaya
#     `DB_HOST=... ./script.sh up` tetap menang (sama seperti godotenv.Load)
load_env() {
	local file="$SCRIPT_DIR/.env"
	if [[ ! -f "$file" ]]; then
		warn ".env tidak ditemukan; memakai variabel environment yang ada"
		return 0
	fi

	local line key val
	while IFS= read -r line || [[ -n "$line" ]]; do
		line="$(trim "$line")"
		if [[ -z "$line" || "$line" == \#* || "$line" != *=* ]]; then
			continue
		fi

		key="$(trim "${line%%=*}")"
		val="$(trim "${line#*=}")"

		case "$val" in
		\"*\") val="${val#\"}" && val="${val%\"}" ;;
		\'*\') val="${val#\'}" && val="${val%\'}" ;;
		esac

		if [[ ! "$key" =~ ^[A-Za-z_][A-Za-z0-9_]*$ ]]; then
			continue
		fi
		# Jangan timpa environment yang sudah ada (perilaku godotenv.Load).
		if [[ -n "${!key:-}" ]]; then
			continue
		fi
		export "$key=$val"
	done <"$file"
}

# require_db_vars memastikan variabel koneksi yang dipakai pg_dump/pg_restore ada.
require_db_vars() {
	local v
	for v in DB_HOST DB_PORT DB_USERNAME DB_PASSWORD DB_DATABASE; do
		[[ -n "${!v:-}" ]] || die "$v belum diset (isi .env)"
	done
}

# confirm menanyakan persetujuan. --yes melewatinya; tanpa TTY, menolak.
confirm() {
	local prompt="$1" reply
	if ((ASSUME_YES)); then
		return 0
	fi
	if [[ ! -t 0 ]]; then
		die "butuh konfirmasi interaktif; tambahkan --yes bila memang disengaja"
	fi
	read -r -p "$prompt [y/N] " reply || true
	[[ "$reply" == "y" || "$reply" == "Y" ]]
}

# Jalur pg_dump/pg_restore.
#
# Bila biner-nya ada di host, pakai itu. Bila tidak (umum di macOS tanpa paket
# client PostgreSQL), jalankan lewat container postgres:alpine — Docker hanya
# perlu bisa menjangkau DB_HOST.
#
# Password selalu lewat PGPASSWORD, bukan argumen, supaya tidak muncul di `ps`.
# Untuk jalur container dipakai `-e PGPASSWORD` TANPA nilai agar Docker
# mewariskan nilainya dari environment client, bukan dari baris perintah.
PG_IMAGE="${PG_IMAGE:-postgres:alpine}"

need_docker() {
	command -v docker >/dev/null 2>&1 ||
		die "$1 tidak ada di host dan docker tidak ditemukan — pasang salah satunya"
}

do_backup() {
	require_db_vars
	mkdir -p "$BACKUP_DIR"
	local out="$BACKUP_DIR/backup-$(date +%F-%H%M%S).dump"
	info "mencadangkan $DB_DATABASE@$DB_HOST:$DB_PORT → $out"

	if command -v pg_dump >/dev/null 2>&1; then
		PGPASSWORD="$DB_PASSWORD" pg_dump \
			--host="$DB_HOST" --port="$DB_PORT" --username="$DB_USERNAME" \
			--format=custom --file="$out" "$DB_DATABASE"
	else
		need_docker pg_dump
		warn "pg_dump tidak ada di host; memakai container $PG_IMAGE"
		PGPASSWORD="$DB_PASSWORD" docker run --rm \
			-v "$BACKUP_DIR:/pgdata" -e PGPASSWORD "$PG_IMAGE" \
			pg_dump --host="$DB_HOST" --port="$DB_PORT" --username="$DB_USERNAME" \
			--format=custom --file="/pgdata/$(basename "$out")" "$DB_DATABASE"
	fi
	info "cadangan selesai: $out"
}

# urlencode meng-encode karakter di luar himpunan unreserved RFC 3986, sehingga
# password yang memuat ':', '@', '/', '#' atau sejenisnya tetap bisa dipakai
# di dalam DATABASE_URL. Tanpa ini, URL yang "kelihatan benar" gagal konek.
urlencode() {
	local s="$1" out="" c i
	for ((i = 0; i < ${#s}; i++)); do
		c="${s:i:1}"
		case "$c" in
		[A-Za-z0-9.~_-]) out+="$c" ;;
		*) out+="$(printf '%%%02X' "'$c")" ;;
		esac
	done
	printf '%s' "$out"
}

# cmd_url mencetak DATABASE_URL yang dipakai pg_dump/psql.
#
# Catatan: runner Go di proyek ini TIDAK memakai DATABASE_URL — db.NewDatabase
# membaca DB_HOST/DB_PORT/DB_USERNAME/DB_PASSWORD/DB_DATABASE. URL ini hanya
# untuk alat luar seperti pg_dump, psql, atau pg_restore.
cmd_url() {
	require_db_vars
	printf 'postgres://%s:%s@%s:%s/%s?sslmode=disable\n' \
		"$(urlencode "$DB_USERNAME")" "$(urlencode "$DB_PASSWORD")" \
		"$DB_HOST" "$DB_PORT" "$DB_DATABASE"
}

cmd_status() {
	"${MIGRATE[@]}" status
}

cmd_up() {
	if ((NO_BACKUP)); then
		warn "melewati pg_dump (--no-backup) — pastikan cadangan sudah ada di tempat lain"
	else
		do_backup
	fi
	confirm "Terapkan semua migrasi tertunda ke $DB_DATABASE@$DB_HOST?" || die "dibatalkan"
	"${MIGRATE[@]}" up
}

cmd_down() {
	warn "down membalik SKEMA, bukan data. Baris presence yang di-soft-delete"
	warn "migrasi 0004 dan hasil backfill 0003/0005 tidak akan kembali."
	warn "Bila ragu, pulihkan dari cadangan — bukan dari down."
	confirm "Batalkan $STEPS migrasi terakhir di $DB_DATABASE@$DB_HOST?" || die "dibatalkan"
	"${MIGRATE[@]}" -n "$STEPS" down
}

cmd_baseline() {
	warn "baseline mencatat SEMUA migrasi sebagai sudah diterapkan TANPA"
	warn "menjalankannya. Tabel dari 0007-0012 (community_profiles, xp_ledger,"
	warn "missions, campaigns, threads, products) TIDAK akan dibuat, dan"
	warn "index/constraint dari 0002/0006 juga tidak terpasang."
	warn "Pakai HANYA bila database memang sudah lengkap sampai 0012."
	confirm "Catat semua migrasi sebagai baseline di $DB_DATABASE@$DB_HOST?" || die "dibatalkan"
	"${MIGRATE[@]}" baseline
}

cmd_restore() {
	local file="$1"
	[[ -f "$file" ]] || die "berkas tidak ditemukan: $file"
	require_db_vars

	file="$(cd "$(dirname "$file")" && pwd)/$(basename "$file")"
	warn "restore --clean menghapus objek yang ada sebelum memuat ulang."
	confirm "Pulihkan $DB_DATABASE@$DB_HOST dari $file?" || die "dibatalkan"

	if command -v pg_restore >/dev/null 2>&1; then
		PGPASSWORD="$DB_PASSWORD" pg_restore --clean --if-exists \
			--host="$DB_HOST" --port="$DB_PORT" --username="$DB_USERNAME" \
			--dbname="$DB_DATABASE" "$file"
	else
		need_docker pg_restore
		warn "pg_restore tidak ada di host; memakai container $PG_IMAGE"
		PGPASSWORD="$DB_PASSWORD" docker run --rm \
			-v "$(dirname "$file"):/pgdata:ro" -e PGPASSWORD "$PG_IMAGE" \
			pg_restore --clean --if-exists \
			--host="$DB_HOST" --port="$DB_PORT" --username="$DB_USERNAME" \
			--dbname="$DB_DATABASE" "/pgdata/$(basename "$file")"
	fi
	info "restore selesai"
}

usage() {
	cat <<'EOF'
script.sh — pembantu operasi migrasi database.

Pemakaian:
  ./script.sh status
  ./script.sh up [--yes] [--no-backup]
  ./script.sh down [-n N] [--yes]
  ./script.sh baseline --yes
  ./script.sh backup
  ./script.sh restore <berkas.dump> [--yes]
  ./script.sh url
  ./script.sh help

Opsi:
  -n N          jumlah migrasi untuk `down` (default 1)
  --yes, -y     lewati konfirmasi interaktif
  --no-backup   jangan pg_dump otomatis sebelum `up`

`up` menjalankan pg_dump lebih dulu, kecuali --no-backup diberikan.
`baseline` berbahaya: mencatat semua migrasi sebagai diterapkan tanpa
menjalankannya. Lihat docs/migrations.md.

`url` mencetak DATABASE_URL yang memuat password — jangan disalin ke berkas
yang ikut ter-commit. Runner Go sendiri tidak memakai URL ini; ia membaca
variabel DB_* dari .env.
EOF
}

COMMAND="${1:-help}"
shift || true

REST=()
while (($#)); do
	case "$1" in
	-n)
		[[ $# -ge 2 ]] || die "-n butuh angka"
		STEPS="$2"
		[[ "$STEPS" =~ ^[0-9]+$ ]] || die "-n butuh angka, bukan '$STEPS'"
		shift 2
		;;
	--yes | -y)
		ASSUME_YES=1
		shift
		;;
	--no-backup)
		NO_BACKUP=1
		shift
		;;
	-h | --help)
		COMMAND=help
		shift
		;;
	-*)
		die "opsi tidak dikenal: $1"
		;;
	*)
		REST+=("$1")
		shift
		;;
	esac
done

load_env

case "$COMMAND" in
status) cmd_status ;;
url) cmd_url ;;
up) cmd_up ;;
down) cmd_down ;;
baseline) cmd_baseline ;;
backup) do_backup ;;
restore)
	[[ ${#REST[@]} -ge 1 ]] || die "pemakaian: ./script.sh restore <berkas.dump>"
	cmd_restore "${REST[0]}"
	;;
help | -h | --help) usage ;;
*) die "perintah tidak dikenal: $COMMAND (pakai: status|up|down|baseline|backup|restore|url|help)" ;;
esac
