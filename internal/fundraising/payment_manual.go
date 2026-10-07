package fundraising

import "mainyuk/internal/payment"

// NewManualProvider membuat provider pembayaran manual.
//
// Badannya sudah pindah ke internal/payment bersama adapter-nya; yang tersisa
// di sini hanyalah pintu masuk dengan nama lama, supaya titik konstruksi di
// cmd/main.go tidak perlu tahu bahwa seam-nya pernah berpindah.
func NewManualProvider(methods MethodResolver) PaymentProvider {
	return payment.NewManualProvider(methods)
}
