package order

import "time"

// ShouldExpire melaporkan apakah order pending telah mencapai batas pembayaran.
// Order selain pending tidak boleh diubah otomatis oleh worker.
func ShouldExpire(status string, expiresAt, now time.Time) bool {
	return status == StatusPending && !expiresAt.IsZero() && !now.Before(expiresAt)
}
