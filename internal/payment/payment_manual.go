package payment

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// manualProvider adalah satu-satunya provider yang dipasang saat ini.
//
// Uang berpindah lewat transfer bank/e-wallet biasa dan diverifikasi pengurus.
// Provider ini tidak menghubungi jaringan mana pun; ia hanya menyusun instruksi
// transfer dari metode pembayaran yang sudah dikelola admin.
type manualProvider struct {
	methods MethodResolver
}

// NewManualProvider membuat provider pembayaran manual.
func NewManualProvider(methods MethodResolver) Provider {
	return &manualProvider{methods: methods}
}

func (p *manualProvider) Name() string {
	return "manual"
}

// CreateCharge menyusun instruksi transfer untuk satu transaksi.
//
// Transaksi tanpa metode pembayaran tetap menghasilkan charge yang sah — hanya
// tanpa instruksi — supaya pembayar dapat memilih metode belakangan.
// Kegagalan mengambil metode TIDAK ditelan: pembayar harus tahu bahwa instruksi
// transfer belum bisa ditampilkan, bukan menerima halaman kosong yang tampak
// normal.
func (p *manualProvider) CreateCharge(c *gin.Context, req ChargeRequest) (*Charge, error) {
	charge := &Charge{
		Provider: p.Name(),
		Amount:   req.Amount,
	}

	if req.PaymentMethodID == nil || strings.TrimSpace(*req.PaymentMethodID) == "" {
		return charge, nil
	}
	if p.methods == nil {
		return charge, nil
	}

	method, err := p.methods.Show(c, *req.PaymentMethodID)
	if err != nil {
		return nil, err
	}
	if method == nil {
		return charge, nil
	}

	charge.Instructions = []Instruction{{
		MethodID:      method.ID,
		Type:          method.Type,
		Name:          method.Name,
		AccountName:   method.AccountName,
		AccountNumber: method.AccountNumber,
		ImageURL:      imageURL(method.ImageUrl),
	}}
	return charge, nil
}

// ParseWebhook selalu gagal: tidak ada pihak ketiga yang memanggil balik pada
// alur manual. Verifikasi dilakukan pengurus lewat aksi konfirmasi masing-masing
// domain.
func (p *manualProvider) ParseWebhook(c *gin.Context, body []byte, headers http.Header) (*WebhookEvent, error) {
	return nil, ErrWebhookUnsupported
}

// VerifyManual memeriksa bukti transfer yang diserahkan pengurus.
//
// Dua hal diperiksa, dan keduanya wajib: nominal yang masuk harus sama persis
// dengan nominal tagihan (kelebihan sekecil apa pun tidak boleh diam-diam
// dianggap lunas), dan referensi transfer harus ada supaya baris tagihan dapat
// ditelusuri kembali ke mutasi rekening. Nominal diperiksa di sini, bukan di
// handler, supaya aturannya tetap berlaku walau ada jalur masuk lain kelak.
func VerifyManual(expected, received int, reference string) error {
	if received != expected {
		return invalid("nominal transfer (%d) tidak sama dengan nominal tagihan (%d)", received, expected)
	}
	if strings.TrimSpace(reference) == "" {
		return invalid("referensi transfer wajib diisi")
	}
	return nil
}

// imageURL mengubah string kosong menjadi nil, supaya klien dapat membedakan
// "tidak ada gambar" dari "ada gambar dengan url kosong".
func imageURL(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
