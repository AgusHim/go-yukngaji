package presence

import "time"

// JakartaLocation adalah zona waktu acuan untuk perhitungan hari check-in.
// Jakarta tidak memakai daylight saving, jadi offset tetap aman.
var JakartaLocation = time.FixedZone("Asia/Jakarta", 7*60*60)

// CheckInDate mengembalikan tanggal (tanpa jam) di zona loc.
//
// Hasilnya dinyatakan sebagai tengah malam UTC supaya aman disimpan ke kolom
// SQL bertipe date tanpa bergeser sehari karena konversi zona waktu.
func CheckInDate(t time.Time, loc *time.Location) time.Time {
	if loc == nil {
		loc = JakartaLocation
	}
	year, month, day := t.In(loc).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}
