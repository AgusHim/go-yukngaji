package fundraising

import (
	"strings"

	"mainyuk/internal/community"
)

// ResolveDonorName menentukan nama donatur yang boleh tampil di permukaan
// publik, beserta penanda apakah identitasnya disembunyikan.
//
// Aturannya ditulis satu kali di sini supaya tidak ada endpoint yang bisa lupa
// menerapkannya:
//
//   - Profil tidak ada: donatur belum pernah membuat identitas komunitas, jadi
//     tidak ada nama yang boleh dipublikasikan atas namanya.
//   - Akun terblokir: identitasnya tidak pernah tampil, di mana pun.
//   - Profil privat: pemiliknya sudah memilih untuk tidak tampil.
//   - Selain itu: alias bila ada, kalau tidak nama akun sebagai cadangan.
//
// Perhatikan bahwa fungsi ini TIDAK menangani is_anonymous: itu atribut donasi,
// bukan profil, dan diterapkan ToPublicDonation.
func ResolveDonorName(p *community.Profile, fallback string) (string, bool) {
	if p == nil || p.IsBlocked || p.ProfileVisibility == community.VisibilityPrivate {
		return AnonymousDonorName, true
	}
	if p.Alias != nil {
		if alias := strings.TrimSpace(*p.Alias); alias != "" {
			return alias, false
		}
	}
	if name := strings.TrimSpace(fallback); name != "" {
		return name, false
	}
	return AnonymousDonorName, true
}

// ToPublicDonation menyusun bentuk publik satu donasi.
//
// Ini SATU-SATUNYA tempat anonimitas diterapkan, supaya aturannya tidak dapat
// terlewat di salah satu endpoint. Donasi anonim menyembunyikan identitas dan
// nominal sekaligus: menampilkan nominal sambil menyembunyikan nama tetap
// membocorkan siapa yang memberi, karena nominal sering kali khas.
//
// Donasi yang identitasnya disembunyikan karena profil privat atau terblokir
// tetap menyembunyikan nominalnya, dengan alasan yang sama.
func ToPublicDonation(d *Donation, donorName string, donorHidden bool, avatarURL *string) *PublicDonation {
	if d == nil {
		return nil
	}

	if d.IsAnonymous {
		donorName = AnonymousDonorName
		donorHidden = true
	}

	view := &PublicDonation{
		PublicID:    d.PublicID,
		DonorName:   donorName,
		DonorHidden: donorHidden,
		CreatedAt:   d.CreatedAt,
	}

	if !donorHidden {
		view.AvatarURL = avatarURL
	}

	// Nominal disembunyikan bila identitasnya disembunyikan, bila donatur
	// mematikan tampilan nominal, atau bila donasinya anonim.
	if !donorHidden && !d.IsAnonymous && d.ShowAmount {
		amount := d.Amount
		view.Amount = &amount
	}

	// Hanya pesan yang sudah disetujui yang muncul di permukaan publik. Pesan
	// donasi anonim tetap boleh tampil: donatur yang menuliskannya sendiri.
	if d.MessageStatus == MessageApproved {
		view.Message = d.Message
	}

	return view
}

// ToAdminDonationView menyusun tampilan donasi untuk antrean pengurus.
//
// Identitas donatur diambil terpisah lewat DonorIdentityResolver dan boleh
// nil: kegagalan mengambil identitas tidak boleh menghilangkan donasi dari
// antrean verifikasi.
func ToAdminDonationView(d *Donation, donor *community.PublicProfile, campaignSlug, campaignTitle string) *AdminDonationView {
	if d == nil {
		return nil
	}
	return &AdminDonationView{
		Donation:      d,
		Donor:         donor,
		CampaignSlug:  campaignSlug,
		CampaignTitle: campaignTitle,
	}
}
