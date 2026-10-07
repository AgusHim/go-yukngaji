package payment

import (
	"errors"
	"net/http"
	"testing"

	"mainyuk/internal/apperr"
	"mainyuk/internal/payment_method"

	"github.com/gin-gonic/gin"
)

// fakeMethods adalah MethodResolver palsu untuk menguji provider manual tanpa
// database. Test ini tetap memakai stdlib testing, tanpa mock framework.
type fakeMethods struct {
	method *payment_method.PaymentMethod
	err    error
	calls  int
}

func (f *fakeMethods) Show(c *gin.Context, id string) (*payment_method.PaymentMethod, error) {
	f.calls++
	return f.method, f.err
}

func TestManualProviderName(t *testing.T) {
	provider := NewManualProvider(&fakeMethods{})
	if provider.Name() != "manual" {
		t.Errorf("Name() = %q, ingin \"manual\"", provider.Name())
	}
}

func TestManualProviderCreateCharge(t *testing.T) {
	methodID := "pm-bca"
	ginCtx := &gin.Context{}

	t.Run("nominal selalu tersalin", func(t *testing.T) {
		provider := NewManualProvider(&fakeMethods{})
		charge, err := provider.CreateCharge(ginCtx, ChargeRequest{Amount: 75_000})
		if err != nil {
			t.Fatalf("error tak terduga: %v", err)
		}
		if charge.Amount != 75_000 {
			t.Errorf("nominal = %d, ingin 75000", charge.Amount)
		}
		if charge.Provider != "manual" {
			t.Errorf("provider = %q, ingin \"manual\"", charge.Provider)
		}
		if charge.ExternalID != nil {
			t.Errorf("external id = %v, ingin nil untuk provider manual", *charge.ExternalID)
		}
	})

	t.Run("tanpa metode pembayaran tidak ada instruksi", func(t *testing.T) {
		methods := &fakeMethods{method: &payment_method.PaymentMethod{ID: methodID}}
		provider := NewManualProvider(methods)

		charge, err := provider.CreateCharge(ginCtx, ChargeRequest{Amount: 50_000})
		if err != nil {
			t.Fatalf("error tak terduga: %v", err)
		}
		if len(charge.Instructions) != 0 {
			t.Errorf("instruksi = %d, ingin 0", len(charge.Instructions))
		}
		if methods.calls != 0 {
			t.Errorf("resolver dipanggil %d kali, ingin 0", methods.calls)
		}
	})

	t.Run("metode kosong diperlakukan sebagai tanpa metode", func(t *testing.T) {
		empty := ""
		methods := &fakeMethods{method: &payment_method.PaymentMethod{ID: methodID}}
		provider := NewManualProvider(methods)

		charge, err := provider.CreateCharge(ginCtx, ChargeRequest{Amount: 50_000, PaymentMethodID: &empty})
		if err != nil {
			t.Fatalf("error tak terduga: %v", err)
		}
		if len(charge.Instructions) != 0 {
			t.Errorf("instruksi = %d, ingin 0", len(charge.Instructions))
		}
	})

	t.Run("dengan metode menyusun satu instruksi", func(t *testing.T) {
		methods := &fakeMethods{method: &payment_method.PaymentMethod{
			ID:            methodID,
			Type:          "BANK",
			Name:          "BCA",
			AccountName:   "Yayasan YukNgaji",
			AccountNumber: "1234567890",
			ImageUrl:      "https://example.test/bca.png",
		}}
		provider := NewManualProvider(methods)

		charge, err := provider.CreateCharge(ginCtx, ChargeRequest{Amount: 25_000, PaymentMethodID: &methodID})
		if err != nil {
			t.Fatalf("error tak terduga: %v", err)
		}
		if len(charge.Instructions) != 1 {
			t.Fatalf("instruksi = %d, ingin 1", len(charge.Instructions))
		}
		got := charge.Instructions[0]
		if got.MethodID != methodID || got.AccountNumber != "1234567890" || got.AccountName != "Yayasan YukNgaji" {
			t.Errorf("instruksi tidak sesuai: %+v", got)
		}
		if got.ImageURL == nil || *got.ImageURL != "https://example.test/bca.png" {
			t.Errorf("image url = %v, ingin terisi", got.ImageURL)
		}
	})

	t.Run("gambar kosong menjadi nil", func(t *testing.T) {
		methods := &fakeMethods{method: &payment_method.PaymentMethod{ID: methodID, ImageUrl: "   "}}
		provider := NewManualProvider(methods)

		charge, err := provider.CreateCharge(ginCtx, ChargeRequest{Amount: 25_000, PaymentMethodID: &methodID})
		if err != nil {
			t.Fatalf("error tak terduga: %v", err)
		}
		if charge.Instructions[0].ImageURL != nil {
			t.Errorf("image url = %v, ingin nil", *charge.Instructions[0].ImageURL)
		}
	})

	t.Run("kegagalan resolver diteruskan, bukan ditelan", func(t *testing.T) {
		sentinel := errors.New("db mati")
		provider := NewManualProvider(&fakeMethods{err: sentinel})

		if _, err := provider.CreateCharge(ginCtx, ChargeRequest{Amount: 25_000, PaymentMethodID: &methodID}); !errors.Is(err, sentinel) {
			t.Errorf("error = %v, ingin %v", err, sentinel)
		}
	})
}

func TestManualProviderParseWebhook(t *testing.T) {
	provider := NewManualProvider(&fakeMethods{})
	if _, err := provider.ParseWebhook(&gin.Context{}, nil, http.Header{}); !errors.Is(err, ErrWebhookUnsupported) {
		t.Errorf("error = %v, ingin ErrWebhookUnsupported", err)
	}
}

func TestVerifyManual(t *testing.T) {
	cases := []struct {
		name      string
		expected  int
		received  int
		reference string
		wantErr   bool
	}{
		{
			name:      "nominal sama dan referensi terisi",
			expected:  50_000,
			received:  50_000,
			reference: "TRF-2026-0001",
			wantErr:   false,
		},
		{
			name:      "nominal kurang",
			expected:  50_000,
			received:  49_999,
			reference: "TRF-2026-0001",
			wantErr:   true,
		},
		{
			name:      "nominal lebih",
			expected:  50_000,
			received:  50_001,
			reference: "TRF-2026-0001",
			wantErr:   true,
		},
		{
			name:      "referensi kosong",
			expected:  50_000,
			received:  50_000,
			reference: "",
			wantErr:   true,
		},
		{
			name:      "referensi hanya spasi",
			expected:  50_000,
			received:  50_000,
			reference: "   ",
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := VerifyManual(tc.expected, tc.received, tc.reference)
			if tc.wantErr && err == nil {
				t.Fatal("error = nil, ingin galat validasi")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("error = %v, ingin nil", err)
			}
			if err != nil && !apperr.IsValidation(err) {
				t.Errorf("error = %v, ingin memenuhi Validation", err)
			}
		})
	}
}
