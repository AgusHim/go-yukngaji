package thread

import (
	"strings"
	"testing"

	"mainyuk/internal/community"
)

// publicProfile adalah profil yang lolos semua pengaman privasi: publik,
// tidak terblokir, dan sudah punya alias.
func publicProfile() *community.Profile {
	return &community.Profile{
		UserID:            "u-1",
		PublicID:          "p-1",
		Alias:             strPtr("Ranger Senja"),
		ProfileVisibility: community.VisibilityPublic,
	}
}

// baseContext adalah keadaan yang seharusnya boleh dibagikan; tiap kasus uji
// mengubah satu hal saja darinya.
func baseContext() AutoPostContext {
	return AutoPostContext{
		Activity:  SourceDonation,
		Prefs:     nil,
		Profile:   publicProfile(),
		SourceRef: "d-1",
		Title:     "Bantu Sekolah Desa",
	}
}

func TestShouldAutoPost(t *testing.T) {
	allOn := &SharePrefs{
		ShareEventRegistration: true,
		ShareDonation:          true,
		ShareMission:           true,
	}

	cases := []struct {
		name string
		// mutate mengubah salinan baseContext; nil berarti tidak diubah.
		mutate func(*AutoPostContext)
		want   bool
	}{
		{
			name:   "prefs nil berarti boleh (default opt-out)",
			mutate: nil,
			want:   true,
		},
		{
			name:   "semua preferensi menyala",
			mutate: func(in *AutoPostContext) { in.Prefs = allOn },
			want:   true,
		},
		{
			name: "preferensi donasi dimatikan",
			mutate: func(in *AutoPostContext) {
				in.Prefs = &SharePrefs{ShareEventRegistration: true, ShareDonation: false, ShareMission: true}
			},
			want: false,
		},
		{
			name: "preferensi event dimatikan tidak memengaruhi donasi",
			mutate: func(in *AutoPostContext) {
				in.Prefs = &SharePrefs{ShareEventRegistration: false, ShareDonation: true, ShareMission: true}
			},
			want: true,
		},
		{
			name:   "donasi anonim tidak pernah dibagikan",
			mutate: func(in *AutoPostContext) { in.IsAnonymous = true },
			want:   false,
		},
		{
			name: "profil privat tidak pernah dibagikan",
			mutate: func(in *AutoPostContext) {
				in.Profile.ProfileVisibility = community.VisibilityPrivate
			},
			want: false,
		},
		{
			name:   "profil members tetap dibagikan",
			mutate: func(in *AutoPostContext) { in.Profile.ProfileVisibility = community.VisibilityMembers },
			want:   true,
		},
		{
			name:   "akun terblokir tidak pernah dibagikan",
			mutate: func(in *AutoPostContext) { in.Profile.IsBlocked = true },
			want:   false,
		},
		{
			name:   "profil belum ada",
			mutate: func(in *AutoPostContext) { in.Profile = nil },
			want:   false,
		},
		{
			name:   "rujukan sumber kosong",
			mutate: func(in *AutoPostContext) { in.SourceRef = "" },
			want:   false,
		},
		{
			name:   "rujukan sumber hanya spasi",
			mutate: func(in *AutoPostContext) { in.SourceRef = "   " },
			want:   false,
		},
		{
			name:   "judul sumber kosong",
			mutate: func(in *AutoPostContext) { in.Title = "" },
			want:   false,
		},
		{
			name:   "judul sumber hanya spasi",
			mutate: func(in *AutoPostContext) { in.Title = "  \n " },
			want:   false,
		},
		{
			name:   "aktivitas tak dikenal",
			mutate: func(in *AutoPostContext) { in.Activity = "sesuatu" },
			want:   false,
		},
		{
			name:   "aktivitas kosong",
			mutate: func(in *AutoPostContext) { in.Activity = "" },
			want:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := baseContext()
			if tc.mutate != nil {
				tc.mutate(&in)
			}
			if got := ShouldAutoPost(in); got != tc.want {
				t.Fatalf("ShouldAutoPost() = %v, mau %v (konteks: %+v)", got, tc.want, in)
			}
		})
	}
}

func TestResolveSharePrefs(t *testing.T) {
	activities := []string{SourceEventRegistration, SourceDonation, SourceMission}

	t.Run("prefs nil berarti semua aktivitas dikenal dibagikan", func(t *testing.T) {
		for _, activity := range activities {
			if !ResolveSharePrefs(nil, activity) {
				t.Fatalf("%q seharusnya dibagikan saat prefs nil", activity)
			}
		}
	})

	t.Run("prefs nil tidak membuka aktivitas tak dikenal", func(t *testing.T) {
		if ResolveSharePrefs(nil, "sesuatu") {
			t.Fatal("aktivitas tak dikenal seharusnya tidak dibagikan")
		}
	})

	t.Run("aktivitas tak dikenal ditolak walau prefs ada", func(t *testing.T) {
		prefs := &SharePrefs{ShareEventRegistration: true, ShareDonation: true, ShareMission: true}
		if ResolveSharePrefs(prefs, "sesuatu") {
			t.Fatal("aktivitas tak dikenal seharusnya tidak dibagikan")
		}
	})

	t.Run("tiap field dipetakan ke aktivitasnya", func(t *testing.T) {
		prefs := &SharePrefs{ShareEventRegistration: false, ShareDonation: true, ShareMission: false}
		if ResolveSharePrefs(prefs, SourceEventRegistration) {
			t.Fatal("event registration seharusnya mati")
		}
		if !ResolveSharePrefs(prefs, SourceDonation) {
			t.Fatal("donasi seharusnya nyala")
		}
		if ResolveSharePrefs(prefs, SourceMission) {
			t.Fatal("misi seharusnya mati")
		}
	})
}

