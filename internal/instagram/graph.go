package instagram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// graphBaseURL adalah host Graph API Instagram. Dipisah sebagai variabel
// supaya pengujian bisa mengarahkannya ke httptest.
var graphBaseURL = "https://graph.instagram.com"

// defaultGraphVersion dipakai bila INSTAGRAM_GRAPH_VERSION kosong.
const defaultGraphVersion = "v21.0"

// maxPages membatasi penelusuran paging.next. Tanpa batas, satu sync bisa
// berjalan tanpa ujung pada akun dengan ribuan postingan.
const maxPages = 5

// mediaFields adalah field untuk postingan milik akun sendiri.
var mediaFields = []string{
	"id", "caption", "media_type", "media_url", "thumbnail_url",
	"permalink", "timestamp", "username", "like_count", "comments_count",
}

// taggedFields sengaja TANPA like_count/comments_count.
//
// Untuk media milik orang lain, Graph API kerap menolak kedua field itu. Karena
// kartu di landing page menyembunyikan baris like/komentar ketika angkanya
// NULL, menghilangkannya di sini justru menghasilkan perilaku yang benar.
var taggedFields = []string{
	"id", "caption", "media_type", "media_url", "thumbnail_url",
	"permalink", "timestamp", "username",
}

type graphClient struct {
	http    *http.Client
	base    string
	token   string
	igID    string
	version string
}

// NewGraphClient membuat client Graph API.
//
// Mengembalikan nil bila token atau IG user id belum diisi. Pemanggil wajib
// memperlakukan nil sebagai "sinkronisasi mati" — bukan sebagai kesalahan.
func NewGraphClient(token, igUserID, version string) Client {
	token = strings.TrimSpace(token)
	igUserID = strings.TrimSpace(igUserID)
	if token == "" || igUserID == "" {
		return nil
	}
	return newGraphClient(graphBaseURL, token, igUserID, version)
}

func newGraphClient(base, token, igUserID, version string) Client {
	version = strings.TrimSpace(version)
	if version == "" {
		version = defaultGraphVersion
	}
	return &graphClient{
		http:    &http.Client{Timeout: 10 * time.Second},
		base:    strings.TrimRight(base, "/"),
		token:   token,
		igID:    igUserID,
		version: version,
	}
}

// Media mengambil postingan milik akun sendiri.
func (g *graphClient) Media(c *gin.Context) ([]*Media, error) {
	return g.fetchEdge(c, "media", mediaFields)
}

// Tagged mengambil postingan orang lain yang menandai akun ini.
func (g *graphClient) Tagged(c *gin.Context) ([]*Media, error) {
	return g.fetchEdge(c, "tags", taggedFields)
}

func (g *graphClient) fetchEdge(c *gin.Context, edge string, fields []string) ([]*Media, error) {
	ctx := c.Request.Context()

	next := fmt.Sprintf(
		"%s/%s/%s/%s?fields=%s&limit=50&access_token=%s",
		g.base,
		url.PathEscape(g.version),
		url.PathEscape(g.igID),
		url.PathEscape(edge),
		url.QueryEscape(strings.Join(fields, ",")),
		url.QueryEscape(g.token),
	)

	all := []*Media{}
	for page := 0; page < maxPages && next != ""; page++ {
		body, err := g.get(ctx, next)
		if err != nil {
			return nil, err
		}

		parsed := graphResponse{}
		if err := json.Unmarshal(body, &parsed); err != nil {
			return nil, fmt.Errorf("%w: balasan Graph API tidak terbaca", ErrSyncFailed)
		}
		if parsed.Error != nil && parsed.Error.Message != "" {
			return nil, fmt.Errorf("%w: %s", ErrSyncFailed, parsed.Error.Message)
		}

		all = append(all, parsed.Data...)
		next = parsed.Paging.Next
	}
	return all, nil
}

func (g *graphClient) get(ctx context.Context, target string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyncFailed, err)
	}

	res, err := g.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyncFailed, err)
	}
	defer func() { _ = res.Body.Close() }()

	// Batasi pembacaan supaya balasan yang membengkak tidak menghabiskan
	// memori proses.
	body, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSyncFailed, err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		// Badan balasan Graph API memuat error.message yang berguna bagi
		// pengurus, jadi ikut dibaca sebelum dilempar.
		parsed := graphResponse{}
		if json.Unmarshal(body, &parsed) == nil && parsed.Error != nil && parsed.Error.Message != "" {
			return nil, fmt.Errorf("%w: %s", ErrSyncFailed, parsed.Error.Message)
		}
		return nil, fmt.Errorf("%w: HTTP %d dari Graph API", ErrSyncFailed, res.StatusCode)
	}

	return body, nil
}

type graphResponse struct {
	Data   []*Media `json:"data"`
	Paging struct {
		Next string `json:"next"`
	} `json:"paging"`
	Error *graphError `json:"error"`
}

type graphError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    int    `json:"code"`
}
