package shopee

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }

func TestCreateQRISRequiresAPIKeyAndReturnsDynamicQR(t *testing.T) {
	gin.SetMode(gin.TestMode)
	service := &Service{
		apiKey:     "test-key",
		qrisStatic: "0002010102115204000053033605802ID5904Test6003abc6304ABCD",
		qrs:        make(map[string]qrPayment),
	}
	router := gin.New()
	Register(router, service)

	request := httptest.NewRequest(http.MethodPost, "/create-qris", bytes.NewBufferString(`{"amount":12500}`))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status without API key = %d, want %d", response.Code, http.StatusUnauthorized)
	}

	request = httptest.NewRequest(http.MethodPost, "/create-qris", bytes.NewBufferString(`{"amount":12500}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", "test-key")
	response = httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}

	var payload struct {
		Success bool      `json:"success"`
		Data    qrPayment `json:"data"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.Success || payload.Data.ID == "" || payload.Data.Amount != 12500 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if len(payload.Data.QRString) < 8 || payload.Data.QRString[len(payload.Data.QRString)-8:len(payload.Data.QRString)-4] != "6304" {
		t.Fatalf("QRIS checksum missing: %q", payload.Data.QRString)
	}
}

func TestDynamicQRISAddsAmountAndValidChecksum(t *testing.T) {
	qr, err := dynamicQRIS("0002010102115204000053033605802ID5904Test6003abc6304ABCD", 12500)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains([]byte(qr), []byte("540512500")) {
		t.Fatalf("nominal tag missing from %q", qr)
	}
	withoutChecksum := qr[:len(qr)-4]
	checksum, err := strconv.ParseUint(qr[len(qr)-4:], 16, 16)
	if err != nil {
		t.Fatalf("invalid checksum: %v", err)
	}
	if got, want := crc16CCITT([]byte(withoutChecksum)), uint16(checksum); got != want {
		t.Fatalf("checksum = %04X, want %04X", checksum, got)
	}
}

func TestCheckPaymentUsesShopeeTransactionProvider(t *testing.T) {
	gin.SetMode(gin.TestMode)
	sawRequestToken := false
	service := &Service{
		apiKey:           "test-key",
		shopeeToken:      "stored-token",
		qrs:              make(map[string]qrPayment),
		usedTransactions: make(map[string]time.Time),
		httpClient: &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			if request.Method != http.MethodPost {
				t.Fatalf("method = %s", request.Method)
			}
			if request.Header.Get("Origin") != "https://partner.shopee.co.id" || request.Header.Get("X-Timestamp-Ms") == "" {
				t.Fatalf("missing Shopee request headers: %#v", request.Header)
			}
			body, err := io.ReadAll(request.Body)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(body), `"token":"request-token"`) {
				sawRequestToken = true
			}
			payload := `{"code":0,"data":{"list":[{"transactionId":"transaction-1","displayTransactionId":"order-1","amount":"10.000","status":3,"createTime":` + strconv.FormatInt(time.Now().Unix(), 10) + `}]}}`
			if request.URL.String() == transactionDetailURL {
				payload = `{"code":0,"data":{"issuer":"ShopeePay"}}`
			}
			return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(payload)), Request: request}, nil
		})},
	}
	router := gin.New()
	Register(router, service)

	request := httptest.NewRequest(http.MethodPost, "/check-payment", strings.NewReader(`{"amount":10000}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", "test-key")
	request.Header.Set("X-Shopee-Token", "request-token")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), `"paid":true`) || !strings.Contains(response.Body.String(), `"issuer":"ShopeePay"`) {
		t.Fatalf("unexpected response: %s", response.Body.String())
	}

	// Mutasi yang sama tidak boleh diklaim lagi dalam 24 jam.
	response = httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/check-payment", strings.NewReader(`{"amount":10000}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-API-Key", "test-key")
	router.ServeHTTP(response, request)
	if !strings.Contains(response.Body.String(), `"paid":false`) {
		t.Fatalf("duplicate transaction accepted: %s", response.Body.String())
	}
	if !sawRequestToken {
		t.Fatal("request-specific X-Shopee-Token was not forwarded")
	}
}
