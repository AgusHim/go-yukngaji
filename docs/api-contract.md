# Kontrak API

Rujukan bentuk request/response untuk kedua repositori: `server/` (Go) dan
`web/` (Next.js). Dokumen ini **dipelihara tangan** — tidak ada generator tipe.
Yang membuatnya tetap dapat dipercaya adalah kebiasaan membandingkannya
dengan kode setiap kali handler berubah, bukan otomatisasi.

Untuk siapa endpoint boleh diakses (peran dan izin), lihat
`docs/roles-and-permissions.md`. Dokumen ini tidak menduplikasi tabel itu; ia
melengkapinya dengan **bentuk data**.

## Konvensi umum

Berlaku untuk semua endpoint. Bagian per modul tidak mengulanginya.

### Grup dan prefiks

| Prefiks | Siapa | Middleware |
| --- | --- | --- |
| `/api` | Terbuka untuk tamu | `AuthOptionalUser` (tidak pernah menolak) |
| `/user_api` | Anggota yang sudah masuk | `AuthUser` |
| `/ranger_api` | Ranger | `AuthRanger` |
| `/admin_api` | Pengurus | `AuthAdmin` (admin) atau `AuthPJ` (admin + pj), per route |

Beberapa endpoint `/api` tetap menuntut login meski berada di grup terbuka
(mis. menulis komentar memakai `AuthUser`); middleware per route selalu
menang atas prefiks grup.

### Autentikasi

Header `Authorization: Bearer <access_token>`. Token didapat dari
`POST /api/login`, `POST /api/auth/otp/verify`, atau callback Google.

WebSocket (`/ws/events/:id`) menerima token lewat query `?token=<JWT>` **atau**
header `Authorization`, karena browser tidak dapat menyetel header saat
handshake.

### Pagination

Query `page` dan `per_page`, dinormalkan `gamification.NormalizePagination`
(`internal/gamification/leaderboard.go:49`):

- `page < 1` → `1`
- `per_page < 1` → `20` (`DefaultPerPage`)
- `per_page > 100` → `100` (`MaxPerPage`)

Amplop respons selalu memuat kunci koleksi, lalu `page`, `per_page`,
`has_more`. Nilai `page`/`per_page` yang dikembalikan adalah hasil normalisasi,
bukan yang dikirim klien. `has_more` dihitung dengan mengambil `per_page + 1`
baris, sehingga tidak ada query `COUNT(*)` terpisah.

```json
{ "products": [ ... ], "page": 1, "per_page": 20, "has_more": true }
```

Kunci koleksinya berbeda per endpoint (`products`, `orders`, `donations`,
`threads`, `audit_logs`, …). Tabel di tiap modul menyebutnya.

**Tidak semua daftar berhalaman.** Beberapa endpoint mengembalikan seluruh
koleksi sekaligus tanpa `page`/`per_page` — ditandai "tanpa pagination" di
tabel modul.

### Galat

`internal/httperr.JSON(c, err)` memetakan error ke `{"error": "<pesan>"}`:

| Sumber | Status | Contoh isi |
| --- | --- | --- |
| `ratelimit.ErrTooManyRequests` | 429 | `TooManyRequests` |
| `apperr.ErrUnauthorized` | 401 | `NotAuthrized` (salah ketik di kode, sengaja tidak diubah) |
| `apperr.ErrForbidden` | 403 | `Forbidden` |
| `apperr.ErrNotFound` | 404 | `NotFound` |
| `apperr.ErrInvalidRequest` | 400 | `InvalidRequest` |
| error apa pun dengan `IsValidation() == true` | 400 | pesan spesifik, mis. `alasan wajib diisi` |
| selain itu | 500 | pesan internal |

Galat validasi membawa pesan berbahasa Indonesia yang layak ditampilkan ke
pengguna. Galat 500 tidak — jangan pernah menampilkan isinya apa adanya.

Body JSON yang gagal di-*bind* **tidak** lewat `httperr`; handler menulis
sendiri `400 {"error":"Invalid JSON"}`. Beberapa handler lama menulisnya
sebagai **500** (lihat catatan di modulnya).

### Rate limit

In-memory, **satu proses** (`internal/ratelimit`). Menjalankan lebih dari satu
instance backend berarti setiap instance punya hitungannya sendiri, sehingga
batas efektifnya berlipat. Diperiksa **di dalam service**, bukan sebagai
middleware — jadi ia hanya berlaku pada operasi yang memang memanggilnya.

Kunci: `user:<id>` bila identitas diketahui, selain itu `ip:<clientIP>`.
Jendela bergulir. Pelanggaran → 429 `{"error":"TooManyRequests"}`.

Batas per operasi ada di tabel modulnya masing-masing.

### Aturan retry

Ini bagian terpenting dari kontrak, karena menentukan apakah klien boleh
mengulang permintaan yang gagal di jaringan.

**Operasi transisi status bersifat idempoten.** Perpindahan status dikerjakan
dengan `UPDATE ... WHERE id = ? AND status = ?`, lalu diperiksa
`RowsAffected`. Bila `RowsAffected == 0`, artinya status sudah berpindah lebih
dulu — dan itu diperlakukan sebagai **sukses**, bukan galat. Konsekuensinya:

- Mengirim ulang permintaan yang sama menghasilkan 200 dengan data terkini.
- Klien boleh mengulang tanpa takut menggandakan efek samping (XP, stok,
  refund).

**Pembuatan bersifat idempoten lewat kunci dedup.** Beberapa operasi menerima
`client_token` dari klien (mis. `POST /user_api/donations`). Unique index
parsial + `INSERT ... ON CONFLICT DO NOTHING` memastikan token yang sama tidak
pernah menghasilkan dua baris. Kirim ulang token yang sama → 200/201 dengan
baris yang sudah ada.

**Yang tidak boleh diulang otomatis:** operasi yang mengubah status ke nilai
berbeda (mis. `reject` setelah `confirm`) akan dijawab 400
`pesanan sudah diputuskan`. Mengulang permintaan yang **gagal validasi** tidak
akan berhasil sampai isinya diperbaiki.

### Soft delete

Model mendeklarasikan `DeletedAt *time.Time` sendiri, **bukan**
`gorm.DeletedAt`. Artinya `db.Delete` melakukan **hard delete**, dan setiap
pembacaan menambahkan `.Where("deleted_at IS NULL")` secara manual.

Akibat yang perlu diketahui klien: baris yang terhapus **tidak dapat
dibedakan dari baris yang tidak pernah ada** — keduanya menjawab 404. Ini
disengaja, termasuk untuk konten yang disembunyikan moderator.

### Konversi ke TypeScript

| Go | TypeScript | Catatan |
| --- | --- | --- |
| `string`, `int`, `int64`, `bool` | `string`, `number`, `boolean` | |
| `*string`, `*int`, `*time.Time` | `string \| null`, `number \| null` | pointer = nullable |
| tag `json:"-"` | — | tidak pernah muncul di JSON |
| `omitempty` | `?:` opsional | kunci hilang, bukan `null` |
| `time.Time` | `string` | RFC3339 dengan offset |
| struct yang di-*embed* | bidang-bidangnya diratakan | JSON tidak bersarang |

Struct Go yang menyematkan struct lain (`ProductView` menyematkan `Product`)
menghasilkan JSON **datar**, bukan `{"Product": {...}}`. Ini sering menjadi
sumber kesalahpahaman saat membaca definisi Go-nya.

### Peta modul ↔ berkas tipe

Setiap modul backend berpasangan dengan satu berkas tipe di web. Saat mengubah
DTO, ubah keduanya.

