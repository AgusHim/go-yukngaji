// Package shopee menyediakan endpoint operasional ShopeePay/QRIS yang dahulu
// dijalankan oleh server.js. Paket ini sengaja berdiri sendiri: route pembayaran
// tidak bercampur dengan API aplikasi utama maupun autentikasi pengguna.
package shopee

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const qrLifetime = 30 * time.Minute

const (
	transactionListURL   = "https://shopeepay.shopee.co.id/merchant/v1/partner-web/get-transaction-list"
	transactionDetailURL = "https://shopeepay.shopee.co.id/merchant/v1/partner-web/get-transaction-detail"
)

type logEntry struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

type qrPayment struct {
	ID        string    `json:"id"`
	Amount    int64     `json:"amount"`
	QRString  string    `json:"qr_string"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// Service menyimpan state pendek yang dibutuhkan QR dinamis dan token Shopee.
// State ini sengaja hanya hidup di memori, sama seperti server.js terdahulu.
type Service struct {
	mu               sync.RWMutex
	apiKey           string
	shopeeToken      string
	qrisStatic       string
	qrs              map[string]qrPayment
	usedTransactions map[string]time.Time
	logs             []logEntry
	httpClient       *http.Client
	paymentLookup    PaymentLookup
	autoValidation   bool
}

// PaymentLookup membuat endpoint legacy /check-payment membaca klaim
// persisten yang sama dengan worker, bukan membuat klaim kedua di memori.
type PaymentLookup interface {
	PaidTransaction(ctx context.Context, amount int64, start time.Time) (*SuccessfulTransaction, error)
}

func (s *Service) SetPaymentLookup(lookup PaymentLookup) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.paymentLookup = lookup
}

// SetAutoValidation diperbarui worker agar health melaporkan proses yang
// benar-benar berjalan, bukan sekadar keberadaan environment.
func (s *Service) SetAutoValidation(active bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.autoValidation = active
}

func NewFromEnv() *Service {
	return &Service{
		apiKey:           os.Getenv("API_KEY"),
		shopeeToken:      os.Getenv("SHOPEE_TOKEN"),
		qrisStatic:       os.Getenv("QRIS_STATIC"),
		qrs:              make(map[string]qrPayment),
		usedTransactions: make(map[string]time.Time),
		httpClient:       &http.Client{Timeout: 20 * time.Second},
	}
}

func (s *Service) addLog(level, message string) {
	entry := logEntry{Timestamp: time.Now().UTC().Format(time.RFC3339), Level: level, Message: message}
	log.Printf("[shopee][%s] %s", level, message)

	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, entry)
	if len(s.logs) > 100 {
		s.logs = s.logs[len(s.logs)-100:]
	}
}

func (s *Service) authorize(c *gin.Context) bool {
	s.mu.RLock()
	expected := s.apiKey
	s.mu.RUnlock()
	provided := c.GetHeader("X-API-Key")
	if provided == "" {
		provided = c.Query("api_key")
	}
	if expected == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 {
		s.addLog("WARN", "Permintaan ditolak: API key tidak valid")
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"success": false, "error": "Invalid or missing API key"})
		return false
	}
	return true
}

func (s *Service) authMiddleware(c *gin.Context) {
	if s.authorize(c) {
		c.Next()
	}
}

// Register memasang route legacy pada root agar URL integrasi lama tidak berubah.
func Register(router gin.IRouter, service *Service) {
	if service == nil {
		service = NewFromEnv()
	}

	router.GET("/api/health", service.health)
	router.POST("/update-token", service.authMiddleware, service.updateToken)
	router.GET("/token-status", service.authMiddleware, service.tokenStatus)
	router.POST("/create-qris", service.authMiddleware, service.createQRIS)
	router.GET("/qr/:id", service.authMiddleware, service.getQR)
	router.GET("/transactions", service.authMiddleware, service.transactions)
	router.GET("/transactions/all", service.authMiddleware, service.transactionsAll)
	router.POST("/check-payment", service.authMiddleware, service.checkPayment)
	router.GET("/api/logs", service.authMiddleware, service.getLogs)
	router.GET("/", func(c *gin.Context) { c.String(http.StatusOK, "Shopee API Running") })
}

func (s *Service) client() *http.Client {
	if s.httpClient != nil {
		return s.httpClient
	}
	return http.DefaultClient
}

func (s *Service) health(c *gin.Context) {
	s.mu.RLock()
	tokenConfigured := strings.TrimSpace(s.shopeeToken) != ""
	qrisConfigured := strings.TrimSpace(s.qrisStatic) != ""
	autoValidation := s.autoValidation
	s.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{
		"service": "ShopeePay/QRIS", "token_configured": tokenConfigured,
		"qris_configured": qrisConfigured, "auto_validation": autoValidation,
	}})
}

func (s *Service) updateToken(c *gin.Context) {
	var body struct {
		Token string `json:"token"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || strings.TrimSpace(body.Token) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Token is required"})
		return
	}
	s.mu.Lock()
	s.shopeeToken = strings.TrimSpace(body.Token)
	s.mu.Unlock()
	s.addLog("INFO", "Token Shopee diperbarui melalui API")
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "Token updated successfully"}})
}

