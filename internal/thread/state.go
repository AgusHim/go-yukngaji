package thread

// Mesin status konten dan laporan.
//
// Bentuknya tabel transisi murni, sama seperti internal/fundraising/message.go:
// seluruh aturan ada di satu tempat yang bisa dibaca sekaligus dan diuji tanpa
// database, sementara repository hanya menjalankan UPDATE bersyarat.

// contentTransitions berlaku untuk thread maupun komentar — keduanya memakai
// himpunan status yang sama, jadi tidak ada gunanya menulisnya dua kali.
var contentTransitions = map[string][]string{
	StatusPublished: {StatusHidden, StatusDeleted},
	StatusHidden:    {StatusPublished, StatusDeleted},
	// deleted bersifat terminal: baris yang sudah dihapus tidak pernah
	// dihidupkan lagi. Menghapus adalah keputusan pemiliknya atau moderator,
	// dan menghidupkannya kembali akan membuat riwayat moderasi menyesatkan.
	StatusDeleted: {},
}

var reportTransitions = map[string][]string{
	ReportOpen: {ReportActioned, ReportDismissed},
	// Keduanya terminal: satu laporan diputuskan sekali. Laporan yang
	// keputusannya perlu diubah dibuat ulang oleh pelapornya, sehingga jejak
	// keputusan lama tetap utuh di audit_logs.
	ReportActioned:  {},
	ReportDismissed: {},
}

// CanTransitionThread melaporkan apakah thread boleh berpindah dari from ke to.
func CanTransitionThread(from, to string) bool {
	return canTransition(contentTransitions, from, to)
}

// CanTransitionThreadComment melaporkan apakah komentar boleh berpindah dari
// from ke to. Aturannya identik dengan thread.
func CanTransitionThreadComment(from, to string) bool {
	return canTransition(contentTransitions, from, to)
}

// CanTransitionReport melaporkan apakah laporan boleh berpindah status.
func CanTransitionReport(from, to string) bool {
	return canTransition(reportTransitions, from, to)
}

// IsKnownContentStatus melaporkan apakah status konten dikenal. Dipakai saat
// menyaring antrean moderasi: status tak dikenal tidak pernah disaringkan ke
// query, supaya salah ketik tidak diam-diam berubah menjadi "semua status".
func IsKnownContentStatus(status string) bool {
	_, ok := contentTransitions[status]
	return ok
}

// IsKnownReportStatus melaporkan apakah status laporan dikenal.
func IsKnownReportStatus(status string) bool {
	_, ok := reportTransitions[status]
	return ok
}

// canTransition menolak from == to dan status yang tidak dikenal, sehingga
// "sudah dalam status itu" tidak pernah disalahartikan sebagai perpindahan sah.
func canTransition(table map[string][]string, from, to string) bool {
	if from == to {
		return false
	}
	allowed, ok := table[from]
	if !ok {
		return false
	}
	for _, candidate := range allowed {
		if candidate == to {
			return true
		}
	}
	return false
}
