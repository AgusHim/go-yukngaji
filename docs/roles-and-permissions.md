# Role dan Izin

Dokumen ini adalah rujukan tunggal untuk pertanyaan "siapa boleh apa" di
backend. Sumber kebenarannya ada di `internal/authz/authz.go`; kalau dokumen
ini berbeda dengan kode, kode yang menang.

## Daftar role

| Nilai di database | Konstanta | Arti |
| --- | --- | --- |
| `admin` | `authz.RoleAdmin` | Pengurus penuh. |
| `pj` | `authz.RolePJ` | Penanggung jawab; izinnya sama dengan admin pada modul misi, dana, produk, dan moderasi. |
| `ranger` | `authz.RoleRanger` | Petugas lapangan; boleh moderasi dan memindai tiket. |
| `user` | `authz.RoleMember` | Anggota. Nilai kanonik untuk akun baru. |
| `jamaah` | `authz.RoleJamaah` | Nilai lama untuk anggota yang dibuat dari alur tamu (presence). |

### `user` dan `jamaah` adalah satu tier

Fase 0 tidak menulis ulang data lama. Kedua nilai dianggap anggota lewat
`authz.IsMember(role)`. Kode baru **tidak boleh** membandingkan langsung
dengan string `"user"` atau `"jamaah"`; pakai `IsMember` supaya keduanya
tertangani.

Akun baru ditulis dengan `authz.RoleMember` (`user`). Baris lama tetap
`jamaah` dan tetap sah.

## Izin

`authz.Can(role, permission)` adalah satu-satunya cara memeriksa izin.

| Izin | admin | pj | ranger | user/jamaah |
| --- | --- | --- | --- | --- |
| `mission:manage` | ya | ya | tidak | tidak |
| `funds:manage` | ya | ya | tidak | tidak |
| `product:manage` | ya | ya | tidak | tidak |
| `moderation:moderate` | ya | ya | ya | tidak |

Moderator **belum** menjadi role tersendiri. Saat ini izin moderasi melekat
pada ranger dan pengurus. Bila nanti ditambahkan, langkahnya adalah menambah
konstanta role baru, memasukkannya ke `rolePermissions`, lalu memberi
`PermissionModerate` — bukan menambah pengecekan string baru di handler.

> Menambah moderator adalah perubahan otorisasi, jadi harus lewat
> persetujuan server: role disimpan di database dan diverifikasi ulang di
> setiap permintaan, tidak pernah dipercaya dari client.

## Middleware

Semua route yang butuh identitas memakai middleware di
`internal/auth/auth_middleware.go`.

| Middleware | Menerima | Menolak dengan |
| --- | --- | --- |
| `AuthAdmin` | `admin` | 401 |
| `AuthPJ` | `admin`, `pj` | 401 |
| `AuthRanger` | `ranger`, `pj`, `admin` | 401 |
| `AuthUser` | semua pengguna terautentikasi (role apa pun) | 401 |
| `AuthOptionalUser` | tidak pernah menolak | — |

`AuthUser` sengaja menerima semua role: yang membedakan hanya "sudah login".
Pembedaan wewenang dilakukan di lapisan izin (`authz.Can`), bukan dengan
menambah middleware per role.

`AuthOptionalUser` dipakai pada route yang perilakunya berbeda untuk tamu.
Handler **wajib** mengambil identitas lewat `user.FromContext(c)`, bukan dari
body permintaan.

## Identitas tidak pernah dari client

Aturan yang berlaku sejak Fase 0:

- `POST /api/comments`, `POST /api/comments/like`,
  `DELETE /api/comments/like/:id` — wajib login; `user_id` di body diabaikan
  sepenuhnya. Field-nya sudah dihapus dari DTO.
- `DELETE /api/comments/like/:id` hanya boleh dilakukan pemilik like atau
  pemegang `moderation:moderate`.
- `POST /api/polls/:id/respond` — tamu boleh menjawab, tetapi jawabannya
  disimpan dengan `user_id` NULL dan `is_verified = false`. Jawaban tamu
  tidak boleh dipakai sebagai bukti aktivitas akun (mis. bukti XP).
- `POST /api/presence` — bila login, `user_id` diambil dari token; jalur tamu
  tetap membuat akun `jamaah`.
- `POST /api/feedback` — wajib login, karena kolom `feedback.user_id` bersifat
  `NOT NULL`.
- `GET /ws/events/:id` — identitas hanya dari `?token=` yang valid. Parameter
  `user_id` dan `username` di query diabaikan. Tanpa token valid, koneksi
  tetap dibuka sebagai tamu anonim.

## Tabel route

| Route | Middleware |
| --- | --- |
| `POST /api/register`, `POST /api/login` | publik (rate limit per IP) |
| `GET /api/auth/google/login`, `/callback` | publik (state diverifikasi) |
| `POST /api/auth/otp/request`, `/verify` | publik (rate limit per email) |
| `GET /user_api/me` | `AuthUser` |
| `PUT /user_api/auth` | `AuthUser` |
| `POST /api/presence` | `AuthOptionalUser` |
| `GET /user_api/presence` | `AuthUser` |
| `GET /admin_api/presence` | `AuthAdmin` |
| `POST /ranger_api/event/:slug/presence` | `AuthRanger` |
| `POST /api/comments` | `AuthUser` |
| `GET /api/comments` | publik |
| `POST` / `DELETE /api/comments/like` | `AuthUser` |
| `GET /api/comments/like` | publik |
| `POST /api/feedback` | `AuthUser` |
| `GET /api/feedback` | `AuthAdmin` |
| `POST /api/polls/:id/respond` | `AuthOptionalUser` |
| `GET /api/polls/...` | publik |
| `/admin_api/polls/...` | `AuthAdmin` |
| `POST /user_api/orders` | `AuthUser` |
| `PUT /admin_api/orders/:id/verify` | `AuthAdmin` |
| `GET /ws/events/:id` | token di query (opsional) |

## Akun aktif

`GET /user_api/me` mengembalikan `{"user": <AccountResponse>}` — profil dan
role terbaru dari database. Frontend harus memakai ini alih-alih menyimpan
role di localStorage, karena role bisa berubah di server.

`AccountResponse` **tidak** memuat `password` dan `google_id`. Jangan
menambahkan kolom internal ke DTO ini; kalau butuh data internal, buat tipe
terpisah.
