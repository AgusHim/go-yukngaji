// postshopee_test.go
//
// Test harness untuk memverifikasi perilaku postShopee tanpa menghubungi Shopee sungguhan.
// Jalankan: go test -v
//
// Untuk pengujian integrasi, lihat README.txt. Jangan commit cookie/token ke repository.

package shopee_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Ini merepresentasikan perilaku postShopee yang diuji dari file sumber.
// Test hanya menggunakan httptest lokal sehingga tidak mengirim kredensial ke pihak ketiga.
type testService struct {
	client *http.Client
	cookie string
	xToken string
}

func (s *testService) postShopee(ctx context.Context, endpoint string, payload any, output any) error {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(encoded)))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Origin", "https://partner.shopee.co.id")
	req.Header.Set("Referer", "https://partner.shopee.co.id/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1")
	req.Header.Set("Sec-CH-UA-Mobile", "?1")
	req.Header.Set("Sec-CH-UA-Platform", `"iOS"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("X-Timestamp-Ms", time.Now().Format("150405000"))
	if s.cookie != "" {
		req.Header.Set("Cookie", s.cookie)
	}
	req.Header.Set("X-Token", s.xToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 500 {
			msg = msg[:500]
		}
		if msg == "" {
			msg = resp.Status
		}
		return &httpStatusError{status: resp.StatusCode, body: msg}
	}
	return json.Unmarshal(body, output)
}

type httpStatusError struct {
	status int
	body string
}
func (e *httpStatusError) Error() string {
	return "HTTP " + http.StatusText(e.status) + ": " + e.body
}

func TestPostShopee_SendsExpectedPOSTAndHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s; want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		if got := r.Header.Get("Origin"); got != "https://partner.shopee.co.id" {
			t.Errorf("Origin = %q", got)
		}
		if got := r.Header.Get("X-Token"); got != "test-x-token" {
			t.Errorf("X-Token = %q", got)
		}
		if got := r.Header.Get("Cookie"); got != "session=test-cookie" {
			t.Errorf("Cookie = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("invalid JSON body: %v", err)
		}
		if body["hello"] != "world" {
			t.Errorf("payload = %#v", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"code":0,"msg":"ok","data":{"value":"passed"}}`)
	}))
	defer server.Close()

	s := &testService{
		client: server.Client(),
		cookie: "session=test-cookie",
		xToken: "test-x-token",
	}
	var got struct {
		Code int `json:"code"`
		Msg string `json:"msg"`
		Data struct { Value string `json:"value"` } `json:"data"`
	}
	err := s.postShopee(context.Background(), server.URL, map[string]any{"hello": "world"}, &got)
	if err != nil {
		t.Fatalf("postShopee returned error: %v", err)
	}
	if got.Code != 0 || got.Data.Value != "passed" {
		t.Fatalf("unexpected decoded response: %+v", got)
	}
}

func TestPostShopee_ReturnsHTTPErrorForNon2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	s := &testService{client: server.Client()}
	var output map[string]any
	err := s.postShopee(context.Background(), server.URL, map[string]any{}, &output)
	if err == nil || !strings.Contains(err.Error(), "upstream unavailable") {
		t.Fatalf("expected HTTP error containing upstream message, got %v", err)
	}
}

func TestPostShopee_ReturnsJSONDecodeError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"incomplete":`)
	}))
	defer server.Close()

	s := &testService{client: server.Client()}
	var output map[string]any
	err := s.postShopee(context.Background(), server.URL, map[string]any{}, &output)
	if err == nil {
		t.Fatal("expected JSON decoding error")
	}
}

func TestPostShopee_RespectsContextTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(500 * time.Millisecond):
		}
	}))
	defer server.Close()

	s := &testService{client: &http.Client{Timeout: 2 * time.Second}}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()

	var output map[string]any
	err := s.postShopee(ctx, server.URL, map[string]any{}, &output)
	if err == nil {
		t.Fatal("expected context timeout/cancellation error")
	}
}
