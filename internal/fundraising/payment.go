package fundraising

import "mainyuk/internal/payment"

// Sejak Fase 4, seam pembayaran tinggal di internal/payment supaya donasi dan
// merchandise memakai adapter yang sama. Berkas ini hanya menyimpan aliasnya:
// seluruh call site, bentuk JSON, dan nama yang sudah beredar di paket ini
// tetap sama persis, sehingga pemindahan itu tidak mengubah perilaku Fase 2.
//
// Tipe alias (bukan tipe baru) dipakai dengan sengaja — donasi yang dikirim
// lewat JSON tetap berbentuk sama, dan `errors.Is` terhadap
// ErrWebhookUnsupported tetap bekerja karena variabelnya menunjuk galat yang
// sama.

type PaymentInstruction = payment.Instruction

type Charge = payment.Charge

type WebhookEvent = payment.WebhookEvent

type PaymentProvider = payment.Provider

// ErrWebhookUnsupported adalah galat yang dikembalikan provider manual.
var ErrWebhookUnsupported = payment.ErrWebhookUnsupported

// VerifyManualPayment memeriksa bukti transfer yang diserahkan pengurus.
//
// Nama lama dipertahankan supaya tidak ada pemanggil yang perlu diubah;
// aturannya sendiri tinggal di internal/payment bersama adapter-nya.
func VerifyManualPayment(expected, received int, reference string) error {
	return payment.VerifyManual(expected, received, reference)
}
