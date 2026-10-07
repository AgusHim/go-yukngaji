package fundraising

// ComputeReport merangkum agregat mentah menjadi angka laporan.
//
// Dua angka yang sengaja berbeda dan tidak boleh tertukar di UI:
//
//   - ConfirmedAmount adalah PROGRES, selalu bruto. Biaya dan penggunaan dana
//     tidak menguranginya, sesuai keputusan bahwa donasi dicatat bruto.
//   - NetAmount adalah sisa dana: terkumpul dikurangi refund, biaya, dan
//     penggunaan. Nilainya boleh negatif, dan itu memang sinyal yang berguna —
//     memagarinya di nol justru menyembunyikan dana yang terpakai berlebih.
//
// Pending dan rejected tidak pernah masuk ke angka mana pun: uangnya belum
// (atau tidak akan pernah) diterima.
func ComputeReport(in ReportInput) ReportTotals {
	return ReportTotals{
		ConfirmedAmount: in.ConfirmedAmount,
		ConfirmedCount:  in.ConfirmedCount,
		PendingAmount:   in.PendingAmount,
		PendingCount:    in.PendingCount,
		RejectedAmount:  in.RejectedAmount,
		RejectedCount:   in.RejectedCount,
		RefundedAmount:  in.RefundedAmount,
		RefundedCount:   in.RefundedCount,
		FeeAmount:       in.FeeAmount,
		UsageAmount:     in.UsageAmount,
		NetAmount:       in.ConfirmedAmount - in.RefundedAmount - in.FeeAmount - in.UsageAmount,
	}
}