| Modul Go | Berkas tipe web |
| --- | --- |
| `internal/user/user.go` | `web/src/types/user.ts` |
| `internal/community/community.go` | `web/src/types/community.ts` |
| `internal/gamification/gamification.go`, `leaderboard.go` | `web/src/types/gamification.ts` |
| `internal/mission/mission.go` | `web/src/types/mission.ts` |
| `internal/fundraising/fundraising.go` | `web/src/types/fundraising.ts` |
| `internal/thread/thread.go` | `web/src/types/thread.ts` |
| `internal/shop/shop.go` | `web/src/types/shop.ts` |
| `internal/order/order.go` | `web/src/types/order.ts` |
| `internal/payment_method/payment_method.go` | `web/src/types/PaymentMethod.ts` |
| `internal/event/event.go` | `web/src/types/event.ts` |
| `internal/ticket/ticket.go` | `web/src/types/ticket.ts` |
| `internal/user_ticket/user_ticket.go` | `web/src/types/user_ticket.ts` |
| `internal/presence/presence.go` | `web/src/types/presence.ts` |
| `internal/ranger/ranger.go` | `web/src/types/ranger.ts` |
| `internal/agenda/agenda.go` | `web/src/types/agenda.ts` |
| `internal/divisi/divisi.go` | `web/src/types/divisi.ts` |
| `internal/feedback/feedback.go` | `web/src/types/feedback.ts` |
| `internal/poll/poll.go` | `web/src/types/poll.ts` |
| `internal/region/region.go` | `web/src/types/Region.ts` |
| `internal/comment/comment.go` | `web/src/types/comment.ts` |
| `internal/like/like.go` | `web/src/types/like.ts` |

Kesamaan nama tidak dijamin. `payment_method` menjadi `PaymentMethod.ts`
(huruf besar), `region` menjadi `Region.ts`, sedangkan `user_ticket` tetap
`user_ticket.ts`. `internal/metrics/` (ditambahkan pada pekerjaan lintas
modul) berpasangan dengan `web/src/types/metrics.ts`.

Beberapa berkas tipe tidak punya pasangan langsung dan murni milik UI:
`cards.ts`, `chat.ts`, `wsMessage.ts`, `rengerPresence.ts`.

---

## Modul bersama

### Auth dan akun — `internal/user/`

Model `User` menandai `Password` dan `GoogleID` sebagai `json:"-"`, dan
`AccountResponse` (`internal/user/user.go:59`) tidak memuat keduanya sama
sekali. **Tidak ada respons mana pun yang mengembalikan kata sandi atau
`google_id`.**

| Endpoint | Sukses | Body |
| --- | --- | --- |
| `POST /api/register` | 200 | `AccountResponse` langsung, tanpa amplop |
| `POST /api/login` | 200 | `{"user": AccountResponse, "access_token": string}` |
| `GET /api/auth/google/login` | 200 | `{"authUrl": string, "state": string}` — query `redirectTo` (default `/events`) |
| `GET /api/auth/google/callback` | 200 | `{"user":…, "access_token":…, "redirectTo":…}` — query `state`, `code` |
| `GET /user_api/me` | 200 | `{"user": AccountResponse}` |
| `PUT /user_api/auth` | 200 | `{"user": AccountResponse}` |
| `GET /admin_api/users/:id` | 200 | `{"user": AccountResponse}` — **tanpa** `access_token` |
| `PUT /admin_api/users/:id` | 200 | `{"user": AccountResponse}` |
| `GET /admin_api/users` | 200 | `{"users": […], "page", "per_page", "has_more"}` — *baru di pekerjaan lintas modul* |

`AccountResponse` memuat: `id`, `name`, `username`, `gender`, `birth_date`,
`age`, `phone`, `email`, `instagram`, `address`, `role`, `activity`, `source`,
`image_url`, `province_code`, `district_code`, `sub_district_code`, `province`,
`district`, `sub_district`, `created_at`, `updated_at`.

Body `POST /api/register`, `PUT /user_api/auth`, dan `PUT /admin_api/users/:id`
memakai `CreateUser`: `name`, `gender`, `age`, `birth_date`, `phone`, `email`,
`instagram`, `username`, `address`, `password`, `activity`, `source`,
`province_code`, `district_code`, `sub_district_code`. `password` **hanya**
dikirim, tidak pernah diterima kembali. Untuk `PUT`, kata sandi kosong berarti
tidak diubah.

`POST /api/login` memakai `{"email", "password"}`.

**Kekhasan modul lama yang perlu diketahui:**

- `Register`, `Login`, `UpdateAuth`, `UpdateByAdmin`, dan `Show` memetakan
  kesalahan service ke **500**, bukan 400/404. Kredensial salah menjawab
  `500 {"error":"Wrong email or password"}`.
- `Register` gagal *bind* → 400 `Invalid JSON`.
- Rate limit login: **10 per menit per IP**.

### OTP — `internal/otp/`

Body kedua endpoint: `{"email": string, "code": string}`.

| Endpoint | Sukses | Catatan |
| --- | --- | --- |
| `POST /api/auth/otp/request` | 200 `{"success":true,"message":"Success request OTP. Please check email"}` | 5/jam per email; jeda kirim ulang 60 detik |
| `POST /api/auth/otp/verify` | 200 `{"user": AccountResponse, "access_token": string}` | 10/menit per email; kode berlaku 15 menit, maksimum 5 percobaan |

Pemetaan galat di sini berbeda dari modul lain: `ErrRateLimited` → 429;
`ErrEmailInvalid`, `ErrOTPNotFound`, `ErrOTPUsed`, `ErrOTPExpired`,
`ErrOTPMismatch`, `ErrOTPAttempts` → **400**; selain itu 500.

### Pesanan tiket event — `internal/order/`

Modul ini menangani **tiket event**, bukan merchandise. Pesanan merchandise ada
di modul `shop` dan memakai path `/shop/orders`. Keduanya tidak pernah
bercampur.

Respons **tanpa amplop**: handler menulis objek atau larik apa adanya.

| Endpoint | Sukses | Body |
| --- | --- | --- |
| `GET /user_api/orders` | 200 | `[]Order` |
| `POST /user_api/orders` | 200 | `Order` |
| `GET /user_api/orders/:public_id` | 200 | `Order` |
| `GET /admin_api/orders` | 200 | `[]Order` |
| `PUT /admin_api/orders/:id/verify` | 200 | `Order` |

`POST /user_api/orders` memakai: `event_id`, `payment_method_id`,
`user_tickets[]` (`user_name`, `user_email`, `user_gender`, `ticket_id`,
`event_id`), `donation`, `admin_fee`. **Identitas pembeli diambil dari token,
bukan dari body.**

`PUT /admin_api/orders/:id/verify` memakai `{"status": string}`.

Status order: `pending`, `paid`, `expired`, `cancelled`, `failed`, `refunded`.
Perpindahan yang sah: `pending` → `paid`/`failed`/`expired`/`cancelled`;
`paid` → `refunded`; sisanya terminal.

Galat: status tidak dikenal → 400 `status "X" tidak dikenal`; perpindahan
terlarang → 400 `status tidak bisa berpindah dari A ke B`; kalah balapan →
400 `status order sudah berubah, muat ulang datanya`; keranjang tiket kosong →
400 `order harus berisi minimal satu tiket`.

### Metode pembayaran — `internal/payment_method/`

Respons **tanpa amplop**. `PaymentMethod` = `id`, `type` (`BANK`, `E-WALLET`,
`QRIS`), `code`, `name`, `image_url`, `account_name`, `account_number`.

