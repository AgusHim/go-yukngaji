package fundraising

import (
	"testing"
	"time"

	"mainyuk/internal/community"
)

func TestResolveDonorName(t *testing.T) {
	cases := []struct {
		name       string
		profile    *community.Profile
		fallback   string
		wantName   string
		wantHidden bool
	}{
		{
			name:       "profil nil disembunyikan",
			profile:    nil,
			fallback:   "Budi",
			wantName:   AnonymousDonorName,
			wantHidden: true,
		},
		{
			name:       "akun terblokir disembunyikan",
			profile:    &community.Profile{Alias: strPtr("budi"), ProfileVisibility: community.VisibilityPublic, IsBlocked: true},
			fallback:   "Budi",
			wantName:   AnonymousDonorName,
			wantHidden: true,
		},
		{
			name:       "profil privat disembunyikan",
			profile:    &community.Profile{Alias: strPtr("budi"), ProfileVisibility: community.VisibilityPrivate},
			fallback:   "Budi",
			wantName:   AnonymousDonorName,
			wantHidden: true,
		},
		{
			name:       "alias dipakai bila ada",
			profile:    &community.Profile{Alias: strPtr("budi baik"), ProfileVisibility: community.VisibilityPublic},
			fallback:   "Budi Santoso",
			wantName:   "budi baik",
			wantHidden: false,
		},
		{
			name:       "alias kosong jatuh ke nama akun",
			profile:    &community.Profile{Alias: strPtr("   "), ProfileVisibility: community.VisibilityPublic},
			fallback:   "Budi Santoso",
			wantName:   "Budi Santoso",
			wantHidden: false,
		},
		{
			name:       "tanpa alias dan tanpa nama disembunyikan",
			profile:    &community.Profile{ProfileVisibility: community.VisibilityPublic},
			fallback:   "",
			wantName:   AnonymousDonorName,
			wantHidden: true,
		},
		{
			name:       "visibilitas members tetap tampil",
			profile:    &community.Profile{Alias: strPtr("budi"), ProfileVisibility: community.VisibilityMembers},
			fallback:   "Budi",
			wantName:   "budi",
			wantHidden: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			name, hidden := ResolveDonorName(tc.profile, tc.fallback)
			if name != tc.wantName || hidden != tc.wantHidden {
				t.Errorf("ResolveDonorName() = (%q, %v), ingin (%q, %v)",
					name, hidden, tc.wantName, tc.wantHidden)
			}
		})
	}
}

func TestToPublicDonation(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	msg := "semoga berkah"
	avatar := "https://example.test/a.png"

	base := func() *Donation {
		return &Donation{
			PublicID:      "abc123",
			Amount:        150_000,
			ShowAmount:    true,
			MessageStatus: MessageNone,
			CreatedAt:     now,
		}
	}

	t.Run("donasi normal menampilkan nama dan nominal", func(t *testing.T) {
		view := ToPublicDonation(base(), "budi", false, &avatar)
		if view.Amount == nil || *view.Amount != 150_000 {
			t.Fatalf("nominal = %v, ingin 150000", view.Amount)
		}
		if view.DonorName != "budi" || view.DonorHidden {
			t.Errorf("nama = (%q, %v), ingin (budi, false)", view.DonorName, view.DonorHidden)
		}
		if view.AvatarURL == nil {
			t.Error("avatar seharusnya tampil untuk donatur non-anonim")
		}
	})

	t.Run("anonim menyembunyikan nama DAN nominal", func(t *testing.T) {
		d := base()
		d.IsAnonymous = true
		view := ToPublicDonation(d, "budi", false, &avatar)
		if view.DonorName != AnonymousDonorName || !view.DonorHidden {
			t.Errorf("nama = (%q, %v), ingin (%q, true)", view.DonorName, view.DonorHidden, AnonymousDonorName)
		}
		if view.Amount != nil {
			t.Errorf("nominal = %v, ingin nil", *view.Amount)
		}
		if view.AvatarURL != nil {
			t.Error("avatar harus disembunyikan untuk donasi anonim")
		}
	})

	t.Run("show_amount false menyembunyikan nominal tapi nama tetap", func(t *testing.T) {
		d := base()
		d.ShowAmount = false
		view := ToPublicDonation(d, "budi", false, &avatar)
		if view.Amount != nil {
			t.Errorf("nominal = %v, ingin nil", *view.Amount)
		}
		if view.DonorName != "budi" || view.DonorHidden {
			t.Errorf("nama = (%q, %v), ingin (budi, false)", view.DonorName, view.DonorHidden)
		}
	})

	t.Run("identitas tersembunyi juga menyembunyikan nominal", func(t *testing.T) {
		view := ToPublicDonation(base(), AnonymousDonorName, true, &avatar)
		if view.Amount != nil {
			t.Errorf("nominal = %v, ingin nil", *view.Amount)
		}
		if view.AvatarURL != nil {
			t.Error("avatar harus disembunyikan saat identitas disembunyikan")
		}
	})

	t.Run("hanya pesan approved yang tampil", func(t *testing.T) {
		for _, status := range []string{MessageNone, MessagePending, MessageHidden} {
			d := base()
			d.MessageStatus = status
			d.Message = &msg
			if view := ToPublicDonation(d, "budi", false, nil); view.Message != nil {
				t.Errorf("status %q: pesan tampil, ingin disembunyikan", status)
			}
		}

		d := base()
		d.MessageStatus = MessageApproved
		d.Message = &msg
		view := ToPublicDonation(d, "budi", false, nil)
		if view.Message == nil || *view.Message != msg {
			t.Errorf("pesan approved tidak tampil: %v", view.Message)
		}
	})

	t.Run("donasi anonim boleh menampilkan pesannya", func(t *testing.T) {
		d := base()
		d.IsAnonymous = true
		d.MessageStatus = MessageApproved
		d.Message = &msg
		view := ToPublicDonation(d, "budi", false, nil)
		if view.Message == nil || *view.Message != msg {
			t.Errorf("pesan donasi anonim tidak tampil: %v", view.Message)
		}
	})

	t.Run("donasi nil menghasilkan view nil", func(t *testing.T) {
		if view := ToPublicDonation(nil, "budi", false, nil); view != nil {
			t.Errorf("view = %v, ingin nil", view)
		}
	})
}