func (s *Service) tokenStatus(c *gin.Context) {
	s.mu.RLock()
	configured := s.shopeeToken != ""
	s.mu.RUnlock()
	message := "Token is not configured"
	status := "invalid"
	if configured {
		message, status = "Token is configured", "configured"
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"token_status": status, "message": message}})
}

func (s *Service) createQRIS(c *gin.Context) {
	var body struct {
		Amount int64 `json:"amount"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Provide valid amount (positive integer)"})
		return
	}

	s.mu.RLock()
	staticQR := s.qrisStatic
	s.mu.RUnlock()
	if staticQR == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"success": false, "error": "QRIS_STATIC is not configured"})
		return
	}
	qrString, err := dynamicQRIS(staticQR, body.Amount)
	if err != nil {
		s.addLog("ERROR", "Gagal membuat QRIS dinamis: "+err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": "Unable to create dynamic QRIS"})
		return
	}

	now := time.Now().UTC()
	payment := qrPayment{ID: uuid.NewString(), Amount: body.Amount, QRString: qrString, CreatedAt: now, ExpiresAt: now.Add(qrLifetime)}
	s.mu.Lock()
	for id, candidate := range s.qrs {
		if now.After(candidate.ExpiresAt) {
			delete(s.qrs, id)
		}
	}
	s.qrs[payment.ID] = payment
	s.mu.Unlock()
	s.addLog("INFO", "QRIS dinamis dibuat untuk nominal Rp "+strconv.FormatInt(body.Amount, 10))
	c.JSON(http.StatusOK, gin.H{"success": true, "data": payment})
}

func (s *Service) getQR(c *gin.Context) {
	s.mu.RLock()
	payment, ok := s.qrs[c.Param("id")]
	s.mu.RUnlock()
	if !ok || time.Now().UTC().After(payment.ExpiresAt) {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": "QR payment not found or expired"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": payment})
}

func (s *Service) transactions(c *gin.Context) {
	end := time.Now().UTC()
	transactions, _, err := s.listTransactions(c.Request.Context(), s.requestToken(c), end.Add(-72*time.Hour), end, 10, "")
	if err != nil {
		s.transactionError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "total_amount": strconv.Itoa(len(transactions)), "data": gin.H{"transactions": transactions}})
}

func (s *Service) transactionsAll(c *gin.Context) {
	now := time.Now().UTC()
	jakarta, _ := time.LoadLocation("Asia/Jakarta")
	local := now.In(jakarta)
	start := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, jakarta).UTC()
	var all []transaction
	position := ""
	for {
		page, next, err := s.listTransactions(c.Request.Context(), s.requestToken(c), start, now, 100, position)
		if err != nil {
			s.transactionError(c, err)
			return
		}
		all = append(all, page...)
		if len(page) < 100 || next == "" {
			break
		}
		position = next
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "total_amount": strconv.Itoa(len(all)), "data": gin.H{"period": start.In(jakarta).Format("2006-01-02") + " s/d " + now.In(jakarta).Format("2006-01-02"), "total_count": len(all), "transactions": all}})
}

func (s *Service) checkPayment(c *gin.Context) {
	var body struct {
		Amount    int64 `json:"amount"`
		StartTime int64 `json:"startTime"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Provide valid amount (positive integer)"})
		return
	}
	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)
	if body.StartTime > 0 {
		start = time.Unix(body.StartTime, 0).UTC()
	}
	s.mu.RLock()
	lookup := s.paymentLookup
	s.mu.RUnlock()
	if lookup != nil {
		matched, lookupErr := lookup.PaidTransaction(c.Request.Context(), body.Amount, start)
		if lookupErr != nil {
			s.transactionError(c, lookupErr)
			return
		}
		if matched == nil {
			c.JSON(http.StatusOK, gin.H{"success": true, "paid": false})
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true, "paid": true, "transaction": gin.H{
			"transactionId": matched.ID, "amount": matched.Amount, "status": "Success",
			"time": matched.CreateTime.In(time.FixedZone("WIB", 7*60*60)).Format("2006-01-02 15:04:05"), "issuer": "ShopeePay/QRIS",
		}})
		return
	}
	transactions, _, err := s.listTransactions(c.Request.Context(), s.requestToken(c), start, end, 50, "")
	if err != nil {
		s.transactionError(c, err)
		return
	}
	for _, candidate := range transactions {
		if candidate.Status != 3 || candidate.Amount != body.Amount || candidate.CreateTime < start.Unix() || candidate.ID == "" || s.isUsed(candidate.ID) {
			continue
		}
		s.markUsed(candidate.ID)
		issuer := "ShopeePay/QRIS"
		if detail, detailErr := s.transactionDetail(c.Request.Context(), s.requestToken(c), candidate.DisplayIDOrID()); detailErr == nil && detail.Issuer != "" {
			issuer = detail.Issuer
		}
		s.addLog("INFO", "Pembayaran ditemukan: "+candidate.ID)
		c.JSON(http.StatusOK, gin.H{"success": true, "paid": true, "transaction": gin.H{"transactionId": candidate.ID, "amount": candidate.Amount, "status": candidate.StatusText(), "time": candidate.Time(), "issuer": issuer}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "paid": false})
}

