package thread

import (
	"strings"

	"mainyuk/internal/community"
)

// Satu titik penegakan identitas untuk seluruh paket ini.
//
// Aturan mainnya: tidak ada permukaan publik yang boleh menampilkan apa pun
// selain username. Username akun pada sistem ini adalah nama samaran yang
// dipilih anggota. Bila username kosong, penulisnya privat, atau akunnya
// terblokir, nama tampilnya "Anonim".
//
// Perbedaan penting dengan community.Profile.DisplayAlias: fungsi itu menerima
// nama akun sebagai fallback, sehingga akun tanpa alias tampil dengan nama
// aslinya. Di sini fallback itu tidak ada sama sekali, dan memang tidak boleh
// ada — kalau ada satu saja jalur yang memakainya, feed ini bukan feed anonim.

// ThreadAuthorName menentukan nama tampil penulis dari username akun.
func ThreadAuthorName(p *community.Profile) string {
	if p == nil {
		return AnonymousAuthorName
	}
	if p.IsBlocked {
		return AnonymousAuthorName
	}
	if p.ProfileVisibility == community.VisibilityPrivate {
		return AnonymousAuthorName
	}
	username := strings.TrimSpace(p.Username)
	if username == "" {
		return AnonymousAuthorName
	}
	return username
}

// ToThreadAuthor menyusun bentuk publik penulis.
//
// Selalu mengembalikan penulis — bukan nil — supaya tidak ada pemanggil yang
// terpaksa memutuskan sendiri apa yang ditampilkan saat profilnya tidak ada.
// Keputusan itu ada di sini, satu kali.
//
// Avatar ikut disembunyikan ketika penulisnya dianonimkan: avatar adalah
// pengait identitas yang sama kuatnya dengan username.
func ToThreadAuthor(p *community.Profile) *ThreadAuthor {
	name := ThreadAuthorName(p)

	author := &ThreadAuthor{Alias: name}
	if p == nil {
		return author
	}

	author.PublicID = p.PublicID
	author.ShowBadges = p.ShowBadges
	if name != AnonymousAuthorName {
		author.AvatarURL = p.AvatarURL
	}
	return author
}

// ToModerationAuthor menyusun penulis untuk antrean moderasi.
//
// Petugas berizin melihat aliasnya walaupun profilnya privat atau terblokir —
// itulah "pengaitan alias ke akun asli hanya untuk petugas berizin". Yang tetap
// tidak ikut: id akun, email, telepon, dan seluruh kolom internal lain; bentuk
// masuknya pun sudah PublicProfile, yang sejak awal tidak memuatnya.
func ToModerationAuthor(p *community.PublicProfile) *ThreadAuthor {
	if p == nil {
		return &ThreadAuthor{Alias: AnonymousAuthorName}
	}
	alias := strings.TrimSpace(p.Alias)
	if alias == "" {
		alias = AnonymousAuthorName
	}
	return &ThreadAuthor{
		PublicID:   p.PublicID,
		Alias:      alias,
		AvatarURL:  p.AvatarURL,
		ShowBadges: p.ShowBadges,
	}
}

// IsBlockedAccount melaporkan apakah akun pemilik profil sedang dibatasi.
// Profil yang belum ada dianggap tidak terblokir — akun baru tidak boleh
// tertahan hanya karena baris profilnya belum sempat dibuat.
func IsBlockedAccount(p *community.Profile) bool {
	return p != nil && p.IsBlocked
}

// IsPrivateAccount melaporkan apakah profilnya privat. Dipakai auto-post:
// aktivitas pemilik profil privat tidak pernah dibagikan ke feed.
func IsPrivateAccount(p *community.Profile) bool {
	return p != nil && p.ProfileVisibility == community.VisibilityPrivate
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
