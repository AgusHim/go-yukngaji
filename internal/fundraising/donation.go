package fundraising

import "strings"

// donationTransitions adalah satu-satunya definisi perpindahan status donasi.
//
// pending -> confirmed: transfer terverifikasi pengurus.
// pending -> rejected:  transfer tidak ditemukan atau nominalnya tidak cocok.
// pending -> cancelled: donatur membatalkan sebelum transfer.
// confirmed -> refunded: dana dikembalikan; XP yang pernah diberikan dibalik.
//
// rejected, cancelled, dan refunded bersifat terminal. Donasi yang batal tidak
// dihidupkan kembali — donatur membuat donasi baru, sehingga riwayatnya utuh.
var donationTransitions = map[string][]string{
	DonationPending:   {DonationConfirmed, DonationRejected, DonationCancelled},
	DonationConfirmed: {DonationRefunded},
	DonationRejected:  {},
	DonationCancelled: {},
	DonationRefunded:  {},
}

// CanTransitionDonation melaporkan apakah perpindahan status donasi sah.
// Status yang tidak dikenal selalu ditolak.
func CanTransitionDonation(from, to string) bool {
	if from == to {
		return false
	}
	for _, allowed := range donationTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}

// Batas nominal donasi. Nilainya berupa rupiah bulat, sesuai konvensi repo:
// tidak ada pecahan, tidak ada float.
const (
	// MinDonationAmount adalah nominal terkecil yang diterima.
	MinDonationAmount = 1000
	// MaxDonationAmount adalah pagar kewarasan. Donasi di atas ini hampir pasti
	// salah ketik, dan menolaknya lebih baik daripada menyimpan angka yang
	// merusak laporan.
	MaxDonationAmount = 1_000_000_000_000
)

// ValidateDonationAmount memvalidasi nominal donasi.
func ValidateDonationAmount(amount int) error {
	if amount <= 0 {
		return invalid("nominal donasi harus lebih dari nol")
	}
	if amount < MinDonationAmount {
		return invalid("nominal donasi minimal Rp %d", MinDonationAmount)
	}
	if amount > MaxDonationAmount {
		return invalid("nominal donasi melebihi batas wajar")
	}
	return nil
}

// NormalizeMessage merapikan pesan donasi.
//
// Pesan kosong menjadi nil, dan statusnya 'none' — bukan 'pending' — supaya
// donasi tanpa pesan tidak masuk antrean moderasi.
func NormalizeMessage(message *string) (*string, string) {
	if message == nil {
		return nil, MessageNone
	}
	trimmed := strings.TrimSpace(*message)
	if trimmed == "" {
		return nil, MessageNone
	}
	return &trimmed, MessagePending
}

// IsValidMessageStatus melaporkan apakah status pesan dikenal.
func IsValidMessageStatus(status string) bool {
	switch status {
	case MessageNone, MessagePending, MessageApproved, MessageHidden:
		return true
	}
	return false
}