func (s *Service) requestToken(c *gin.Context) string {
	if token := strings.TrimSpace(c.GetHeader("X-Shopee-Token")); token != "" {
		return token
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.shopeeToken
}

func (s *Service) isUsed(id string) bool {
	s.mu.RLock()
	_, used := s.usedTransactions[id]
	s.mu.RUnlock()
	return used
}

func (s *Service) markUsed(id string) {
	now := time.Now().UTC()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.usedTransactions == nil {
		s.usedTransactions = make(map[string]time.Time)
	}
	for key, usedAt := range s.usedTransactions {
		if now.Sub(usedAt) > 24*time.Hour {
			delete(s.usedTransactions, key)
		}
	}
	s.usedTransactions[id] = now
}

type transaction struct {
	ID                   string `json:"transactionId"`
	DisplayTransactionID string `json:"displayTransactionId"`
	AmountRaw            any    `json:"amount"`
	Status               int    `json:"status"`
	CreateTime           int64  `json:"createTime"`
	Amount               int64  `json:"-"`
}

// SuccessfulTransaction adalah mutasi berhasil yang dapat dipakai worker
// pembayaran. Worker tetap wajib mengklaim ID-nya secara atomik di database.
type SuccessfulTransaction struct {
	ID         string
	DisplayID  string
	Amount     int64
	CreateTime time.Time
}

// SuccessfulTransactions membaca seluruh halaman mutasi berhasil pada rentang
// waktu yang diminta.
func (s *Service) SuccessfulTransactions(ctx context.Context, start, end time.Time) ([]SuccessfulTransaction, error) {
	if s == nil {
		return nil, fmt.Errorf("Shopee transaction provider is not configured")
	}
	s.mu.RLock()
	token := strings.TrimSpace(s.shopeeToken)
	s.mu.RUnlock()
	if token == "" {
		return nil, fmt.Errorf("Shopee token is not configured")
	}

	position := ""
	result := make([]SuccessfulTransaction, 0)
	for {
		page, next, err := s.listTransactions(ctx, token, start, end, 100, position)
		if err != nil {
			return nil, err
		}
		for _, candidate := range page {
			if candidate.Status != 3 || candidate.ID == "" || candidate.Amount <= 0 {
				continue
			}
			result = append(result, SuccessfulTransaction{
				ID:         candidate.ID,
				DisplayID:  candidate.DisplayIDOrID(),
				Amount:     candidate.Amount,
				CreateTime: time.Unix(candidate.CreateTime, 0).UTC(),
			})
		}
		if len(page) < 100 || next == "" {
			break
		}
		position = next
	}
	return result, nil
}

// Configured menyatakan apakah token mutasi dan QRIS statis tersedia.
func (s *Service) Configured() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return strings.TrimSpace(s.shopeeToken) != "" && strings.TrimSpace(s.qrisStatic) != ""
}

