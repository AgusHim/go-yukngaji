package order

import (
	"fmt"
	"strings"
	"time"

	"mainyuk/internal/ticket"
)

// ValidateTicketEvents memastikan setiap tiket yang dipesan benar-benar ada
// dan memang milik event order. Tanpa ini, pemesan bisa menempelkan tiket
// murah dari event lain ke order-nya.
//
// Fungsi murni: byID disiapkan pemanggil, sehingga bisa diuji tanpa database.
func ValidateTicketEvents(eventID string, reqs []UserTicket, byID map[string]*ticket.Ticket) error {
	if len(reqs) == 0 {
		return fmt.Errorf("order harus berisi minimal satu tiket")
	}
	for _, req := range reqs {
		t, ok := byID[req.TicketID]
		if !ok || t == nil {
			return fmt.Errorf("tiket %s tidak ditemukan", req.TicketID)
		}
		if t.EventID != eventID {
			return fmt.Errorf("tiket %s bukan milik event ini", req.TicketID)
		}
	}
	return nil
}

// ValidateTicketWindow memastikan waktu sekarang berada dalam jendela
// penjualan tiket. Tiket tanpa jendela (nol) dianggap selalu terbuka.
func ValidateTicketWindow(t *ticket.Ticket, now time.Time) error {
	if t == nil {
		return fmt.Errorf("tiket tidak ditemukan")
	}
	if !t.StartAt.IsZero() && now.Before(t.StartAt) {
		return fmt.Errorf("penjualan tiket %s belum dibuka", t.Name)
	}
	if !t.EndAt.IsZero() && now.After(t.EndAt) {
		return fmt.Errorf("penjualan tiket %s sudah ditutup", t.Name)
	}
	return nil
}

// NormalizeParticipantEmails mengumpulkan email peserta yang unik, huruf
// kecil, dan tidak kosong. Fungsi murni.
func NormalizeParticipantEmails(reqs []UserTicket) []string {
	seen := make(map[string]bool, len(reqs))
	emails := make([]string, 0, len(reqs))
	for _, req := range reqs {
		email := strings.ToLower(strings.TrimSpace(req.UserEmail))
		if email == "" || seen[email] {
			continue
		}
		seen[email] = true
		emails = append(emails, email)
	}
	return emails
}

// MatchParticipants mencari akun untuk tiap email peserta. lookup
// mengembalikan id akun dan apakah akun ditemukan.
//
// Email yang belum punya akun dipetakan ke nil: tiket tetap dibuat dan bisa
// diklaim saat orang tersebut mendaftar (lihat user_ticket.ClaimByEmail).
func MatchParticipants(emails []string, lookup func(email string) (string, bool)) map[string]*string {
	matched := make(map[string]*string, len(emails))
	for _, email := range emails {
		id, found := lookup(email)
		if !found || id == "" {
			matched[email] = nil
			continue
		}
		participantID := id
		matched[email] = &participantID
	}
	return matched
}
