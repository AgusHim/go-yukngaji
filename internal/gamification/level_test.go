package gamification

import "testing"

// kurvaDefault meniru seed migrasi 0008 supaya test membaca kurva yang sama
// dengan yang dipakai produksi secara default.
func kurvaDefault() []LevelRuleView {
	return []LevelRuleView{
		{Level: 1, Name: "Pemula", MinXP: 0, Badge: "Pemula"},
		{Level: 2, Name: "Pejuang", MinXP: 100, Badge: "Pejuang"},
		{Level: 3, Name: "Aktif", MinXP: 300, Badge: "Aktif"},
		{Level: 4, Name: "Kontributor", MinXP: 700, Badge: "Kontributor"},
		{Level: 5, Name: "Inspirator", MinXP: 1500, Badge: "Inspirator"},
		{Level: 6, Name: "Legenda", MinXP: 3000, Badge: "Legenda"},
	}
}

func TestResolveLevel(t *testing.T) {
	rules := kurvaDefault()

	cases := []struct {
		name    string
		totalXP int
		want    int
	}{
		{"nol", 0, 1},
		{"di bawah ambang level 2", 99, 1},
		{"tepat di ambang level 2", 100, 2},
		{"di antara ambang", 299, 2},
		{"tepat di ambang level 6", 3000, 6},
		{"jauh di atas level tertinggi", 999999, 6},
		// Koreksi admin bisa membuat total XP negatif. Pemetaannya harus
		// tetap ke level terendah, bukan ke level nol.
		{"XP negatif", -50, 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ResolveLevel(rules, tc.totalXP)
			if got.Level != tc.want {
				t.Fatalf("totalXP %d: got level %d want %d", tc.totalXP, got.Level, tc.want)
			}
		})
	}
}

func TestResolveLevelKurvaKosong(t *testing.T) {
	got := ResolveLevel(nil, 500)
	if got.Level != 0 {
		t.Fatalf("kurva kosong harus menghasilkan level nol, got %d", got.Level)
	}
}

func TestResolveLevelTidakBergantungUrutanInput(t *testing.T) {
	rules := kurvaDefault()
	// Dibalik: resolusi harus mengurutkan sendiri berdasarkan min_xp.
	for i, j := 0, len(rules)-1; i < j; i, j = i+1, j-1 {
		rules[i], rules[j] = rules[j], rules[i]
	}
	if got := ResolveLevel(rules, 700); got.Level != 4 {
		t.Fatalf("urutan input tidak boleh memengaruhi hasil, got level %d", got.Level)
	}
}

func TestNextLevel(t *testing.T) {
	rules := kurvaDefault()

	next, ok := NextLevel(rules, 0)
	if !ok || next.Level != 2 || next.MinXP != 100 {
		t.Fatalf("dari level 1 harus menuju level 2 (100 XP), got %+v ok=%v", next, ok)
	}

	next, ok = NextLevel(rules, 150)
	if !ok || next.Level != 3 {
		t.Fatalf("dari 150 XP harus menuju level 3, got %+v ok=%v", next, ok)
	}

	if _, ok := NextLevel(rules, 3000); ok {
		t.Fatal("di level tertinggi tidak boleh ada level berikutnya")
	}
	if _, ok := NextLevel(rules, 999999); ok {
		t.Fatal("di atas level tertinggi tidak boleh ada level berikutnya")
	}
}

func TestProgressToNext(t *testing.T) {
	rules := kurvaDefault()

	cases := []struct {
		name          string
		totalXP       int
		wantPercent   int
		wantRemaining int
	}{
		{"awal level 1", 0, 0, 100},
		{"separuh level 1", 50, 50, 50},
		{"akhir level 1", 99, 99, 1},
		{"awal level 2", 100, 0, 200},
		{"level tertinggi", 3000, 100, 0},
		{"di atas level tertinggi", 5000, 100, 0},
		// Total XP negatif akibat koreksi admin: level berikutnya adalah level
		// terendah itu sendiri (0 XP), jadi sisanya 40 XP menuju level 1 —
		// bukan 140 menuju level 2. Progres tidak boleh negatif.
		{"XP negatif", -40, 0, 40},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			percent, remaining := ProgressToNext(rules, tc.totalXP)
			if percent != tc.wantPercent || remaining != tc.wantRemaining {
				t.Fatalf("totalXP %d: got (%d%%, sisa %d) want (%d%%, sisa %d)",
					tc.totalXP, percent, remaining, tc.wantPercent, tc.wantRemaining)
			}
		})
	}
}

