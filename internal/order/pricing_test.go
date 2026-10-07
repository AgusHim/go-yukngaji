package order

import "testing"

func TestComputeOrderAmounts(t *testing.T) {
	tests := []struct {
		name        string
		prices      []int
		donation    int
		adminFee    int
		wantTickets int
		wantTotal   int
	}{
		{
			name:        "tanpa tiket, tanpa donasi",
			prices:      nil,
			wantTickets: 0,
			wantTotal:   0,
		},
		{
			name:        "dua tiket berbayar",
			prices:      []int{50000, 75000},
			wantTickets: 125000,
			wantTotal:   125000,
		},
		{
			name:        "tiket gratis dengan donasi tetap ditagih",
			prices:      []int{0},
			donation:    20000,
			wantTickets: 0,
			wantTotal:   20000,
		},
		{
			name:        "tiket gratis dengan biaya admin tetap ditagih",
			prices:      []int{0, 0},
			adminFee:    5000,
			wantTickets: 0,
			wantTotal:   5000,
		},
		{
			name:        "semua komponen dijumlahkan",
			prices:      []int{30000},
			donation:    10000,
			adminFee:    2500,
			wantTickets: 30000,
			wantTotal:   42500,
		},
		{
			name:        "harga negatif diabaikan",
			prices:      []int{-1000, 40000},
			wantTickets: 40000,
			wantTotal:   40000,
		},
		{
			name:        "donasi dan biaya negatif dianggap nol",
			prices:      []int{10000},
			donation:    -5000,
			adminFee:    -1000,
			wantTickets: 10000,
			wantTotal:   10000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ComputeOrderAmounts(tt.prices, tt.donation, tt.adminFee)
			if got.TicketTotal != tt.wantTickets {
				t.Errorf("TicketTotal = %d, want %d", got.TicketTotal, tt.wantTickets)
			}
			if got.Total != tt.wantTotal {
				t.Errorf("Total = %d, want %d", got.Total, tt.wantTotal)
			}
			if got.Total != got.TicketTotal+got.Donation+got.AdminFee {
				t.Errorf("Total (%d) harus sama dengan jumlah komponen", got.Total)
			}
		})
	}
}

func TestDeriveStatus(t *testing.T) {
	tests := []struct {
		name  string
		total int
		want  string
	}{
		{"tagihan nol langsung paid", 0, StatusPaid},
		{"total negatif tetap paid", -1, StatusPaid},
		{"tiket gratis dengan donasi belum paid", 20000, StatusPending},
		{"tiket berbayar belum paid", 50000, StatusPending},
		{"hanya biaya admin pun belum paid", 2500, StatusPending},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DeriveStatus(tt.total); got != tt.want {
				t.Errorf("DeriveStatus(%d) = %q, want %q", tt.total, got, tt.want)
			}
		})
	}
}
