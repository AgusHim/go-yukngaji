package instagram

import (
	"errors"
	"testing"

	"github.com/gin-gonic/gin"
)

// repoPalsu mencatat setiap penulisan supaya tes bisa membuktikan bahwa sync
// yang gagal tidak menyentuh database sama sekali.
type repoPalsu struct {
	disimpan []*Post
}

func (r *repoPalsu) ListPublished(*gin.Context) ([]*Post, error) { return nil, nil }
func (r *repoPalsu) ListAll(*gin.Context) ([]*Post, error)       { return nil, nil }
func (r *repoPalsu) FindByID(*gin.Context, string) (*Post, error) {
	return nil, errors.New("tidak dipakai")
}
func (r *repoPalsu) Create(*gin.Context, *Post) error { return nil }
func (r *repoPalsu) Update(*gin.Context, string, map[string]interface{}) error {
	return nil
}
func (r *repoPalsu) SoftDelete(*gin.Context, string) error { return nil }
func (r *repoPalsu) UpsertSynced(_ *gin.Context, p *Post) (bool, error) {
	r.disimpan = append(r.disimpan, p)
	return true, nil
}

// clientPalsu menggantikan Graph API.
type clientPalsu struct {
	media  []*Media
	tagged []*Media
	// errTagged mensimulasikan /tags yang ditolak, yang tidak boleh
	// menggagalkan seluruh sinkronisasi.
	errTagged error
}

func (c *clientPalsu) Media(*gin.Context) ([]*Media, error)  { return c.media, nil }
func (c *clientPalsu) Tagged(*gin.Context) ([]*Media, error) { return c.tagged, c.errTagged }

func str(s string) *string { return &s }
func num(i int) *int       { return &i }

// Tanpa kredensial, sync harus ditolak dan tidak boleh menulis apa pun.
func TestSyncTanpaKredensialTidakMenulisApaPun(t *testing.T) {
	repo := &repoPalsu{}
	svc := NewService(repo, nil)

	_, err := svc.Sync(newTestContext())
	if !errors.Is(err, ErrSyncDisabled) {
		t.Fatalf("galat = %v, mau ErrSyncDisabled", err)
	}
	if len(repo.disimpan) != 0 {
		t.Fatalf("ada %d baris tertulis padahal sync mati", len(repo.disimpan))
	}
}

// Baris hasil sync selalu belum terbit, supaya sinkronisasi tidak pernah
// menerbitkan apa pun tanpa ditinjau pengurus.
func TestSyncMenyimpanSebagaiBelumTerbit(t *testing.T) {
	repo := &repoPalsu{}
	svc := NewService(repo, &clientPalsu{
		media: []*Media{{
			ID:            "17890000000000001",
			Username:      str("yukngajisolo"),
			Caption:       str("Kajian rutin"),
			MediaType:     str(MediaTypeImage),
			MediaURL:      str("https://scontent.cdninstagram.com/a.jpg"),
			Permalink:     str("https://www.instagram.com/p/abc/"),
			Timestamp:     str("2026-09-28T03:15:00+0000"),
			LikeCount:     num(120),
			CommentsCount: num(16),
		}},
	})

	hasil, err := svc.Sync(newTestContext())
	if err != nil {
		t.Fatalf("sync gagal: %v", err)
	}
	if hasil.Fetched != 1 || hasil.Created != 1 || hasil.Skipped != 0 {
		t.Fatalf("hasil = %+v, mau fetched=1 created=1 skipped=0", hasil)
	}

	post := repo.disimpan[0]
	if post.Source != SourceSync {
		t.Fatalf("source = %q, mau %q", post.Source, SourceSync)
	}
	if post.IsPublished {
		t.Fatal("baris hasil sync terbit sendiri")
	}
	if post.IGMediaID == nil || *post.IGMediaID != "17890000000000001" {
		t.Fatal("ig_media_id tidak tersalin, deduplikasi akan gagal")
	}
	// "+0000" bukan layout RFC3339; tanpa penanganan khusus tanggalnya hilang.
	if post.PostedAt == nil {
		t.Fatal("timestamp Graph API gagal diurai")
	}
	if post.LikeCount == nil || *post.LikeCount != 120 {
		t.Fatal("like_count tidak tersalin")
	}
}

// /tags yang ditolak tidak boleh menggagalkan sync; postingan akun sendiri
// tetap masuk, dan angka like/komentar milik orang lain tetap kosong.
func TestSyncTetapJalanSaatTaggedGagal(t *testing.T) {
	repo := &repoPalsu{}
	svc := NewService(repo, &clientPalsu{
		media: []*Media{{
			ID:       "17890000000000002",
			Username: str("yukngajisolo"),
			MediaURL: str("https://scontent.cdninstagram.com/b.jpg"),
		}},
		errTagged: errors.New("(#10) Application does not have permission"),
	})

	hasil, err := svc.Sync(newTestContext())
	if err != nil {
		t.Fatalf("sync seharusnya tetap sukses: %v", err)
	}
	if hasil.Created != 1 {
		t.Fatalf("created = %d, mau 1", hasil.Created)
	}
}

// Item tanpa id tidak bisa dideduplikasi, jadi harus dilewati — bukan
// disimpan sebagai baris baru setiap kali sync berjalan.
func TestSyncMelewatiItemTanpaID(t *testing.T) {
	repo := &repoPalsu{}
	svc := NewService(repo, &clientPalsu{
		media: []*Media{
			{ID: "", MediaURL: str("https://scontent.cdninstagram.com/c.jpg")},
			{ID: "17890000000000003", MediaURL: str("https://scontent.cdninstagram.com/d.jpg")},
		},
	})

	hasil, err := svc.Sync(newTestContext())
	if err != nil {
		t.Fatalf("sync gagal: %v", err)
	}
	if hasil.Fetched != 2 || hasil.Created != 1 || hasil.Skipped != 1 {
		t.Fatalf("hasil = %+v, mau fetched=2 created=1 skipped=1", hasil)
	}
	if len(repo.disimpan) != 1 {
		t.Fatalf("tersimpan %d baris, mau 1", len(repo.disimpan))
	}
}

// Nama akun disimpan tanpa '@' supaya kartu tidak menampilkannya dua kali.
func TestCreateMembuangAtPadaUsername(t *testing.T) {
	repo := &repoPalsu{}
	svc := NewService(repo, nil)

	post, err := svc.Create(newTestContext(), &CreatePost{
		Username: "@yukngajisolo",
		MediaURL: "https://scontent.cdninstagram.com/e.jpg",
	})
	if err != nil {
		t.Fatalf("create gagal: %v", err)
	}
	if post.Username != "yukngajisolo" {
		t.Fatalf("username = %q, mau tanpa '@'", post.Username)
	}
	if post.Source != SourceManual {
		t.Fatalf("source = %q, mau %q", post.Source, SourceManual)
	}
}

func TestCreateMenolakJumlahNegatif(t *testing.T) {
	svc := NewService(&repoPalsu{}, nil)

	_, err := svc.Create(newTestContext(), &CreatePost{
		Username:  "yukngajisolo",
		MediaURL:  "https://scontent.cdninstagram.com/f.jpg",
		LikeCount: num(-1),
	})
	if err == nil {
		t.Fatal("jumlah suka negatif seharusnya ditolak")
	}
}