// DynamicQRIS membentuk payload QRIS nominal dinamis dari QRIS statis service.
func (s *Service) DynamicQRIS(amount int64) (string, error) {
	if s == nil || amount <= 0 {
		return "", fmt.Errorf("invalid QRIS amount")
	}
	s.mu.RLock()
	staticQR := s.qrisStatic
	s.mu.RUnlock()
	if strings.TrimSpace(staticQR) == "" {
		return "", fmt.Errorf("QRIS_STATIC is not configured")
	}
	return dynamicQRIS(staticQR, amount)
}

func (t transaction) DisplayIDOrID() string {
	if t.DisplayTransactionID != "" {
		return t.DisplayTransactionID
	}
	return t.ID
}

func (t transaction) StatusText() string {
	return map[int]string{1: "Pending", 2: "Processing", 3: "Success", 4: "Failed", 5: "Cancelled"}[t.Status]
}

func (t transaction) Time() string {
	return time.Unix(t.CreateTime, 0).In(time.FixedZone("WIB", 7*60*60)).Format("2006-01-02 15:04:05")
}

type transactionDetail struct {
	Issuer string `json:"issuer"`
}

func (s *Service) listTransactions(ctx context.Context, token string, start, end time.Time, pageSize int, position string) ([]transaction, string, error) {
	if token == "" {
		return nil, "", fmt.Errorf("Shopee token is not configured")
	}
	var response struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			List         []transaction `json:"list"`
			NextPosition string        `json:"next_position"`
		} `json:"data"`
	}
	body := gin.H{"data": gin.H{"metadata": gin.H{"token": token, "language": "id", "timezone": "Asia/Jakarta"}, "pageSize": pageSize, "filter": gin.H{"startTime": start.Unix(), "endTime": end.Unix(), "serviceList": []int{1, 3}}, "sorter": gin.H{"field": "createTime", "order": "descend"}, "next_position": position}}
	if err := s.postShopee(ctx, transactionListURL, body, &response); err != nil {
		return nil, "", err
	}
	if response.Code != 0 {
		return nil, "", fmt.Errorf("Shopee API error %d: %s", response.Code, response.Msg)
	}
	for index := range response.Data.List {
		response.Data.List[index].Amount = parseAmount(response.Data.List[index].AmountRaw)
	}
	return response.Data.List, response.Data.NextPosition, nil
}