| Endpoint | Sukses |
| --- | --- |
| `GET /user_api/payment_methods` | 200 `[]PaymentMethod` |
| `POST /admin_api/payment_methods` | 200 `PaymentMethod` |
| `PUT /admin_api/payment_methods/:id` | 200 `PaymentMethod` |
| `DELETE /admin_api/payment_methods/:id` | 200 `{"message":"Success delete ticket"}` |

Body create/update: `{"name","type","code","account_name","account_number"}`,
semuanya wajib.

Dua kekhasan modul lama: kegagalan *bind* menjawab **500** `Invalid JSON`
(bukan 400), dan pesan hapus berbunyi `Success delete ticket` — sisa salin
tempel yang tidak memengaruhi perilaku.

---

## Fase 1 — profil komunitas, XP, misi

Modul: `internal/community/`, `internal/gamification/`, `internal/mission/`.
Migrasi: `0007_community_profiles`, `0008_gamification`, `0009_missions`.

Tiga modul yang saling terkait tetapi terpisah. `community` menyimpan profil
publik; `gamification` menyimpan buku besar XP dan kurva level; `mission`
menyimpan misi dan klaimnya.

### Profil komunitas — `internal/community/`

Identitas publik dipisahkan dari akun. `Profile` menyimpan `user_id` dan
`public_id`; **hanya `public_id` yang pernah keluar**. `alias` adalah nama
tampilan; bila kosong, pemanggil menggantinya dengan `"Anggota"`.

```ts
type PublicProfile = {
  public_id: string; alias: string;
  bio: string | null; avatar_url: string | null;
  show_badges: boolean;
};

type OwnProfile = {
  public_id: string; alias: string | null;
  bio: string | null; avatar_url: string | null;
  profile_visibility: "public" | "members" | "private";
  leaderboard_opt_out: boolean; show_badges: boolean;
};
```

| Endpoint | Amplop |
| --- | --- |
| `GET /api/community/profiles/:public_id` | `{"profile": PublicProfile}` |
| `GET /user_api/community/profile` | `{"profile": OwnProfile}` |
| `PUT /user_api/community/profile` | `{"profile": OwnProfile}` |

`GET /user_api/community/profile` membuat baris profil bila belum ada
(`Ensure`), sehingga selalu berhasil untuk anggota yang sudah masuk.

Body `PUT` — semua bidang opsional; yang absen tidak diubah:
`alias`, `bio`, `avatar_url`, `profile_visibility`, `leaderboard_opt_out`,
`show_badges`.

- `alias` disanitasi lebih dulu (spasi berlebih dirapatkan), lalu harus 3–40
  karakter. Karakter yang diizinkan: huruf, angka, spasi, `.`, `_`, `-`.
- `alias` kosong **menghapus** alias (disimpan `NULL`), bukan ditolak.
- Alias unik tanpa membedakan huruf besar-kecil (indeks parsial pada
  `lower(alias)`). Bentrok → 400 `alias sudah dipakai akun lain`.
- `profile_visibility` harus salah satu dari tiga nilai; selain itu 400.

Profil yang tidak ada, `is_blocked = true`, atau `profile_visibility =
"private"` semuanya menjawab **404 yang sama** — keberadaannya tidak
dibocorkan.

**Profil komunitas tidak memberi XP.** Reward "profil lengkap" diberikan modul
`user` saat menyimpan data akun, bukan saat menyimpan profil komunitas.

### XP — `internal/gamification/`

```ts
type LedgerEntry = {
  id: string; delta: number; source_type: string;
  ref_type: string | null; ref_id: string | null;
  note: string | null; created_at: string;
};

type XPSummary = {
  total_xp: number; level: number; level_name: string; badge: string;
  progress_percent: number; remaining_xp: number;
  next_level: { level: number; name: string; min_xp: number } | null;
};

type XPRule = {
  id: string; source_type: string; xp: number;
  description: string | null; monthly_cap: number | null;
};

type LevelRule = {
  id: string; level: number; name: string; min_xp: number;
  badge_label: string | null; badge_icon_url: string | null;
};
```

`user_id`, `dedup_key`, dan `actor_user_id` pada `LedgerEntry` bertag
`json:"-"` — **tidak pernah keluar**. Riwayat XP tidak dapat dipakai untuk
mengaitkan seorang anggota dengan akunnya.

| Endpoint | Amplop | Catatan |
| --- | --- | --- |
| `GET /user_api/xp` | `{"summary": XPSummary}` | — |
| `GET /user_api/xp/history` | `{"entries": [LedgerEntry], page, per_page, has_more}` | urut `created_at DESC` |
| `POST /admin_api/xp/adjustments` | **201** `{"entry": LedgerEntry}` | izin `mission:manage`; 30/menit |
| `GET /admin_api/xp/rules` | `{"rules": [XPRule]}` | urut `source_type ASC` |
| `PUT /admin_api/xp/rules/:source_type` | `{"rule": XPRule}` | izin `mission:manage` |
| `GET /admin_api/level_rules` | `{"rules": [LevelRule]}` | urut `min_xp ASC` |
| `PUT /admin_api/level_rules` | `{"rules": [LevelRule]}` | izin `mission:manage` |

`next_level` bernilai `null` di level tertinggi; di sana
`progress_percent = 100` dan `remaining_xp = 0`.

**Dua endpoint baca tidak memeriksa izin di handler** — `GET
/admin_api/xp/rules` dan `GET /admin_api/level_rules` hanya dijaga middleware
`AuthPJ` di tingkat route. Keduanya tetap tertutup bagi non-staf, tetapi tidak
memakai pemeriksaan izin yang sama dengan pasangan tulisnya.

`POST /admin_api/xp/adjustments` memakai `{"user_id","delta","reason"}`,
ketiganya `binding:"required"`. Karena `delta` bertipe `int`, **`delta: 0`
gagal di tahap bind** dan menjawab `400 {"error":"Invalid JSON"}` — bukan pesan
`delta XP tidak boleh nol` dari service. Pesan service itu hanya muncul untuk
nilai yang lolos bind tetapi ditolak di sana.

Galat: 404 `NotFound: akun tidak ditemukan`; 429 bila melewati 30/menit.

`PUT /admin_api/xp/rules/:source_type` memakai `{"xp","description",
"monthly_cap"}`. `xp` wajib dan harus > 0. **`monthly_cap` ditulis apa
adanya** — `null` atau absen berarti "tanpa batas", bukan "jangan diubah".
`source_type: "mission"` ditolak: 400 `reward misi diatur per misi, bukan di
aturan XP`.

`PUT /admin_api/level_rules` memakai `{"rules": [{level, name, min_xp,
badge_label, badge_icon_url}]}` dan **harus memuat seluruh kurva**. Kurva
divalidasi: tepat satu level ber-`min_xp: 0`; `level` dan `min_xp` masing-masing
unik; nama wajib. Jumlah level tidak boleh berubah, dan setiap level yang
dikirim harus sudah ada di kurva saat ini.

`source_type` XP: `profile_complete`, `checkin`, `mission`, `adjustment`,
`donation`, `donation_reversal`, `shop_order`, `shop_order_reversal`.
Tidak ada `CHECK` di SQL untuk kolom ini — nilainya hanya dijaga kode Go.

**Riwayat audit** (ditambahkan pada pekerjaan lintas modul):
`xp.adjust`, `xp_rule.update`, `level_rule.update` — dibaca lewat
`GET /admin_api/xp/audit_logs`.