func TestProgressToNextTidakMelebihiSeratusPersen(t *testing.T) {
	rules := kurvaDefault()
	for xp := -200; xp <= 3200; xp += 7 {
		percent, remaining := ProgressToNext(rules, xp)
		if percent < 0 || percent > 100 {
			t.Fatalf("totalXP %d: persentase di luar rentang: %d", xp, percent)
		}
		if remaining < 0 {
			t.Fatalf("totalXP %d: sisa XP negatif: %d", xp, remaining)
		}
	}
}

func TestToLevelViewsMemakaiBadgeLabel(t *testing.T) {
	label := "Pejuang Sejati"
	rules := []*LevelRule{
		{Level: 1, Name: "Pemula", MinXP: 0},
		{Level: 2, Name: "Pejuang", MinXP: 100, BadgeLabel: &label},
	}

	views := ToLevelViews(rules)
	if len(views) != 2 {
		t.Fatalf("jumlah view tidak sesuai: %d", len(views))
	}
	// Tanpa badge_label, badge jatuh ke nama level.
	if views[0].Badge != "Pemula" {
		t.Fatalf("badge tanpa label harus memakai nama level, got %q", views[0].Badge)
	}
	if views[1].Badge != label {
		t.Fatalf("badge dengan label harus memakai labelnya, got %q", views[1].Badge)
	}

	if got := ResolveLevel(views, 150); got.Badge != label {
		t.Fatalf("level 2 harus membawa badge label, got %q", got.Badge)
	}
}

func TestToLevelViewsMengabaikanBarisNil(t *testing.T) {
	views := ToLevelViews([]*LevelRule{nil, {Level: 1, Name: "Pemula", MinXP: 0}})
	if len(views) != 1 {
		t.Fatalf("baris nil harus dilewati, got %d view", len(views))
	}
}

func TestValidateLevelRules(t *testing.T) {
	str := func(s string) *string { return &s }

	cases := []struct {
		name    string
		rules   []*LevelRuleInput
		wantErr bool
	}{
		{
			name: "kurva valid",
			rules: []*LevelRuleInput{
				{Level: 1, Name: "Pemula", MinXP: 0},
				{Level: 2, Name: "Pejuang", MinXP: 100, BadgeLabel: str("Pejuang")},
			},
		},
		{name: "kosong", rules: nil, wantErr: true},
		{
			name:    "tanpa titik dasar",
			rules:   []*LevelRuleInput{{Level: 1, Name: "Pemula", MinXP: 10}},
			wantErr: true,
		},
		{
			name: "dua titik dasar",
			rules: []*LevelRuleInput{
				{Level: 1, Name: "Pemula", MinXP: 0},
				{Level: 2, Name: "Pejuang", MinXP: 0},
			},
			wantErr: true,
		},
		{
			name: "level kembar",
			rules: []*LevelRuleInput{
				{Level: 1, Name: "Pemula", MinXP: 0},
				{Level: 1, Name: "Pejuang", MinXP: 100},
			},
			wantErr: true,
		},
		{
			name: "ambang kembar",
			rules: []*LevelRuleInput{
				{Level: 1, Name: "Pemula", MinXP: 0},
				{Level: 2, Name: "Pejuang", MinXP: 0},
			},
			wantErr: true,
		},
		{
			name:    "level nol",
			rules:   []*LevelRuleInput{{Level: 0, Name: "Pemula", MinXP: 0}},
			wantErr: true,
		},
		{
			name:    "nama kosong",
			rules:   []*LevelRuleInput{{Level: 1, Name: "  ", MinXP: 0}},
			wantErr: true,
		},
		{
			name:    "min_xp negatif",
			rules:   []*LevelRuleInput{{Level: 1, Name: "Pemula", MinXP: -1}},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateLevelRules(tc.rules)
			if tc.wantErr && err == nil {
				t.Fatal("harus ditolak, tetapi lolos")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("harus lolos, tetapi ditolak: %v", err)
			}
		})
	}
}

func TestValidateLevelRulesErrorAdalahValidasi(t *testing.T) {
	// Handler memetakan error ini ke HTTP 400 lewat apperr.IsValidation.
	err := ValidateLevelRules(nil)
	if err == nil {
		t.Fatal("harus menghasilkan error")
	}
	ve, ok := err.(interface{ IsValidation() bool })
	if !ok || !ve.IsValidation() {
		t.Fatalf("error harus menandai dirinya sebagai kegagalan validasi: %T", err)
	}
}
