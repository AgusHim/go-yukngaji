package mission

import (
	"fmt"
	"time"

	"mainyuk/internal/period"
)

// PeriodKey menghitung kunci periode sebuah misi pada satu titik waktu.
//
// Kunci inilah yang membuat batas klaim dapat ditegakkan per periode: misi
// harian boleh diklaim sekali setiap hari, misi mingguan sekali setiap minggu,
// dan seterusnya. Perhitungan dilakukan pada zona Asia/Jakarta, sehingga
// pergantian hari mengikuti waktu setempat, bukan UTC.
//
// Misi special tidak dipecah per periode: seluruh durasinya adalah satu
// periode, jadi batas klaimnya berlaku untuk keseluruhan masa berlaku.
func PeriodKey(missionType string, t time.Time) string {
	loc := period.JakartaLocation

	switch missionType {
	case TypeDaily:
		start, _ := period.DayRange(t, loc)
		return "daily:" + start.Format("2006-01-02")
	case TypeWeekly:
		_, _, isoYear, isoWeek := period.ISOWeekRange(t, loc)
		return fmt.Sprintf("weekly:%04d-W%02d", isoYear, isoWeek)
	default:
		return "special"
	}
}

// ClaimWindowOpen melaporkan apakah klaim boleh dibuat pada waktu now.
//
// Jendela klaim adalah irisan antara masa berlaku misi dan waktu sekarang:
// misi yang belum dipublikasikan, belum mulai, atau sudah berakhir tidak
// dapat diklaim, meskipun barisnya ada di database.
func ClaimWindowOpen(now, startsAt, endsAt time.Time, isPublished bool) bool {
	if !isPublished {
		return false
	}
	if now.Before(startsAt) {
		return false
	}
	// ends_at bersifat eksklusif: misi yang berakhir pukul 23:59 tidak dapat
	// diklaim pada pukul 23:59:00 berikutnya.
	return now.Before(endsAt)
}

// CanClaim melaporkan apakah jumlah klaim aktif masih di bawah batas.
func CanClaim(claimLimit, activeCount int) bool {
	if claimLimit < 1 {
		return false
	}
	return activeCount < claimLimit
}

// RequiredProof melaporkan apakah klaim dengan mode verifikasi ini wajib
// menyertakan bukti.
func RequiredProof(verificationMode string) bool {
	return verificationMode == VerificationProofApproval
}

// IsAutoApproved melaporkan apakah klaim dengan mode verifikasi ini langsung
// disetujui sistem tanpa antrean pengurus.
//
// Mode self_claim sengaja tidak termasuk: klaim mandiri tetap tercatat sebagai
// pending supaya ada jejak yang dapat diperiksa, tetapi tanpa kewajiban bukti.
func IsAutoApproved(verificationMode string) bool {
	return verificationMode == VerificationAuto
}

// ValidateMissionType memeriksa jenis misi yang dikenal.
func ValidateMissionType(t string) error {
	switch t {
	case TypeDaily, TypeWeekly, TypeSpecial:
		return nil
	default:
		return invalid("jenis misi tidak dikenal: %s", t)
	}
}

// ValidateVerificationMode memeriksa mode verifikasi yang dikenal.
func ValidateVerificationMode(m string) error {
	switch m {
	case VerificationAuto, VerificationSelfClaim, VerificationProofApproval:
		return nil
	default:
		return invalid("mode verifikasi tidak dikenal: %s", m)
	}
}

// ValidateClaimStatus memeriksa status klaim yang dapat diminta admin.
func ValidateClaimStatus(s string) error {
	switch s {
	case StatusPending, StatusApproved, StatusRejected, StatusCancelled:
		return nil
	default:
		return invalid("status klaim tidak dikenal: %s", s)
	}
}
