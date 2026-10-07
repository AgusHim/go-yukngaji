package order

// allowedTransitions adalah tabel transisi status pembayaran.
//
// pending boleh menuju paid/failed/expired/cancelled; paid hanya boleh
// menuju refunded. Status akhir tidak boleh berpindah lagi, sehingga
// verifikasi berulang tidak bisa mengubah order yang sudah selesai.
var allowedTransitions = map[string]map[string]bool{
	StatusPending: {
		StatusPaid:      true,
		StatusFailed:    true,
		StatusExpired:   true,
		StatusCancelled: true,
	},
	StatusPaid: {
		StatusRefunded: true,
	},
	StatusExpired:   {},
	StatusCancelled: {},
	StatusFailed:    {},
	StatusRefunded:  {},
}

// IsValidStatus melaporkan apakah status termasuk status yang dikenal.
func IsValidStatus(status string) bool {
	_, ok := allowedTransitions[status]
	return ok
}

// CanTransition melaporkan apakah perpindahan status from -> to diizinkan.
// Fungsi murni: dipakai VerifyOrder dan diuji tanpa database.
func CanTransition(from, to string) bool {
	if !IsValidStatus(from) || !IsValidStatus(to) {
		return false
	}
	if from == to {
		return false
	}
	return allowedTransitions[from][to]
}
