package payment

import (
	"net/http"
	"strings"
	"time"

	"mainyuk/internal/shopee_qris"

	"github.com/gin-gonic/gin"
)

// shopeeQRISProvider mengalihkan metode SHOPEE_QRIS ke charge persisten dan
// mempertahankan metode lain pada alur manual yang sudah ada.
type shopeeQRISProvider struct {
	methods MethodResolver
	qris    *shopee_qris.Service
	manual  Provider
}

func NewShopeeQRISProvider(methods MethodResolver, qris *shopee_qris.Service) Provider {
	return &shopeeQRISProvider{methods: methods, qris: qris, manual: NewManualProvider(methods)}
}

func (p *shopeeQRISProvider) Name() string { return "shopee_qris" }

func (p *shopeeQRISProvider) CreateCharge(c *gin.Context, req ChargeRequest) (*Charge, error) {
	if req.PaymentMethodID == nil || strings.TrimSpace(*req.PaymentMethodID) == "" || p.methods == nil {
		return p.manual.CreateCharge(c, req)
	}
	method, err := p.methods.Show(c, *req.PaymentMethodID)
	if err != nil {
		return nil, err
	}
	if method == nil || !strings.EqualFold(strings.TrimSpace(method.Code), "SHOPEE_QRIS") {
		return p.manual.CreateCharge(c, req)
	}
	if p.qris == nil {
		return nil, ErrProviderUnavailable
	}

	expiresAt := time.Time{}
	if req.ExpiresAt != nil {
		expiresAt = *req.ExpiresAt
	}
	stored, err := p.qris.CreateCharge(c, req.EntityType, req.EntityID, int64(req.Amount), expiresAt)
	if err != nil {
		return nil, err
	}
	externalID := stored.ID
	return &Charge{
		Provider: p.Name(), ExternalID: &externalID,
		Amount: int(stored.PayableAmount), BaseAmount: int(stored.BaseAmount),
		UniqueCode: stored.UniqueCode, QRISPayload: stored.QRISPayload,
		PaymentStatus: stored.Status, ExpiresAt: &stored.ExpiresAt,
		Instructions: []Instruction{{
			MethodID: method.ID, Type: method.Type, Name: method.Name,
			AccountName: method.AccountName, AccountNumber: method.AccountNumber,
			ImageURL: imageURL(method.ImageUrl),
		}},
	}, nil
}

func (p *shopeeQRISProvider) LookupCharge(c *gin.Context, entityType, entityID string) (*Charge, error) {
	if p.qris == nil {
		return nil, nil
	}
	stored, err := p.qris.LookupCharge(c, entityType, entityID)
	if err != nil || stored == nil {
		return nil, err
	}
	externalID := stored.ID
	return &Charge{
		Provider: p.Name(), ExternalID: &externalID,
		Amount: int(stored.PayableAmount), BaseAmount: int(stored.BaseAmount),
		UniqueCode: stored.UniqueCode, QRISPayload: stored.QRISPayload,
		PaymentStatus: stored.Status, ExpiresAt: &stored.ExpiresAt,
	}, nil
}

func (p *shopeeQRISProvider) ParseWebhook(c *gin.Context, body []byte, headers http.Header) (*WebhookEvent, error) {
	return nil, ErrWebhookUnsupported
}
