package fundraising

import "testing"

func TestComputeReport(t *testing.T) {
	cases := []struct {
		name string
		in   ReportInput
		want ReportTotals
	}{
		{
			name: "belum ada donasi",
			in:   ReportInput{},
			want: ReportTotals{},
		},
		{
			name: "progres bruto tidak dikurangi biaya",
			in: ReportInput{
				ConfirmedAmount: 1_000_000,
				ConfirmedCount:  4,
				FeeAmount:       25_000,
				UsageAmount:     300_000,
			},
			want: ReportTotals{
				ConfirmedAmount: 1_000_000,
				ConfirmedCount:  4,
				FeeAmount:       25_000,
				UsageAmount:     300_000,
				NetAmount:       675_000,
			},
		},
		{
			name: "refund mengurangi sisa dana",
			in: ReportInput{
				ConfirmedAmount: 500_000,
				ConfirmedCount:  2,
				RefundedAmount:  100_000,
				RefundedCount:   1,
			},
			want: ReportTotals{
				ConfirmedAmount: 500_000,
				ConfirmedCount:  2,
				RefundedAmount:  100_000,
				RefundedCount:   1,
				NetAmount:       400_000,
			},
		},
		{
			name: "pending dan rejected tidak pernah dihitung sebagai progres",
			in: ReportInput{
				ConfirmedAmount: 200_000,
				ConfirmedCount:  1,
				PendingAmount:   900_000,
				PendingCount:    3,
				RejectedAmount:  400_000,
				RejectedCount:   2,
			},
			want: ReportTotals{
				ConfirmedAmount: 200_000,
				ConfirmedCount:  1,
				PendingAmount:   900_000,
				PendingCount:    3,
				RejectedAmount:  400_000,
				RejectedCount:   2,
				NetAmount:       200_000,
			},
		},
		{
			name: "sisa dana boleh negatif dan tidak dipagari",
			in: ReportInput{
				ConfirmedAmount: 100_000,
				ConfirmedCount:  1,
				UsageAmount:     250_000,
			},
			want: ReportTotals{
				ConfirmedAmount: 100_000,
				ConfirmedCount:  1,
				UsageAmount:     250_000,
				NetAmount:       -150_000,
			},
		},
		{
			name: "semua donasi dikembalikan",
			in: ReportInput{
				ConfirmedAmount: 300_000,
				ConfirmedCount:  3,
				RefundedAmount:  300_000,
				RefundedCount:   3,
			},
			want: ReportTotals{
				ConfirmedAmount: 300_000,
				ConfirmedCount:  3,
				RefundedAmount:  300_000,
				RefundedCount:   3,
				NetAmount:       0,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ComputeReport(tc.in)
			if got != tc.want {
				t.Errorf("ComputeReport() = %+v, ingin %+v", got, tc.want)
			}
		})
	}
}
