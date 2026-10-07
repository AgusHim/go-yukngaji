package thread

import (
	"testing"

	"mainyuk/internal/community"
)

func strPtr(s string) *string { return &s }

func TestThreadAuthorName(t *testing.T) {
	cases := []struct {
		name    string
		profile *community.Profile
		want    string
	}{
		{
			name:    "profil nil",
			profile: nil,
			want:    AnonymousAuthorName,
		},
		{
			name:    "alias kosong",
			profile: &community.Profile{Alias: strPtr("")},
			want:    AnonymousAuthorName,
		},
		{
			name:    "alias berisi spasi saja",
			profile: &community.Profile{Alias: strPtr("   \t ")},
			want:    AnonymousAuthorName,
		},
		{
			name:    "alias belum diisi sama sekali",
			profile: &community.Profile{},
			want:    AnonymousAuthorName,
		},
		{
			name:    "alias terisi",
			profile: &community.Profile{Alias: strPtr("Ranger Senja")},
			want:    "Ranger Senja",
		},
		{
			name:    "alias terisi dengan spasi tepi",
			profile: &community.Profile{Alias: strPtr("  Ranger Senja  ")},
			want:    "Ranger Senja",
		},
		{
			name: "akun terblokir",
			profile: &community.Profile{
				Alias:     strPtr("Ranger Senja"),
				IsBlocked: true,
			},
			want: AnonymousAuthorName,
		},
		{
			name: "profil privat",
			profile: &community.Profile{
				Alias:             strPtr("Ranger Senja"),
				ProfileVisibility: community.VisibilityPrivate,
			},
			want: AnonymousAuthorName,
		},
		{
			name: "profil members tetap tampil",
			profile: &community.Profile{
				Alias:             strPtr("Ranger Senja"),
				ProfileVisibility: community.VisibilityMembers,
			},
			want: "Ranger Senja",
		},
		{
			name: "profil publik tetap tampil",
			profile: &community.Profile{
				Alias:             strPtr("Ranger Senja"),
				ProfileVisibility: community.VisibilityPublic,
			},
			want: "Ranger Senja",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ThreadAuthorName(tc.profile); got != tc.want {
				t.Fatalf("ThreadAuthorName() = %q, mau %q", got, tc.want)
			}
		})
	}
}

func TestToThreadAuthor(t *testing.T) {
	avatar := "https://contoh.id/a.png"

	t.Run("profil nil menghasilkan penulis anonim, bukan nil", func(t *testing.T) {
		author := ToThreadAuthor(nil)
		if author == nil {
			t.Fatal("ToThreadAuthor(nil) mengembalikan nil; seharusnya penulis anonim")
		}
		if author.Alias != AnonymousAuthorName {
			t.Fatalf("Alias = %q, mau %q", author.Alias, AnonymousAuthorName)
		}
		if author.PublicID != "" {
			t.Fatalf("PublicID = %q, mau kosong", author.PublicID)
		}
		if author.AvatarURL != nil {
			t.Fatal("AvatarURL seharusnya nil saat penulisnya anonim")
		}
	})

	t.Run("profil privat menyembunyikan avatar", func(t *testing.T) {
		author := ToThreadAuthor(&community.Profile{
			PublicID:          "abc123",
			Alias:             strPtr("Ranger Senja"),
			AvatarURL:         &avatar,
			ProfileVisibility: community.VisibilityPrivate,
		})
		if author.Alias != AnonymousAuthorName {
			t.Fatalf("Alias = %q, mau %q", author.Alias, AnonymousAuthorName)
		}
		if author.AvatarURL != nil {
			t.Fatal("AvatarURL seharusnya nil saat penulisnya dianonimkan")
		}
		if author.PublicID != "abc123" {
			t.Fatalf("PublicID = %q, mau abc123", author.PublicID)
		}
	})

	t.Run("akun terblokir menyembunyikan avatar", func(t *testing.T) {
		author := ToThreadAuthor(&community.Profile{
			PublicID:  "abc123",
			Alias:     strPtr("Ranger Senja"),
			AvatarURL: &avatar,
			IsBlocked: true,
		})
		if author.Alias != AnonymousAuthorName || author.AvatarURL != nil {
			t.Fatalf("penulis terblokir masih terbuka: %+v", author)
		}
	})

	t.Run("alias kosong menyembunyikan avatar", func(t *testing.T) {
		author := ToThreadAuthor(&community.Profile{
			PublicID:  "abc123",
			AvatarURL: &avatar,
		})
		if author.Alias != AnonymousAuthorName || author.AvatarURL != nil {
			t.Fatalf("penulis tanpa alias masih terbuka: %+v", author)
		}
	})

	t.Run("profil publik meneruskan seluruh field", func(t *testing.T) {
		author := ToThreadAuthor(&community.Profile{
			PublicID:   "abc123",
			Alias:      strPtr("Ranger Senja"),
			AvatarURL:  &avatar,
			ShowBadges: true,
		})
		if author.PublicID != "abc123" {
			t.Fatalf("PublicID = %q, mau abc123", author.PublicID)
		}
		if author.Alias != "Ranger Senja" {
			t.Fatalf("Alias = %q, mau Ranger Senja", author.Alias)
		}
		if author.AvatarURL == nil || *author.AvatarURL != avatar {
			t.Fatalf("AvatarURL = %v, mau %q", author.AvatarURL, avatar)
		}
		if !author.ShowBadges {
			t.Fatal("ShowBadges tidak diteruskan")
		}
	})
}

