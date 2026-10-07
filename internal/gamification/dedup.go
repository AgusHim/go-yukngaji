package gamification

import "github.com/google/uuid"

// Kunci dedup adalah satu-satunya jaminan anti-pemberian-ganda. Kuncinya
// disimpan di kolom xp_ledger.dedup_key yang berindex unik, sehingga dua
// request paralel tidak mungkin menghasilkan dua baris.
//
// Format: <source_type>:<user_id>[:<ref>]
// Id berupa uuid/heksadesimal tanpa ':', jadi ':' aman dipakai sebagai
// pemisah dan tidak bisa menimbulkan tabrakan antar-komponen.

// ProfileDedupKey memberi satu reward seumur akun. Kuncinya sengaja tidak
// memuat waktu atau referensi apa pun, sehingga mengubah-ubah field profil
// tidak bisa dipakai memanen XP berulang.
func ProfileDedupKey(userID string) string {
	return SourceProfileComplete + ":" + userID
}

// CheckInDedupKey memberi satu reward per akun per event. Event beberapa hari
// tetap hanya memberi satu reward, dan memegang beberapa tiket untuk event
// yang sama juga tidak menambah XP.
func CheckInDedupKey(userID, eventID string) string {
	return SourceCheckIn + ":" + userID + ":" + eventID
}

// MissionDedupKey mengikat reward pada satu klaim, sehingga menyetujui ulang
// klaim yang sama tidak menambah XP.
func MissionDedupKey(userID, claimID string) string {
	return SourceMission + ":" + userID + ":" + claimID
}

// DonationDedupKey mengikat reward pada satu donasi, sehingga mengonfirmasi
// ulang donasi yang sama tidak menambah XP.
func DonationDedupKey(userID, donationID string) string {
	return SourceDonation + ":" + userID + ":" + donationID
}

// DonationReversalDedupKey membalik reward satu donasi tepat satu kali.
// Kuncinya deterministik (bukan uuid), sehingga mengulang refund menghasilkan
// RowsAffected = 0 alih-alih pembalikan kedua.
func DonationReversalDedupKey(userID, donationID string) string {
	return SourceDonationReversal + ":" + userID + ":" + donationID
}

// ShopOrderDedupKey mengikat reward pada satu pesanan merchandise, sehingga
// mengonfirmasi ulang pesanan yang sama tidak menambah XP.
func ShopOrderDedupKey(userID, orderID string) string {
	return SourceShopOrder + ":" + userID + ":" + orderID
}

// ShopOrderReversalDedupKey membalik reward satu pesanan tepat satu kali.
// Kuncinya deterministik (bukan uuid), sehingga mengulang refund menghasilkan
// RowsAffected = 0 alih-alih pembalikan kedua.
func ShopOrderReversalDedupKey(userID, orderID string) string {
	return SourceShopOrderReversal + ":" + userID + ":" + orderID
}

// AdjustmentDedupKey selalu unik: koreksi admin tidak boleh saling menimpa.
func AdjustmentDedupKey() string {
	return SourceAdjustment + ":" + uuid.NewString()
}