func TestIsKnownActivity(t *testing.T) {
	for _, activity := range []string{SourceEventRegistration, SourceDonation, SourceMission} {
		if !IsKnownActivity(activity) {
			t.Fatalf("%q seharusnya dikenal", activity)
		}
	}
	if IsKnownActivity("") || IsKnownActivity("sesuatu") {
		t.Fatal("aktivitas tak dikenal seharusnya ditolak")
	}
}

func TestAutoPostTitle(t *testing.T) {
	cases := []struct {
		name     string
		activity string
		source   string
		want     string
	}{
		{"pendaftaran event", SourceEventRegistration, "Bakti Sosial Desa", "Mendaftar Bakti Sosial Desa"},
		{"donasi", SourceDonation, "Bantu Sekolah Desa", "Berbagi kebaikan di Bantu Sekolah Desa"},
		{"misi", SourceMission, "Kumpulkan 5 Sampah", "Menyelesaikan misi Kumpulkan 5 Sampah"},
		{"aktivitas tak dikenal", "sesuatu", "Apa Saja", ""},
		{"judul sumber kosong", SourceDonation, "", "Berbagi kebaikan di"},
		{"judul sumber dirapikan", SourceDonation, "  Bantu\nSekolah  ", "Berbagi kebaikan di Bantu Sekolah"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := AutoPostTitle(tc.activity, tc.source); got != tc.want {
				t.Fatalf("AutoPostTitle() = %q, mau %q", got, tc.want)
			}
		})
	}

	t.Run("judul sumber sangat panjang dipotong", func(t *testing.T) {
		title := AutoPostTitle(SourceDonation, strings.Repeat("kata ", 100))
		if n := len([]rune(title)); n > ThreadTitleMax {
			t.Fatalf("panjang judul %d, melebihi batas %d", n, ThreadTitleMax)
		}
		if !strings.HasSuffix(title, "…") {
			t.Fatalf("judul terpotong seharusnya diakhiri elipsis: %q", title)
		}
	})

	t.Run("judul sumber dengan karakter kendali dibersihkan", func(t *testing.T) {
		title := AutoPostTitle(SourceDonation, "Bantu\x00Sekolah\u200bDesa")
		if strings.ContainsAny(title, "\x00\u200b") {
			t.Fatalf("judul masih memuat karakter kendali: %q", title)
		}
	})

	t.Run("judul hasil selalu lolos validasi konten", func(t *testing.T) {
		for _, activity := range []string{SourceEventRegistration, SourceDonation, SourceMission} {
			title := AutoPostTitle(activity, "Bakti Sosial Desa")
			if err := CheckThreadContent(title, AutoPostBody(activity, "Bakti Sosial Desa")); err != nil {
				t.Fatalf("%s: judul/badan hasil auto-post tidak lolos validasi: %v", activity, err)
			}
		}
	})
}

func TestAutoPostBody(t *testing.T) {
	t.Run("tiap aktivitas menghasilkan badan yang berbeda", func(t *testing.T) {
		seen := map[string]bool{}
		for _, activity := range []string{SourceEventRegistration, SourceDonation, SourceMission} {
			body := AutoPostBody(activity, "Bakti Sosial Desa")
			if body == "" {
				t.Fatalf("%s: badan kosong", activity)
			}
			if !strings.Contains(body, "Bakti Sosial Desa") {
				t.Fatalf("%s: badan tidak menyebut judul sumber: %q", activity, body)
			}
			if seen[body] {
				t.Fatalf("%s: badan sama dengan aktivitas lain: %q", activity, body)
			}
			seen[body] = true
		}
	})

	t.Run("aktivitas tak dikenal menghasilkan badan kosong", func(t *testing.T) {
		if body := AutoPostBody("sesuatu", "Apa Saja"); body != "" {
			t.Fatalf("badan = %q, mau kosong", body)
		}
	})

	t.Run("judul sumber kosong menghasilkan badan kosong", func(t *testing.T) {
		if body := AutoPostBody(SourceDonation, "   "); body != "" {
			t.Fatalf("badan = %q, mau kosong", body)
		}
	})

	t.Run("nominal tidak pernah muncul di badan donasi", func(t *testing.T) {
		body := AutoPostBody(SourceDonation, "Bantu Sekolah Desa")
		if strings.ContainsAny(body, "0123456789") {
			t.Fatalf("badan donasi memuat angka: %q", body)
		}
		if strings.Contains(strings.ToLower(body), "rp") {
			t.Fatalf("badan donasi menyebut rupiah: %q", body)
		}
	})
}
