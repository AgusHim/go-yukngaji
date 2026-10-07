package order

import (
	"testing"
	"time"

	"mainyuk/internal/ticket"
)

func TestValidateTicketEvents(t *testing.T) {
	byID := map[string]*ticket.Ticket{
		"tkt-a": {ID: "tkt-a", Name: "Reguler", EventID: "evt-1"},
		"tkt-b": {ID: "tkt-b", Name: "VIP", EventID: "evt-2"},
	}

	tests := []struct {
		name    string
		eventID string
		reqs    []UserTicket
		wantErr bool
	}{
		{
			name:    "tiket sesuai event",
			eventID: "evt-1",
			reqs:    []UserTicket{{TicketID: "tkt-a", UserEmail: "a@example.com"}},
		},
		{
			name:    "order tanpa tiket ditolak",
			eventID: "evt-1",
			reqs:    nil,
			wantErr: true,
		},
		{
			name:    "tiket milik event lain ditolak",
			eventID: "evt-1",
			reqs:    []UserTicket{{TicketID: "tkt-b", UserEmail: "a@example.com"}},
			wantErr: true,
		},
		{
			name:    "tiket tidak ada ditolak",
			eventID: "evt-1",
			reqs:    []UserTicket{{TicketID: "tkt-hantu"}},
			wantErr: true,
		},
		{
			name:    "satu tiket salah membatalkan seluruh order",
			eventID: "evt-1",
			reqs: []UserTicket{
				{TicketID: "tkt-a"},
				{TicketID: "tkt-b"},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTicketEvents(tt.eventID, tt.reqs, byID)
			if tt.wantErr && err == nil {
				t.Error("harusnya error, tetapi nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("tidak boleh error, dapat: %v", err)
			}
		})
	}
}

func TestValidateTicketWindow(t *testing.T) {
	now := time.Date(2026, time.October, 2, 12, 0, 0, 0, time.UTC)
	start := now.Add(-time.Hour)
	end := now.Add(time.Hour)

	tests := []struct {
		name    string
		ticket  *ticket.Ticket
		wantErr bool
	}{
		{name: "di dalam jendela", ticket: &ticket.Ticket{Name: "Reguler", StartAt: start, EndAt: end}},
		{name: "tanpa jendela selalu terbuka", ticket: &ticket.Ticket{Name: "Reguler"}},
		{name: "belum dibuka", ticket: &ticket.Ticket{Name: "Reguler", StartAt: now.Add(time.Hour), EndAt: end}, wantErr: true},
		{name: "sudah ditutup", ticket: &ticket.Ticket{Name: "Reguler", StartAt: start, EndAt: now.Add(-time.Minute)}, wantErr: true},
		{name: "tiket nil", ticket: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTicketWindow(tt.ticket, now)
			if tt.wantErr && err == nil {
				t.Error("harusnya error, tetapi nil")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("tidak boleh error, dapat: %v", err)
			}
		})
	}
}

func TestNormalizeParticipantEmails(t *testing.T) {
	reqs := []UserTicket{
		{UserEmail: "  Budi@Example.com "},
		{UserEmail: "budi@example.com"},
		{UserEmail: ""},
		{UserEmail: "   "},
		{UserEmail: "Siti@Example.com"},
	}

	got := NormalizeParticipantEmails(reqs)
	want := []string{"budi@example.com", "siti@example.com"}

	if len(got) != len(want) {
		t.Fatalf("jumlah email = %d (%v), want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("email[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestMatchParticipants(t *testing.T) {
	lookup := func(email string) (string, bool) {
		switch email {
		case "budi@example.com":
			return "usr-budi", true
		case "akun-tanpa-id@example.com":
			return "", true
		default:
			return "", false
		}
	}

	got := MatchParticipants(
		[]string{"budi@example.com", "baru@example.com", "akun-tanpa-id@example.com"},
		lookup,
	)

	if got["budi@example.com"] == nil || *got["budi@example.com"] != "usr-budi" {
		t.Errorf("email terdaftar harus dipetakan ke akunnya, dapat %v", got["budi@example.com"])
	}
	if got["baru@example.com"] != nil {
		t.Error("email tanpa akun harus dipetakan ke nil supaya bisa diklaim nanti")
	}
	if got["akun-tanpa-id@example.com"] != nil {
		t.Error("id kosong harus diperlakukan sebagai tidak ditemukan")
	}
}
