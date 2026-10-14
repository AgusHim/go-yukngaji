package instagram

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

// newTestContext membuat konteks gin minimal; client hanya memakai
// c.Request.Context(), jadi tidak perlu mesin router.
func newTestContext() *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	return c
}

func TestNewGraphClientMengembalikanNilTanpaKredensial(t *testing.T) {
	cases := []struct {
		nama   string
		token  string
		igID   string
		mauNil bool
	}{
		{"keduanya kosong", "", "", true},
		{"token kosong", "", "17841400000000000", true},
		{"ig id kosong", "EAAGtoken", "", true},
		{"keduanya berisi spasi", "   ", "   ", true},
		{"lengkap", "EAAGtoken", "17841400000000000", false},
	}

	for _, tc := range cases {
		t.Run(tc.nama, func(t *testing.T) {
			got := NewGraphClient(tc.token, tc.igID, "")
			if tc.mauNil && got != nil {
				t.Fatal("client seharusnya nil supaya sinkronisasi mati")
			}
			if !tc.mauNil && got == nil {
				t.Fatal("client seharusnya dibuat")
			}
		})
	}
}

func TestMediaMenguraiBalasanDanMengikutiPaging(t *testing.T) {
	var halaman []string

	// server diisi setelah NewServer mengembalikan; handler-nya baru berjalan
	// saat permintaan datang, jadi server.URL sudah terisi pada saat itu.
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		halaman = append(halaman, r.URL.RawQuery)
		if r.URL.Query().Get("access_token") != "EAAGtoken" {
			t.Errorf("access_token tidak diteruskan: %q", r.URL.RawQuery)
		}

		// Halaman pertama menunjuk ke halaman kedua lewat paging.next.
		if r.URL.Query().Get("after") == "" {
			_, _ = w.Write([]byte(`{
				"data": [{
					"id": "17900000000000001",
					"username": "yukngajisolo",
					"caption": "Kajian pekan ini",
					"media_type": "IMAGE",
					"media_url": "https://scontent.cdninstagram.com/a.jpg",
					"permalink": "https://www.instagram.com/p/AAA/",
					"timestamp": "2026-06-08T15:58:00+0000",
					"like_count": 120,
					"comments_count": 16
				}],
				"paging": {"next": "` + server.URL + `/media?after=cursor&access_token=EAAGtoken"}
			}`))
			return
		}

		_, _ = w.Write([]byte(`{
			"data": [{"id": "17900000000000002", "media_type": "VIDEO", "like_count": 0}],
			"paging": {}
		}`))
	}))
	defer server.Close()

	client := newGraphClient(server.URL, "EAAGtoken", "17841400000000000", "")

	media, err := client.Media(newTestContext())
	if err != nil {
		t.Fatalf("Media gagal: %v", err)
	}
	if len(halaman) != 2 {
		t.Fatalf("server dipanggil %d kali, mau 2 (paging tidak diikuti?)", len(halaman))
	}
	if len(media) != 2 {
		t.Fatalf("jumlah media = %d, mau 2", len(media))
	}

	first := media[0]
	if first.ID != "17900000000000001" {
		t.Errorf("id media pertama = %q", first.ID)
	}
	if first.LikeCount == nil || *first.LikeCount != 120 {
		t.Errorf("like_count = %v, mau 120", first.LikeCount)
	}
	if first.CommentsCount == nil || *first.CommentsCount != 16 {
		t.Errorf("comments_count = %v, mau 16", first.CommentsCount)
	}

	// Angka nol harus tetap terbaca sebagai nol, bukan NULL.
	second := media[1]
	if second.LikeCount == nil || *second.LikeCount != 0 {
		t.Errorf("like_count = %v, mau 0 (bukan nil)", second.LikeCount)
	}
	// Field yang tidak dikirim API harus tetap nil supaya kartu menyembunyikan
	// barisnya, bukan menampilkan angka nol.
	if second.CommentsCount != nil {
		t.Errorf("comments_count = %v, mau nil", second.CommentsCount)
	}
}

func TestTaggedTidakMemintaAngkaLikeDanKomentar(t *testing.T) {
	var fields string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fields = r.URL.Query().Get("fields")
		_, _ = w.Write([]byte(`{"data": [], "paging": {}}`))
	}))
	defer server.Close()

	client := newGraphClient(server.URL, "EAAGtoken", "17841400000000000", "")
	if _, err := client.Tagged(newTestContext()); err != nil {
		t.Fatalf("Tagged gagal: %v", err)
	}

	// Graph API menolak kedua field itu untuk media milik orang lain, jadi
	// permintaannya memang tidak boleh memuatnya.
	if strings.Contains(fields, "like_count") || strings.Contains(fields, "comments_count") {
		t.Errorf("permintaan /tags memuat angka yang tidak didukung: %q", fields)
	}
}

func TestMediaMengembalikanPesanErrorGraph(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error": {"message": "Invalid OAuth access token", "code": 190}}`))
	}))
	defer server.Close()

	client := newGraphClient(server.URL, "kedaluwarsa", "17841400000000000", "")
	_, err := client.Media(newTestContext())
	if err == nil {
		t.Fatal("mau error, dapat nil")
	}
	// Pesan Meta harus ikut terbawa supaya pengurus tahu penyebabnya.
	if !strings.Contains(err.Error(), "Invalid OAuth access token") {
		t.Errorf("pesan error tidak memuat keterangan Graph API: %v", err)
	}
}