func TestToModerationAuthor(t *testing.T) {
	avatar := "https://contoh.id/a.png"

	t.Run("profil nil tetap menghasilkan penulis anonim", func(t *testing.T) {
		author := ToModerationAuthor(nil)
		if author == nil || author.Alias != AnonymousAuthorName {
			t.Fatalf("ToModerationAuthor(nil) = %+v", author)
		}
	})

	t.Run("alias tetap terlihat walau profilnya privat atau terblokir", func(t *testing.T) {
		author := ToModerationAuthor(&community.PublicProfile{
			PublicID:   "abc123",
			Alias:      "Ranger Senja",
			AvatarURL:  &avatar,
			ShowBadges: true,
		})
		if author.Alias != "Ranger Senja" {
			t.Fatalf("Alias = %q, mau Ranger Senja", author.Alias)
		}
		if author.AvatarURL == nil {
			t.Fatal("AvatarURL seharusnya ikut untuk petugas berizin")
		}
		if author.PublicID != "abc123" {
			t.Fatalf("PublicID = %q, mau abc123", author.PublicID)
		}
	})

	t.Run("alias kosong di antrean tetap Anonim", func(t *testing.T) {
		author := ToModerationAuthor(&community.PublicProfile{PublicID: "abc123"})
		if author.Alias != AnonymousAuthorName {
			t.Fatalf("Alias = %q, mau %q", author.Alias, AnonymousAuthorName)
		}
	})
}

func TestIsBlockedAccount(t *testing.T) {
	if IsBlockedAccount(nil) {
		t.Fatal("profil nil seharusnya tidak dianggap terblokir")
	}
	if IsBlockedAccount(&community.Profile{}) {
		t.Fatal("profil biasa seharusnya tidak dianggap terblokir")
	}
	if !IsBlockedAccount(&community.Profile{IsBlocked: true}) {
		t.Fatal("profil terblokir seharusnya terdeteksi")
	}
}

func TestIsPrivateAccount(t *testing.T) {
	if IsPrivateAccount(nil) {
		t.Fatal("profil nil seharusnya tidak dianggap privat")
	}
	if IsPrivateAccount(&community.Profile{ProfileVisibility: community.VisibilityMembers}) {
		t.Fatal("visibilitas members seharusnya tidak dianggap privat")
	}
	if !IsPrivateAccount(&community.Profile{ProfileVisibility: community.VisibilityPrivate}) {
		t.Fatal("visibilitas private seharusnya terdeteksi")
	}
}