### Papan peringkat

`GET /api/leaderboard` mengembalikan struct `LeaderboardPage` **tanpa
amplop** — kuncinya ada di tingkat atas.

```json
{
  "period": "weekly",
  "page": 1, "per_page": 20, "has_more": false,
  "entries": [{
    "rank": 1, "public_id": "…", "alias": "…", "avatar_url": null,
    "level": 3, "level_name": "…", "badge": "…", "xp": 1250
  }]
}
```

`period` menerima `weekly` (bawaan bila kosong), `monthly`, `all_time`;
selain itu 400. Profil yang memilih keluar (`leaderboard_opt_out`) atau
dibatasi tidak muncul. `badge` menjadi string kosong bila pemiliknya
menyetel `show_badges = false`; `alias` jatuh ke `"Anggota"`.

### Misi — `internal/mission/`

```ts
type MissionView = Mission & {
  current_period_key: string;
  can_claim_now: boolean;
  required_proof: boolean;
};

type ClaimView = MissionClaim & {
  mission_title: string; mission_type: string;
};

type AdminClaimView = ClaimView & { claimant: PublicProfile | null };
```

`MissionView`, `ClaimView`, dan `AdminClaimView` menyematkan struct dasarnya,
sehingga JSON-nya **datar**. `created_by`, `user_id`, `reviewed_by`, dan
`deleted_at` bertag `json:"-"`.

Nilai yang sah:

| Bidang | Nilai |
| --- | --- |
| `mission.type` | `daily`, `weekly`, `special` |
| `verification_mode` | `auto`, `self_claim`, `proof_approval` |
| klaim `status` | `pending`, `approved`, `rejected`, `cancelled` |

`period_key` dihitung di zona Asia/Jakarta: `daily:YYYY-MM-DD`,
`weekly:YYYY-Www`, dan harfiah `special` untuk misi bertipe `special`.

| Endpoint | Amplop |
| --- | --- |
| `GET /api/missions` | `{"missions": [MissionView]}` — tanpa pagination |
| `GET /api/missions/:id` | `{"mission": MissionView}` |
| `POST /user_api/missions/:id/claims` | **201** `{"claim": MissionClaim}` |
| `GET /user_api/missions/claims` | `{"claims": [ClaimView], page, per_page, has_more}` |
| `GET /admin_api/missions` | `{"missions": [Mission]}` |
| `POST /admin_api/missions` | **201** `{"mission": Mission}` |
| `PUT /admin_api/missions/:id` | `{"mission": Mission}` |
| `DELETE /admin_api/missions/:id` | `{"message":"Misi dihapus"}` |
| `GET /admin_api/missions/claims` | `{"claims": [AdminClaimView], …}` |
| `PUT /admin_api/missions/claims/:id/approve` | `{"claim": MissionClaim}` |
| `PUT /admin_api/missions/claims/:id/reject` | `{"claim": MissionClaim}` |

Semua endpoint `admin_api/missions*` memerlukan izin `mission:manage`.

**Endpoint pengurus mengembalikan model mentah.** `Mission`, bukan
`MissionView` — jadi `current_period_key`, `can_claim_now`, dan
`required_proof` **tidak ada** di sana. Ketiganya hanya muncul di endpoint
anggota. Demikian pula `approve`/`reject` mengembalikan `MissionClaim` mentah,
bukan `ClaimView`.

`GET /api/missions` tidak berhalaman dan hanya memuat misi terbit yang belum
berakhir. `GET /api/missions/:id` menjawab 404 untuk misi yang tidak ada,
terhapus, atau belum terbit.

**Klaim.** Body `POST /user_api/missions/:id/claims` **opsional** — hanya
di-*bind* bila `ContentLength > 0`. Bentuknya `{"proof_url","proof_note"}`.

- Rate limit **20 per menit**.
- Jendela klaim harus terbuka: misi terbit, sudah `starts_at`, dan belum
  `ends_at` (batas akhir eksklusif). Di luar itu 400
  `InvalidRequest: jendela klaim misi sedang tertutup`.
- `verification_mode: "proof_approval"` mewajibkan minimal salah satu dari
  `proof_url`/`proof_note` terisi (spasi saja dianggap kosong).
- Batas klaim per periode dihitung dari `claim_limit`; terlampaui → 400
  `InvalidRequest: batas klaim untuk periode ini sudah tercapai`.
- `verification_mode: "auto"` langsung `approved` dan `reward_xp` disalin ke
  klaim. Mode lain menghasilkan `pending`.

Body `POST`/`PUT` misi (`CreateMission`/`UpdateMission`): `code`, `title`,
`description`, `type`, `verification_mode`, `reward_xp`, `claim_limit`,
`starts_at`, `ends_at`, `is_published`, `sort_order`. `starts_at`/`ends_at`
adalah **RFC3339 dengan offset eksplisit**; tanpa offset → 400
`format waktu tidak valid, gunakan RFC3339 dengan offset zona waktu: <nilai>`.
`ends_at` harus setelah `starts_at`. Pada `PUT`, seluruh bidang opsional dan
setiap perubahan menaikkan `version`.

`code` unik di antara misi aktif. `reward_xp` tidak boleh negatif;
`claim_limit` minimal 1.

**Keputusan klaim.** `approve` dan `reject` berbagi satu handler; body
opsional `{"reason"}`. `reason` **wajib untuk `reject`**, tidak untuk
`approve`.

- Klaim yang sudah diputuskan → 400 `InvalidRequest: klaim sudah diputuskan`.
- **Menyetujui klaim yang sudah `approved` bersifat idempoten**: ia
  mengembalikan 200 dan mencoba lagi pemberian XP, yang aman karena memakai
  kunci dedup. Tidak menggandakan reward.
- `GET /admin_api/missions/claims` menerima filter `status`; nilai di luar
  empat status yang sah → 400. `claimant` dapat bernilai `null` bila
  penyelesaiannya gagal (best-effort).

---

## Fase 2 — donasi dan penggalangan dana

Modul: `internal/fundraising/`. Migrasi: `sql/migrations/0010_fundraising.up.sql`.

Respons memakai amplop bernama kunci: `{"campaign": …}`, `{"donation": …}`,
`{"campaigns": […]}`, dst. Pengecualiannya adalah dua endpoint anggota yang
mengembalikan `DonationResult` di tingkat atas — lihat catatannya.

### Bentuk data

```ts
type Campaign = {
  id: string; slug: string; title: string;
  summary: string | null; story: string | null; cover_image_url: string | null;
  fund_type: "operasional" | "dakwah" | "sosial" | "pendidikan" | "lainnya";
  recipient: string; target_amount: number;
  status: "draft" | "published" | "closed";
  starts_at: string | null; ends_at: string | null;
  published_at: string | null; closed_at: string | null;
  created_at: string; updated_at: string;
};

type Donation = {
  id: string; public_id: string; campaign_id: string;
  amount: number;
  status: "pending" | "confirmed" | "rejected" | "cancelled" | "refunded";
  is_anonymous: boolean; show_amount: boolean;
  message: string | null;
  message_status: "none" | "pending" | "approved" | "hidden";
  payment_method_id: string | null;
  paid_amount: number | null; payment_reference: string | null;
  proof_url: string | null;
  confirmed_at: string | null; decision_reason: string | null;
  rewarded_xp: number;
  created_at: string; updated_at: string;
};

type CampaignUpdate = {
  id: string; campaign_id: string; title: string; body: string;
  kind: "update" | "usage" | "fee";
  amount: number | null; proof_url: string | null;
  is_published: boolean; created_at: string; updated_at: string;
};
```

