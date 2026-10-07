package thread

import (
	"strings"

	"mainyuk/internal/community"
)

// Auto-post: membagikan aktivitas anggota ke feed.
//
// Kebijakannya opt-out — berbagi adalah default, anggota mematikannya sendiri.
// Karena default-nya nyala, seluruh pengaman privasi harus ditegakkan di sini,
// bukan diserahkan ke ingatan pemanggil. Satu fungsi murni, satu tempat.

// AutoPostContext merangkum keadaan yang menentukan boleh tidaknya sebuah
// aktivitas dibagikan otomatis.
type AutoPostContext struct {
	Activity string // SourceEventRegistration | SourceDonation | SourceMission
	Prefs    *SharePrefs
	// Profile adalah profil mentah pemilik aktivitas. Ia membawa IsBlocked dan
	// ProfileVisibility, dua alasan penolakan yang tidak terlihat dari DTO
	// publik mana pun.
	Profile *community.Profile
	// IsAnonymous menandai donasi anonim. Donasi anonim menyembunyikan nominal
	// sekaligus identitas donaturnya di halaman campaign; feed tidak boleh
	// menjadi jalan memutar untuk membukanya.
	IsAnonymous bool
	SourceRef   string
	Title       string
}

// ShouldAutoPost menentukan apakah sebuah aktivitas boleh membuat thread.
//
// Menolak ketika: aktivitasnya tidak dikenal; preferensinya dimatikan; profil
// pemiliknya belum ada, privat, atau terblokir; aktivitasnya anonim; rujukan
// sumbernya kosong; atau judul sumbernya kosong.
//
// Menerima (default) ketika preferensi belum pernah diisi — ketiadaan baris
// berarti semua aktivitas dibagikan, dan itu memang kebijakan opt-out yang
// dipilih. Tidak ada backfill yang dibutuhkan.
func ShouldAutoPost(in AutoPostContext) bool {
	if !IsKnownActivity(in.Activity) {
		return false
	}
	if strings.TrimSpace(in.SourceRef) == "" {
		return false
	}
	if strings.TrimSpace(in.Title) == "" {
		return false
	}
	if in.IsAnonymous {
		return false
	}
	if !ResolveSharePrefs(in.Prefs, in.Activity) {
		return false
	}
	// Profil yang belum ada diperlakukan sebagai "belum boleh": auto-post
	// menulis atas nama seseorang, dan menulis atas nama akun yang bahkan
	// belum punya alias bukan hal yang pantas dilakukan diam-diam.
	if in.Profile == nil {
		return false
	}
	if IsBlockedAccount(in.Profile) {
		return false
	}
	if IsPrivateAccount(in.Profile) {
		return false
	}
	return true
}

// ResolveSharePrefs melaporkan apakah satu jenis aktivitas dibagikan menurut
// preferensi anggota.
//
// prefs nil berarti anggota belum pernah membuka pengaturannya — dan karena
// kebijakannya opt-out, jawabannya true. Aktivitas yang tidak dikenal dijawab
// false: sumber baru yang belum dipikirkan tidak boleh otomatis ikut terbit
// hanya karena default-nya "nyala".
func ResolveSharePrefs(prefs *SharePrefs, activity string) bool {
	if prefs == nil {
		return IsKnownActivity(activity)
	}
	switch activity {
	case SourceEventRegistration:
		return prefs.ShareEventRegistration
	case SourceDonation:
		return prefs.ShareDonation
	case SourceMission:
		return prefs.ShareMission
	}
	return false
}

// IsKnownActivity melaporkan apakah jenis aktivitas dikenal paket ini.
func IsKnownActivity(activity string) bool {
	_, ok := autoPostPrefixes[activity]
	return ok
}

// autoPostPrefixes adalah awal judul untuk tiap jenis aktivitas. Judul sumber
// diambil dari modul asalnya dan dikirim pemanggil, sehingga paket ini tidak
// perlu mengimpor internal/order, internal/fundraising, atau internal/mission.
var autoPostPrefixes = map[string]string{
	SourceEventRegistration: "Mendaftar ",
	SourceDonation:          "Berbagi kebaikan di ",
	SourceMission:           "Menyelesaikan misi ",
}

var autoPostOpeners = map[string]string{
	SourceEventRegistration: "Saya baru saja mendaftar ",
	SourceDonation:          "Saya baru saja berdonasi untuk ",
	SourceMission:           "Saya baru saja menyelesaikan misi ",
}

// AutoPostTitle menyusun judul thread dari jenis aktivitas dan judul sumbernya.
// Mengembalikan string kosong untuk aktivitas tak dikenal — pemanggil tidak
// boleh menebak-nebak, dan ShouldAutoPost sudah menolaknya lebih dulu.
func AutoPostTitle(activity, sourceTitle string) string {
	prefix, ok := autoPostPrefixes[activity]
	if !ok {
		return ""
	}
	return clampTitle(prefix + flatten(sourceTitle))
}

// AutoPostBody menyusun badan thread.
//
// Nominal donasi tidak pernah muncul di sini, dalam bentuk apa pun. Nominal
// punya aturan privasinya sendiri di internal/fundraising, dan feed anonim
// adalah tempat paling mudah untuk membocorkannya tanpa sengaja — cukup dengan
// menambahkan satu format string.
func AutoPostBody(activity, sourceTitle string) string {
	opener, ok := autoPostOpeners[activity]
	if !ok {
		return ""
	}
	source := flatten(sourceTitle)
	if source == "" {
		return ""
	}
	switch activity {
	case SourceDonation:
		return opener + source + ". Semoga makin banyak yang terbantu."
	case SourceMission:
		return opener + source + ". Satu langkah kecil lagi selesai."
	default:
		return opener + source + ". Siapa lagi yang ikut?"
	}
}

// clampTitle memotong judul agar tidak melampaui batas kolom, menyisakan satu
// karakter untuk elipsis supaya pemotongannya terlihat.
func clampTitle(title string) string {
	title = flatten(title)
	runes := []rune(title)
	if len(runes) <= ThreadTitleMax {
		return title
	}
	return strings.TrimSpace(string(runes[:ThreadTitleMax-1])) + "…"
}

// flatten merapikan teks menjadi satu baris. Judul thread tidak boleh memuat
// baris baru, dan judul sumber datang dari modul lain yang tidak tahu itu.
func flatten(text string) string {
	return strings.Join(strings.Fields(NormalizeContent(text)), " ")
}
