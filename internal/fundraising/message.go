package fundraising

// messageTransitions adalah satu-satunya definisi perpindahan status pesan.
//
// pending  -> approved: pesan boleh tampil di permukaan publik.
// pending  -> hidden:   pesan langsung disembunyikan.
// approved -> hidden:   pesan ditarik kembali setelah sempat tampil.
// hidden   -> approved: keputusan moderasi boleh direvisi.
//
// 'none' tidak punya perpindahan: donasi tanpa pesan tidak punya apa pun untuk
// dimoderasi.
var messageTransitions = map[string][]string{
	MessageNone:     {},
	MessagePending:  {MessageApproved, MessageHidden},
	MessageApproved: {MessageHidden},
	MessageHidden:   {MessageApproved},
}

// CanTransitionMessage melaporkan apakah perpindahan status pesan sah.
// Status yang tidak dikenal selalu ditolak.
func CanTransitionMessage(from, to string) bool {
	if from == to {
		return false
	}
	for _, allowed := range messageTransitions[from] {
		if allowed == to {
			return true
		}
	}
	return false
}