`CampaignView` = `Campaign` + `raised_amount`, `donor_count`,
`progress_percent`, `is_open`.

`DonationView` = `Donation` + `campaign_slug`, `campaign_title`.
`AdminDonationView` = `Donation` + `donor` (profil publik komunitas) +
`campaign_slug`, `campaign_title`.

`PublicDonation` (bentuk publik) = `public_id`, `donor_name`, `donor_hidden`,
`avatar_url`, `amount`, `message`, `created_at`. Anonimitas diterapkan **hanya
di sini**: identitas tersembunyi juga menyembunyikan `amount` dan
`avatar_url`. Nama penggantinya `"Hamba Allah"`. Pemilik selalu melihat
`DonationView` utuh — tanpa penyamaran.

`ReportTotals` = `confirmed_amount`, `confirmed_count`, `pending_amount`,
`pending_count`, `rejected_amount`, `rejected_count`, `refunded_amount`,
`refunded_count`, `fee_amount`, `usage_amount`, `net_amount`.
`net_amount = confirmed − refunded − fee − usage`.

### Endpoint publik

| Endpoint | Amplop | Pagination |
| --- | --- | --- |
| `GET /api/campaigns` | `{"campaigns": [CampaignView]}` | tanpa |
| `GET /api/campaigns/:slug` | `{"campaign": CampaignView}` | — |
| `GET /api/campaigns/:slug/donors` | `{"donations": [PublicDonation], page, per_page, has_more}` | ya |
| `GET /api/campaigns/:slug/messages` | `{"messages": [PublicDonation], …}` | ya |
| `GET /api/campaigns/:slug/updates` | `{"updates": [CampaignUpdate]}` | tanpa |
| `GET /api/campaigns/:slug/report` | `{"report": {campaign, totals, updates}}` | — |

`:slug` hanya menemukan campaign berstatus `published` dan berada di dalam
`[starts_at, ends_at)`. Di luar itu → 404. Donor hanya yang `confirmed`;
pesan hanya yang `confirmed` + `message_status='approved'` + pesannya tidak
kosong.

### Endpoint anggota

| Endpoint | Sukses | Amplop |
| --- | --- | --- |
| `POST /user_api/donations` | **201** | `{"donation": DonationView, "charge": Charge}` |
| `GET /user_api/donations` | 200 | `{"donations": [DonationView], page, per_page, has_more}` |
| `GET /user_api/donations/:public_id` | 200 | `{"donation": DonationView, "charge": Charge}` |

Body `POST`: `campaign_slug`, `amount`, `is_anonymous`, `show_amount`,
`message`, `payment_method_id`, `client_token` (**wajib**).

- `client_token` adalah kunci idempotensi. Mengirim token yang sama dua kali
  mengembalikan donasi yang sama, bukan membuat yang baru.
- `amount` harus antara 1.000 dan 1.000.000.000.000.
- `is_anonymous=true` memaksa `show_amount=false`.
- Pesan kosong menghasilkan `message_status="none"`, sehingga tidak masuk
  antrean moderasi.
- Rate limit: **10 per menit** per donatur.
- Donasi milik orang lain menjawab **404**, bukan 403 — keberadaannya pun
  tidak dibocorkan.

`charge` hanya ada selama donasi masih `pending`. Bentuknya:
`{"provider","external_id","amount","instructions":[{"method_id","type","name","account_name","account_number","image_url"}],"expires_at"}`.
Provider manual mengisi `provider:"manual"`, `external_id:null`,
`expires_at:null`.

### Endpoint pengurus

Semua memerlukan izin `funds:manage`. Route memakai `AuthPJ` (admin + pj).

| Endpoint | Sukses | Amplop |
| --- | --- | --- |
| `GET /admin_api/campaigns` | 200 | `{"campaigns": [CampaignView]}` — query `status` |
| `POST /admin_api/campaigns` | **201** | `{"campaign": Campaign}` |
| `PUT /admin_api/campaigns/:id` | 200 | `{"campaign": Campaign}` |
| `DELETE /admin_api/campaigns/:id` | 200 | `{"message":"Campaign dihapus"}` |
| `PUT /admin_api/campaigns/:id/status` | 200 | `{"campaign": Campaign}` |
| `GET /admin_api/campaigns/:id/report` | 200 | `{"report": ReportView}` |
| `POST /admin_api/campaigns/:id/updates` | **201** | `{"update": CampaignUpdate}` |
| `PUT /admin_api/campaign_updates/:id` | 200 | `{"update": CampaignUpdate}` |
| `DELETE /admin_api/campaign_updates/:id` | 200 | `{"message":"Update dihapus"}` |
| `GET /admin_api/donations` | 200 | `{"donations": [AdminDonationView], page, per_page, has_more}` |
| `PUT /admin_api/donations/:id/confirm` | 200 | `{"donation": Donation}` |
| `PUT /admin_api/donations/:id/reject` | 200 | `{"donation": Donation}` |
| `PUT /admin_api/donations/:id/refund` | 200 | `{"donation": Donation}` |
| `PUT /admin_api/donations/:id/message` | 200 | `{"donation": Donation}` |
| `GET /admin_api/audit_logs` | 200 | `{"audit_logs": [AuditLog], page, per_page, has_more}` |

Query: `ListAllCampaigns` → `status`; `DonationsForReview` → `status`,
`message_status`, `campaign_id`; `ListAuditLogs` → `entity_type`, `entity_id`.

Body:

- `POST /admin_api/campaigns` — `CreateCampaign`: `slug`, `title`, `summary`,
  `story`, `cover_image_url`, `fund_type`, `recipient`, `target_amount`,
  `starts_at`, `ends_at`. `starts_at`/`ends_at` adalah string **RFC3339
  dengan offset eksplisit**.
- `PUT /admin_api/campaigns/:id` — semua bidang opsional; `null`/absen berarti
  tidak diubah.
- `PUT …/status` — `{"status", "reason"}`. Menyetel status yang sama
  mengembalikan 200 tanpa perubahan; perpindahan terlarang → 400.
- `PUT …/confirm` — `{"paid_amount","payment_reference","proof_url","reason"}`.
  `paid_amount <= 0` berarti memakai nominal donasi. Referensi wajib, dan
  nominalnya harus **sama persis** — kalau tidak, 400.
- `PUT …/reject` dan `…/refund` — `{"reason"}` (**wajib**).
- `PUT …/message` — `{"decision","reason"}`. `decision` menerima
  `approve`/`approved` atau `hide`/`hidden`, tanpa membedakan huruf besar-kecil.
- `POST …/updates` — `{"title","body","kind","amount","proof_url","is_published"}`.
  `kind` `usage` dan `fee` **wajib** menyertakan `amount`.

**Idempotensi.** `confirm` pada donasi yang sudah `confirmed` mengembalikan 200
dan menghitung ulang XP dengan aman (dideduplikasi lewat kunci unik), bukan
menggandakan reward. Mengonfirmasi donasi yang sudah `rejected`/`refunded`/
`cancelled` → 400. Hanya `confirm` yang dibatasi laju (60/menit); `reject`,
`refund`, dan moderasi pesan tidak.

**Riwayat audit** `/admin_api/audit_logs` memuat aksi dari modul ini saja:
`campaign.create/update/status/delete`, `campaign_update.*`,
`donation.confirm/reject/refund`, `message.approve/hide`. Modul `shop` dan
`thread` punya endpoint riwayatnya sendiri.

---

## Fase 3 — komunitas anonim