func (s *Service) transactionDetail(ctx context.Context, token, orderSN string) (transactionDetail, error) {
	var response struct {
		Code int               `json:"code"`
		Msg  string            `json:"msg"`
		Data transactionDetail `json:"data"`
	}
	body := gin.H{"data": gin.H{"metadata": gin.H{"token": token, "language": "id", "timezone": "Asia/Jakarta"}, "order_sn": orderSN}}
	if err := s.postShopee(ctx, transactionDetailURL, body, &response); err != nil {
		return transactionDetail{}, err
	}
	if response.Code != 0 {
		return transactionDetail{}, fmt.Errorf("Shopee API error %d: %s", response.Code, response.Msg)
	}
	return response.Data, nil
}

func (s *Service) postShopee(ctx context.Context, endpoint string, payload any, output any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://partner.shopee.co.id")
	request.Header.Set("Referer", "https://partner.shopee.co.id/")
	request.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")
	request.Header.Set("X-Timestamp-Ms", strconv.FormatInt(time.Now().UnixMilli(), 10))
	response, err := s.client().Do(request)
	if err != nil {
		return fmt.Errorf("Shopee request failed: %w", err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Shopee HTTP %d", response.StatusCode)
	}
	return json.Unmarshal(data, output)
}

func (s *Service) transactionError(c *gin.Context, err error) {
	s.addLog("ERROR", "Pengecekan mutasi Shopee gagal: "+err.Error())
	c.JSON(http.StatusBadGateway, gin.H{"success": false, "error": err.Error()})
}

func parseAmount(value any) int64 {
	text := strings.NewReplacer(".", "", ",", "", " ", "").Replace(fmt.Sprint(value))
	amount, _ := strconv.ParseInt(text, 10, 64)
	return amount
}

func (s *Service) getLogs(c *gin.Context) {
	s.mu.RLock()
	logs := append([]logEntry(nil), s.logs...)
	s.mu.RUnlock()
	c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"logs": logs}})
}

// dynamicQRIS mengganti tag nominal (54) lalu menghitung ulang checksum CRC-16
// CCITT-FALSE (tag 63), sesuai format EMV QRIS.
func dynamicQRIS(staticQR string, amount int64) (string, error) {
	base := strings.TrimSpace(staticQR)
	if len(base) < 8 {
		return "", strconv.ErrSyntax
	}
	withoutCRC := base
	if index := strings.LastIndex(base, "6304"); index >= 0 {
		withoutCRC = base[:index]
	}
	amountText := strconv.FormatInt(amount, 10)
	var builder strings.Builder
	for index := 0; index < len(withoutCRC); {
		if index+4 > len(withoutCRC) {
			return "", strconv.ErrSyntax
		}
		tag := withoutCRC[index : index+2]
		length, err := strconv.Atoi(withoutCRC[index+2 : index+4])
		if err != nil || index+4+length > len(withoutCRC) {
			return "", strconv.ErrSyntax
		}
		value := withoutCRC[index+4 : index+4+length]
		index += 4 + length
		if tag == "54" {
			continue
		}
		builder.WriteString(tag)
		builder.WriteString(twoDigits(len(value)))
		builder.WriteString(value)
	}
	result := builder.String() + "54" + twoDigits(len(amountText)) + amountText + "6304"
	checksum := crc16CCITT([]byte(result))
	return result + strings.ToUpper(strconv.FormatUint(uint64(checksum), 16)), nil
}

func twoDigits(value int) string {
	return strconv.FormatInt(int64(value/10), 10) + strconv.FormatInt(int64(value%10), 10)
}

func crc16CCITT(data []byte) uint16 {
	var crc uint16 = 0xFFFF
	for _, value := range data {
		crc ^= uint16(value) << 8
		for bit := 0; bit < 8; bit++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}
