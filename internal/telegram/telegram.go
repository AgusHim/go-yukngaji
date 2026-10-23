// Package telegram mengirim notifikasi operasional best-effort ke bot Telegram.
package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

type Notifier struct {
	token  string
	chatID string
	client *http.Client
}

// NewFromEnv mengaktifkan bot bila TELEGRAM_BOT_TOKEN dan TELEGRAM_CHAT_ID
// tersedia. Tanpa keduanya, Notify menjadi no-op sehingga transaksi tak pernah
// terganggu oleh konfigurasi observabilitas yang belum lengkap.
func NewFromEnv() *Notifier {
	return &Notifier{token: strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")), chatID: strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID")), client: &http.Client{Timeout: 8 * time.Second}}
}

func (n *Notifier) Enabled() bool { return n != nil && n.token != "" && n.chatID != "" }

func (n *Notifier) Notify(ctx context.Context, text string) error {
	if !n.Enabled() {
		return nil
	}
	body, err := json.Marshal(map[string]string{"chat_id": n.chatID, "text": text})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.telegram.org/bot"+n.token+"/sendMessage", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := n.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("Telegram sendMessage gagal: %s", res.Status)
	}
	return nil
}