Modul: `internal/thread/`. Migrasi: `sql/migrations/0011_threads.up.sql`.

Identitas penulis **selalu** alias; akun tanpa alias tampil sebagai `"Anonim"`.
`user_id` tidak pernah muncul di respons mana pun.

### Bentuk data

```ts
type ThreadAuthor = {
  public_id: string; alias: string;
  avatar_url: string | null; show_badges: boolean;
};

type ThreadView = {
  public_id: string; title: string;
  body?: string;        // hanya di detail
  excerpt?: string;     // hanya di daftar
  author: ThreadAuthor;
  status: "published" | "hidden" | "deleted";
  source_type: "event_registration" | "donation" | "mission" | null;
  comment_count: number; reaction_count: number;
  reacted: boolean; is_mine: boolean;
  created_at: string;
};

type ThreadCommentView = {
  public_id: string; body: string; author: ThreadAuthor;
  status: "published" | "hidden" | "deleted";
  is_mine: boolean; created_at: string;
};

type ModerationReportView = {
  report: Report; target: ReportTargetView; author: ThreadAuthor;
};
```

`Report` = `id`, `public_id`, `target_type` (`thread` | `thread_comment`),
`reason`, `note`, `status` (`open` | `actioned` | `dismissed`), `handled_by`,
`handled_at`, `decision_reason`, `created_at`. `target_id` **tidak**
diungkapkan.

`SharePrefs` = `share_event_registration`, `share_donation`, `share_mission`
(semuanya boolean). Baris yang belum ada dianggap **semuanya `true`**.

### Endpoint publik

| Endpoint | Amplop | Query |
| --- | --- | --- |
| `GET /api/threads` | `{"threads": [ThreadView], page, per_page, has_more}` | `sort` (`terbaru` bawaan, `populer`) |
| `GET /api/threads/:public_id` | `{"thread": ThreadView}` | — |
| `GET /api/threads/:public_id/comments` | `{"comments": [ThreadCommentView], …}` | `page`, `per_page` |

Thread yang disembunyikan atau dihapus menjawab **404**, sama seperti thread
yang tidak pernah ada. Penulis yang diblokir juga menyembunyikan threadnya.

### Endpoint anggota

| Endpoint | Sukses | Amplop | Rate limit |
| --- | --- | --- | --- |
| `POST /user_api/threads` | **201** | `{"thread": ThreadView}` | 5/jam |
| `DELETE /user_api/threads/:public_id` | 200 | `{"message":"Thread dihapus"}` | — |
| `POST /user_api/threads/:public_id/comments` | **201** | `{"comment": ThreadCommentView}` | 20/menit |
| `DELETE …/comments/:comment_public_id` | 200 | `{"message":"Komentar dihapus"}` | — |
| `PUT /user_api/threads/:public_id/reaction` | 200 | `{"message":"Reaksi tersimpan"}` | 60/menit |
| `DELETE /user_api/threads/:public_id/reaction` | 200 | `{"message":"Reaksi dibatalkan"}` | — |
| `POST /user_api/threads/reports` | 200 | `{"message":"Laporan diterima"}` | 10/jam |
| `GET /user_api/community/share_prefs` | 200 | `{"share_prefs": SharePrefs}` | — |
| `PUT /user_api/community/share_prefs` | 200 | `{"share_prefs": SharePrefs}` | — |

Body: `POST /threads` → `{"title","body"}`; komentar → `{"body"}`;
laporan → `{"target_type","target_id","reason","note"}` dengan `reason` salah
satu dari `spam`, `sara`, `pornografi`, `penipuan`, `perundungan`, `lainnya`;
share prefs → tiga boolean opsional (yang absen tidak diubah).

Semua penulisan melewati `guardWrite`: 401 bila belum masuk, 429 bila melewati
batas, lalu 403 bila akunnya sedang dibatasi.

Melaporkan target yang sama dua kali mengembalikan **200** (idempoten), bukan
galat. `DELETE` pada reaksi/thread/komentar yang tidak ada → 404.

### Endpoint moderasi

Memerlukan izin `moderation:moderate`. Route memakai `AuthRanger`, yang
meloloskan ranger, pj, dan admin — ketiganya memegang izin itu.

| Endpoint | Amplop | Query |
| --- | --- | --- |
| `GET /admin_api/moderation/reports` | `{"reports": [ModerationReportView], …}` | `status` |
| `PUT …/reports/:id/action` | `{"report": Report}` | body `{"reason","hide_target"}` |
| `PUT …/reports/:id/dismiss` | `{"report": Report}` | body `{"reason"}` |
| `GET /admin_api/moderation/threads` | `{"threads": [ThreadView], …}` | `status` |
| `PUT …/threads/:id/:action` | `{"thread": Thread}` | `:action` = `hide`\|`restore`\|`delete` |
| `PUT …/thread_comments/:id/:action` | `{"comment": ThreadComment}` | idem |
| `GET …/accounts/restricted` | `{"accounts": [PublicProfile], …}` | — |
| `PUT …/accounts/:public_id/restrict` | `{"account": PublicProfile}` | body `{"blocked","reason"}` |
| `GET /admin_api/moderation/audit_logs` | `{"audit_logs": [AuditLog], …}` | — |

`:action` harus persis `hide`, `restore`, atau `delete` — huruf kecil, tanpa
spasi. `hide` dan `delete` mewajibkan `reason` sepanjang 3–500 karakter;
`restore` tidak.

**Idempoten.** Tindakan yang tidak mengubah status mengembalikan baris apa
adanya dengan 200, bukan galat.

`reason` pada restrict **wajib** saat `blocked=true`.

**Riwayat audit** di sini memuat `thread.hide/restore/delete`,
`thread_comment.hide/restore/delete`, `report.action`, `report.dismiss`,
`account.restrict/unrestrict`, `share.revoke`. Komentar dan reaksi anggota
biasa **sengaja tidak** diaudit — jumlahnya besar dan bukan tindakan privilese.

---

## Fase 4 — toko merchandise

Modul: `internal/shop/`. Migrasi: `sql/migrations/0012_shop.up.sql`.

### Bentuk data

```ts
type ProductView = Product & {
  images: ProductImage[];
  variants: VariantView[];
  min_price: number; max_price: number; available: number;
};

type VariantView = ProductVariant & {
  label: string;      // "L" / "Merah / L" / "" bila tanpa pembeda
  available: number;  // stok dikurangi reservasi aktif
};

type OrderView = ShopOrder & {
  charge?: Charge;    // hanya saat payment_status = "pending"
};
```

`ProductView`, `VariantView`, dan `OrderView` adalah struct Go yang
di-*embed*, sehingga JSON-nya **datar** — bukan `{"Product": {...}}`.

Nilai yang sah:

| Bidang | Nilai |
| --- | --- |
| `product.status` | `draft`, `published`, `archived` |
| `variant.status` | `active`, `inactive` |
| `payment_status` | `pending`, `paid`, `rejected`, `refunded` |
| `fulfillment_status` | `unfulfilled`, `ready_for_pickup`, `shipped`, `completed`, `cancelled` |
| `fulfillment_method` | `pickup`, `shipping` |
| `stock_movements.reason` | `restock`, `adjustment`, `sale`, `return` |
| query `sort` katalog | `terbaru`, `termurah`, `termahal` |

Perpindahan status yang sah (`internal/shop/state.go`):

- Produk: `draft` → `published`|`archived`; `published` → `draft`|`archived`;
  `archived` → `draft`. Status ke dirinya sendiri ditolak.
- Pembayaran: `pending` → `paid`|`rejected`; `paid` → `refunded`; sisanya
  terminal.
