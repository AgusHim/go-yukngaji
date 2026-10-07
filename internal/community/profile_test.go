package community

import (
	"testing"
	"time"
)

func TestValidateAlias(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		wantErr bool
	}{
		{name: "kosong berarti hapus alias", alias: "", wantErr: false},
		{name: "spasi saja dianggap kosong", alias: "   ", wantErr: false},
		{name: "alias wajar", alias: "Budi Santoso", wantErr: false},
		{name: "huruf angka dan pemisah", alias: "budi_88-x", wantErr: false},
		{name: "terlalu pendek", alias: "ab", wantErr: true},
		{name: "batas bawah lolos", alias: "abc", wantErr: false},
		{name: "terlalu panjang", alias: string(make([]rune, AliasMaxLen+1)), wantErr: true},
		{name: "karakter terlarang", alias: "budi@ynsolo", wantErr: true},
		{name: "emoji ditolak", alias: "budi🎉", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAlias(tt.alias)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateAlias(%q) error = %v, wantErr %v", tt.alias, err, tt.wantErr)
			}
		})
	}
}

func TestValidateAliasBatasPanjang(t *testing.T) {
	// Tepat di batas maksimum lolos, satu karakter lebih ditolak.
	atMax := ""
	for i := 0; i < AliasMaxLen; i++ {
		atMax += "a"
	}
	if err := ValidateAlias(atMax); err != nil {
		t.Errorf("alias %d karakter seharusnya lolos, dapat %v", AliasMaxLen, err)
	}
	if err := ValidateAlias(atMax + "a"); err == nil {
		t.Errorf("alias %d karakter seharusnya ditolak", AliasMaxLen+1)
	}
}

func TestSanitizeAlias(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{in: "  Budi   Santoso  ", want: "Budi Santoso"},
		{in: "Budi", want: "Budi"},
		{in: "\tBudi\nSantoso ", want: "Budi Santoso"},
	}
	for _, tt := range tests {
		if got := SanitizeAlias(tt.in); got != tt.want {
			t.Errorf("SanitizeAlias(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidateVisibility(t *testing.T) {
	for _, v := range []string{VisibilityPublic, VisibilityMembers, VisibilityPrivate} {
		if err := ValidateVisibility(v); err != nil {
			t.Errorf("ValidateVisibility(%q) seharusnya lolos, dapat %v", v, err)
		}
	}
	if err := ValidateVisibility("rahasia"); err == nil {
		t.Error("ValidateVisibility(\"rahasia\") seharusnya gagal")
	}
}

func TestIsProfileComplete(t *testing.T) {
	birth := time.Date(1998, time.May, 4, 0, 0, 0, 0, time.UTC)
	lengkap := ProfileCompletion{
		Name:            "Budi",
		Phone:           "08123456789",
		BirthDate:       &birth,
		ProvinceCode:    "33",
		DistrictCode:    "3374",
		SubDistrictCode: "3374010",
	}

	tests := []struct {
		name string
		mod  func(p *ProfileCompletion)
		want bool
	}{
		{name: "lengkap", mod: func(p *ProfileCompletion) {}, want: true},
		{name: "nama kosong", mod: func(p *ProfileCompletion) { p.Name = "  " }, want: false},
		{name: "telepon kosong", mod: func(p *ProfileCompletion) { p.Phone = "" }, want: false},
		{name: "tanggal lahir nil", mod: func(p *ProfileCompletion) { p.BirthDate = nil }, want: false},
		{name: "tanggal lahir zero", mod: func(p *ProfileCompletion) { z := time.Time{}; p.BirthDate = &z }, want: false},
		{name: "kode provinsi kosong", mod: func(p *ProfileCompletion) { p.ProvinceCode = "" }, want: false},
		{name: "kode kabupaten kosong", mod: func(p *ProfileCompletion) { p.DistrictCode = "" }, want: false},
		{name: "kode kecamatan kosong", mod: func(p *ProfileCompletion) { p.SubDistrictCode = "" }, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := lengkap
			tt.mod(&p)
			if got := IsProfileComplete(p); got != tt.want {
				t.Errorf("IsProfileComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToPublicProfileTidakMembocorkanIdInternal(t *testing.T) {
	alias := "Budi"
	p := &Profile{
		UserID:            "user-internal-123",
		PublicID:          "abc123",
		Alias:             &alias,
		ProfileVisibility: VisibilityPublic,
		IsBlocked:         false,
	}

	got := ToPublicProfile(p, "budi-fallback")
	if got.Alias != "Budi" {
		t.Errorf("alias = %q, want %q", got.Alias, "Budi")
	}
	if got.PublicID != "abc123" {
		t.Errorf("public_id = %q, want %q", got.PublicID, "abc123")
	}
}

func TestDisplayAliasJatuhKeNamaPengguna(t *testing.T) {
	if got := (&Profile{}).DisplayAlias("budi"); got != "budi" {
		t.Errorf("tanpa alias harus jatuh ke nama pengguna, dapat %q", got)
	}
	if got := (&Profile{}).DisplayAlias(""); got != "Anonim" {
		t.Errorf("tanpa alias dan tanpa nama pengguna harus \"Anonim\", dapat %q", got)
	}
	kosong := ""
	if got := (&Profile{Alias: &kosong}).DisplayAlias("budi"); got != "budi" {
		t.Errorf("alias string kosong harus jatuh ke nama pengguna, dapat %q", got)
	}
}
