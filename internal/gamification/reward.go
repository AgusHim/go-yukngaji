package gamification

import (
	"time"

	"mainyuk/internal/period"
)

// CappedRewardDecision memutuskan XP yang boleh diberikan untuk satu sumber
// yang dibatasi bulanan (donasi terkonfirmasi, pesanan merchandise terbayar),
// dengan memperhatikan batas bulanan sumber itu.
//
// monthlyCap adalah batas jumlah SUMBER yang diberi reward per bulan, bukan
// batas jumlah XP. Reward sumber-sumber ini besarnya tetap, jadi batas dalam
// satuan XP akan membuat sumber pertama menghabiskan seluruh kuota dan sumber
// berikutnya bernilai nol — bukan yang dimaksud "maksimum 3 per bulan".
//
// Batas <= 0 berarti tanpa batas. Fungsi ini murni dari angka supaya aturannya
// dapat diuji tanpa database. Pemanggil bertanggung jawab menjalankan
// hitung-lalu-sisip ini di dalam satu transaksi berkunci: rewardedCount dibaca
// dari ledger, dan tanpa serialisasi dua sumber bersamaan dapat sama-sama
// membaca "belum mencapai batas".
func CappedRewardDecision(ruleXP, monthlyCap, rewardedCount int) int {
	if ruleXP <= 0 {
		return 0
	}
	if monthlyCap <= 0 {
		return ruleXP
	}
	if rewardedCount >= monthlyCap {
		return 0
	}
	return ruleXP
}

// MonthlyCapWindow mengembalikan rentang bulan kalender yang berlaku untuk
// batas XP bulanan, dihitung di zona Asia/Jakarta supaya pergantian bulan tidak
// bergantung pada zona server.
//
// Batas bulanan donasi dan pesanan merchandise memakai jendela yang sama:
// keduanya dibatasi per bulan kalender yang sama pula, sehingga tidak ada
// alasan memisahkan perhitungannya.
func MonthlyCapWindow(now time.Time) (time.Time, time.Time) {
	return period.MonthRange(now, period.JakartaLocation)
}