- Pemenuhan: bergantung `fulfillment_method`. `pickup`:
  `unfulfilled` → `ready_for_pickup` → `completed`. `shipping`:
  `unfulfilled` → `shipped` → `completed`. Pembatalan hanya dari
  `unfulfilled` atau `ready_for_pickup`.

Batas: `qty` per baris maksimum **10** (dijaga di Go dan di `CHECK`). **Tidak
ada batas jumlah varian berbeda per pesanan** — hanya minimal satu baris.
Varian unik per kombinasi `(product_id, size, color)`. `slug` cocok dengan
`^[a-z0-9]+(-[a-z0-9]+)*$`, panjang maksimum 160. Nama produk 3–140 karakter.
Harga dan stok tidak boleh negatif.

Reservasi stok kedaluwarsa setelah **24 jam**, dihitung **saat data diakses**
— tidak ada pekerja latar yang menyapunya.

### Endpoint publik

| Endpoint | Amplop | Query |
| --- | --- | --- |
| `GET /api/shop/products` | `{"products": [ProductView], page, per_page, has_more}` | `sort`, `page`, `per_page` |
| `GET /api/shop/products/:slug` | `{"product": ProductView}` | — |

`sort` yang tidak dikenal diam-diam jatuh ke `terbaru`. Produk yang belum
terbit, sudah dihapus, atau slug tidak dikenal → 404.

### Endpoint anggota

| Endpoint | Sukses | Amplop |
| --- | --- | --- |
| `POST /user_api/shop/orders` | **201** | `{"order": OrderView}` |
| `GET /user_api/shop/orders` | 200 | `{"orders": [OrderView], page, per_page, has_more}` |
| `GET /user_api/shop/orders/:public_id` | 200 | `{"order": OrderView}` |
| `PUT /user_api/shop/orders/:public_id/proof` | 200 | `{"order": OrderView}` |
| `PUT /user_api/shop/orders/:public_id/cancel` | 200 | `{"order": OrderView}` |

Body checkout:

```json
{
  "items": [{ "variant_id": "…", "qty": 2 }],
  "fulfillment_method": "pickup" | "shipping",
  "recipient_name": "…",
  "recipient_phone": "…",
  "recipient_address": "…",
  "payment_method_id": "…"
}
```

**Klien tidak pernah mengirim harga.** Server membaca nama, label, dan harga
dari varian di dalam transaksi, lalu menyimpannya sebagai snapshot di baris
pesanan. Pesanan lama tetap utuh meski produknya kemudian berubah atau
dihapus.

Untuk `fulfillment_method: "shipping"`, `recipient_name`, `recipient_phone`,
dan `recipient_address` **wajib**. Untuk `pickup`, ketiganya diabaikan.

Rate limit checkout: **10 per jam**. `proof`: **10 per jam**.

`PUT …/proof` memakai `{"proof_url","payment_reference","payment_method_id"}`;
`proof_url` wajib. Hanya berlaku selama pesanan masih `pending`.

`PUT …/cancel` mengembalikan stok. Pesanan yang **sudah** `cancelled`
menjawab 200 tanpa perubahan — idempoten. Pesanan yang sudah dibayar tidak
dapat dibatalkan sendiri oleh pembeli.

Pesanan milik orang lain menjawab **404**, bukan 403.

`charge` hanya muncul pada `payment_status = "pending"`; pada daftar ia tidak
disertakan sama sekali.

### Endpoint pengurus

Semua memerlukan izin `product:manage`; route memakai `AuthPJ`.

| Endpoint | Sukses | Amplop |
| --- | --- | --- |
| `GET /admin_api/shop/products` | 200 | `{"products": [ProductView], …}` — query `status` |
| `POST /admin_api/shop/products` | **201** | `{"product": ProductView}` |
| `PUT /admin_api/shop/products/:id` | 200 | `{"product": ProductView}` |
| `PUT /admin_api/shop/products/:id/status` | 200 | `{"product": ProductView}` |
| `DELETE /admin_api/shop/products/:id` | 200 | `{"message":"produk dihapus"}` |
| `POST /admin_api/shop/products/:id/variants` | **201** | `{"variant": ProductVariant}` |
| `PUT /admin_api/shop/variants/:id` | 200 | `{"variant": ProductVariant}` |
| `DELETE /admin_api/shop/variants/:id` | 200 | `{"message":"varian dihapus"}` |
| `POST /admin_api/shop/variants/:id/stock` | 200 | `{"variant": ProductVariant}` |
| `GET /admin_api/shop/orders` | 200 | `{"orders": [OrderView], …}` |
| `PUT /admin_api/shop/orders/:id/confirm` | 200 | `{"order": OrderView}` |
| `PUT /admin_api/shop/orders/:id/reject` | 200 | `{"order": OrderView}` |
| `PUT /admin_api/shop/orders/:id/refund` | 200 | `{"order": OrderView}` |
| `PUT /admin_api/shop/orders/:id/fulfill` | 200 | `{"order": OrderView}` |
| `PUT /admin_api/shop/orders/:id/shipping` | 200 | `{"order": OrderView}` |
| `GET /admin_api/shop/audit_logs` | 200 | `{"audit_logs": [AuditLog], …}` |

Query: `AdminProducts` → `status`; `AdminOrders` → `payment_status`,
`fulfillment_status`. Nilai yang tidak dikenal → 400, bukan diabaikan
(different dari `sort` katalog).

Body:

- Produk — `{"name","slug","description","cover_image_url","image_urls"}`.
  Saat `PUT`, seluruh bidang opsional; `image_urls` **menggantikan** seluruh
  galeri.
- Status produk — `{"status","reason"}`.
- Varian (create dan update memakai bentuk sama) — `{"sku","size","color","price","stock","status","position"}`.
  Minimal salah satu dari `size`/`color` harus terisi.
  **`stock` diabaikan pada `PUT`** — stok hanya berubah lewat endpoint
  penyesuaian, supaya buku besar stok tetap dapat dipercaya. Pada `POST`,
  `stock` menjadi stok awal.
- Penyesuaian stok — `{"delta","reason","note"}`. `delta` bertanda, tidak
  boleh nol; `reason` wajib.
- `confirm` — `{"paid_amount","payment_reference","proof_url","reason"}`.
  `paid_amount <= 0` berarti memakai total pesanan. Nominal harus sama persis
  dengan tagihan, dan referensi wajib.
- `reject` dan `refund` — `{"reason"}` (**wajib**).
- `fulfill` — `{"status","reason"}`. Pesanan harus sudah `paid`. Menuju
  `cancelled` mewajibkan `reason`.
- `shipping` — `{"shipping_cost","tracking_number"}`. `shipping_cost` hanya
  dapat diubah selama pembayaran masih `pending`; nomor resi kapan saja. Bila
  tidak ada yang berubah, mengembalikan 200 tanpa perubahan.

Rate limit keputusan pengurus (`confirm`/`reject`/`refund`/`fulfill`):
**60 per menit**.

**Idempotensi — inilah yang menjamin empat kriteria selesai Fase 4:**

- **Tidak ada overselling.** Stok dikunci dengan *advisory lock* per varian
  di dalam satu transaksi, ditambah `UNIQUE (order_id, variant_id)` dan
  `INSERT … ON CONFLICT DO NOTHING`. Checkout bersamaan tidak dapat membuat
  stok negatif; yang kalah mendapat 400 `stok tidak lagi tersedia`.
- **Konfirmasi berulang tidak memproses dua kali.** Perpindahan status
  memakai `WHERE payment_status = 'pending'`, sehingga percobaan kedua
  mengenai nol baris dan diperlakukan sebagai sukses idempoten. XP hadiah
  memakai kunci dedup.
- **Pembatalan mengembalikan stok tepat sekali.** Pengembalian ditandai baris
  `stock_movements` bertipe `return` dan `stock_reservations.released_at IS NULL`,
  sehingga pengulangan tidak menambah stok dua kali.
- **Pesanan toko tidak menyentuh tiket event.** Modul `shop` tidak mengimpor
  `internal/order` maupun `internal/user_ticket`.

**Riwayat audit** di sini memuat `product.create/update/status/delete`,
`variant.create/update/delete`, `stock.adjust`, dan
`shop_order.confirm/reject/refund/cancel/fulfill/shipping`.

---

## Lintas modul

Modul: `internal/metrics/`. Tanpa migrasi, tanpa tabel baru, dan tanpa
penulisan apa pun — seluruhnya pembacaan agregat atas tabel yang sudah ada.

### `GET /admin_api/metrics`

Memerlukan izin `metrics:view` (admin dan pj). Respons:
`{"metrics": MetricsView}`.

```ts
type Window = { from: string; to: string; value: number };

type MetricsView = {
  generated_at: string;
  active_users: {
    calendar_month: Window;   // MAU: bulan kalender Asia/Jakarta berjalan
    last_7_days: Window;      // jendela bergulir 7 hari
    last_30_days: Window;     // jendela bergulir 30 hari
  };
  missions: {
    approved_total: number;
    approved_in_month: Window;   // lewat reviewed_at
  };
  donations: {
    paid_total: number;
    paid_in_month: Window;       // status confirmed, lewat confirmed_at
  };
  orders: {
    by_payment_status: Record<string, number>;      // seluruh riwayat
    by_fulfillment_status: Record<string, number>;  // seluruh riwayat
    created_in_month: Window;
  };
  community: {
    threads_in_month: Window;
    comments_in_month: Window;
    reactions_in_month: Window;
    reports_in_month: Window;
    reports_by_status: Record<string, number>;
  };
};
```

`from` bersifat inklusif dan `to` eksklusif. Untuk jendela bulan kalender,
`to` berada di awal bulan berikutnya — boleh di masa depan; yang terhitung
tetap hanya baris yang sudah ada.

**Arti "akun aktif" perlu dibaca dengan hati-hati.** Skema ini tidak menyimpan
jejak sesi sama sekali: tidak ada `last_seen`, `last_login`, maupun tabel
sesi. Keaktifan karena itu diturunkan dari **aksi bermakna** —
`COUNT(DISTINCT user_id)` atas gabungan sembilan tabel:

`presence`, `user_tickets`, `donations`, `mission_claims`, `shop_orders`,
`threads`, `thread_comments`, `thread_reactions`, `xp_ledger`.

Konsekuensinya: angka ini berarti **"pernah berbuat sesuatu"**, bukan
**"membuka aplikasi"**. Akun yang hanya membaca — membuka katalog, melihat
papan peringkat, membaca thread tanpa bereaksi — tidak akan pernah terhitung.
Itu keterbatasan skema, bukan kekeliruan perhitungan, dan disengaja tidak
ditutupi dengan kolom baru.

Tabel yang punya `deleted_at` menyaringnya; `thread_reactions` dan `xp_ledger`
memang tidak punya kolom itu karena keduanya append-only. `presence.user_id`
boleh `NULL` dan disaring tersendiri.

`reports_by_status` dihitung dari `thread_reports`, yang memakai
`reporter_user_id` — bukan `user_id` — sehingga tidak ikut dalam gabungan
akun aktif.

**Privasi.** Respons hanya berisi angka dan rentang tanggal. Tidak ada id
akun, nama, email, maupun nomor telepon, dan tidak ada analytics eksternal
yang menerima data ini.

**Beban.** Gabungan sembilan tabel dengan `DISTINCT`, dijalankan tiga kali
(satu per jendela). Endpoint ini hanya dipanggil pengurus saat membuka halaman
laporan — bukan jalur panas — dan sengaja tidak diberi cache supaya tidak ada
infrastruktur baru yang perlu dijaga.

### `GET /admin_api/users`

Memerlukan izin `users:view` (admin dan pj). Daftar akun berhalaman.

Query: `search` (mencocokkan `name`, `username`, atau `email`, tanpa membedakan
huruf besar-kecil), `role`, `page`, `per_page`.

```json
{ "users": [ /* AccountResponse */ ], "page": 1, "per_page": 20, "has_more": false }
```

Isinya `AccountResponse` — bentuk yang sama dengan `GET /admin_api/users/:id`.
Urutannya tetap `created_at DESC`; pengurutan tidak dapat dipilih pemanggil.
Akun yang terhapus tidak pernah muncul.

### `GET /admin_api/xp/audit_logs`

Memerlukan izin `mission:manage`. Berhalaman, query `entity_type` dan
`entity_id` (keduanya opsional).

```json
{ "audit_logs": [ /* AuditLog */ ], "page": 1, "per_page": 20, "has_more": false }
```

Hanya memuat baris milik modul gamification:

| Aksi | `entity_type` | `entity_id` |
| --- | --- | --- |
| `xp.adjust` | `xp_adjustment` | id baris `xp_ledger` |
| `xp_rule.update` | `xp_rule` | `source_type` |
| `level_rule.update` | `level_rule` | kosong (kurva ditimpa utuh) |

`entity_type` yang tidak dikenal menghasilkan halaman kosong, bukan baris
modul lain — penyaring dari luar disaring terhadap cakupan modul lebih dulu.
Isi `detail` tidak memuat nama maupun email.

### `AuditLog`

Bentuk baris audit yang dikembalikan oleh keempat endpoint riwayat
(`/admin_api/audit_logs`, `/admin_api/moderation/audit_logs`,
`/admin_api/shop/audit_logs`, `/admin_api/xp/audit_logs`):

```ts
type AuditLog = {
  id: string;
  actor_user_id: string | null;
  action: string;
  entity_type: string;
  entity_id: string;
  reason: string | null;
  detail: string | null;   // JSON yang di-encode sebagai string
  created_at: string;
};
```

`detail` adalah **string berisi JSON**, bukan objek — kolomnya `jsonb` di
database, tetapi dikirim sebagai teks. Klien yang membutuhkannya harus
memanggil `JSON.parse` sendiri.

`dedup_key` bertag `json:"-"` dan tidak pernah keluar.

---

## Catatan pemeliharaan

Tiga hal yang paling sering menjadi sumber drift saat menambah endpoint baru:

1. **Amplop.** Modul lama (`user`, `order`, `payment_method`) mengembalikan
   objek atau larik langsung. Modul Fase 1–4 memakai amplop berkunci. Ikuti
   modul tempat Anda bekerja.
2. **Kode status sukses.** Pembuatan di modul Fase 2–4 mengembalikan **201**,
   sedangkan modul lama mengembalikan 200.
3. **Kode status galat.** Modul lama kadang mengembalikan 500 untuk kesalahan
   pemanggil. Jangan menyalin kebiasaan itu; pakai `httperr.JSON` dengan
   sentinel `apperr` yang tepat.

Tidak ada generator tipe otomatis. `web/src/types/*.ts` dipelihara tangan dan
harus diperiksa terhadap dokumen ini setiap kali handler berubah. Ketika
keduanya berbeda, **kode adalah kebenarannya** — lalu perbaiki dokumen ini dan
tipenya.
