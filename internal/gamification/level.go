package gamification

import (
	"sort"
	"strings"
)

// LevelRuleView adalah bentuk murni dari LevelRule, supaya resolusi level
// dapat diuji tanpa database.
type LevelRuleView struct {
	Level int
	Name  string
	MinXP int
	Badge string
}

// ToLevelViews mengubah aturan level dari database menjadi bentuk murni.
func ToLevelViews(rules []*LevelRule) []LevelRuleView {
	views := make([]LevelRuleView, 0, len(rules))
	for _, r := range rules {
		if r == nil {
			continue
		}
		badge := r.Name
		if r.BadgeLabel != nil && *r.BadgeLabel != "" {
			badge = *r.BadgeLabel
		}
		views = append(views, LevelRuleView{
			Level: r.Level,
			Name:  r.Name,
			MinXP: r.MinXP,
			Badge: badge,
		})
	}
	return views
}

// sortedByMinXP menyalin lalu mengurutkan aturan naik berdasarkan min_xp.
func sortedByMinXP(rules []LevelRuleView) []LevelRuleView {
	out := make([]LevelRuleView, len(rules))
	copy(out, rules)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].MinXP != out[j].MinXP {
			return out[i].MinXP < out[j].MinXP
		}
		return out[i].Level < out[j].Level
	})
	return out
}

// ValidateLevelRules memeriksa kurva level yang dikirim admin sebelum ditimpa.
//
// Kurva wajib punya tepat satu titik dasar (min_xp 0): tanpa itu akun dengan XP
// nol tidak punya level yang jelas, dan persentase progres ke level berikutnya
// menjadi tidak bermakna. Ambang kembar juga ditolak karena membuat urutan
// level tidak deterministik.
func ValidateLevelRules(rules []*LevelRuleInput) error {
	if len(rules) == 0 {
		return invalid("kurva level tidak boleh kosong")
	}

	levels := map[int]bool{}
	minXPs := map[int]bool{}
	floors := 0

	for _, r := range rules {
		if r == nil {
			return invalid("baris level tidak boleh kosong")
		}
		if r.Level <= 0 {
			return invalid("level harus lebih dari nol")
		}
		if strings.TrimSpace(r.Name) == "" {
			return invalid("nama level level %d wajib diisi", r.Level)
		}
		if r.MinXP < 0 {
			return invalid("min_xp level %d tidak boleh negatif", r.Level)
		}
		if levels[r.Level] {
			return invalid("level %d muncul lebih dari sekali", r.Level)
		}
		if minXPs[r.MinXP] {
			return invalid("ambang %d XP dipakai lebih dari satu level", r.MinXP)
		}
		levels[r.Level] = true
		minXPs[r.MinXP] = true
		if r.MinXP == 0 {
			floors++
		}
	}

	if floors != 1 {
		return invalid("kurva level harus punya tepat satu level dengan min_xp 0")
	}
	return nil
}

// ResolveLevel mengembalikan level yang berlaku untuk totalXP.
//
// Total XP di bawah aturan terendah — termasuk nilai negatif akibat koreksi
// admin — tetap dipetakan ke level terendah, bukan ke level nol.
func ResolveLevel(rules []LevelRuleView, totalXP int) LevelRuleView {
	sorted := sortedByMinXP(rules)
	if len(sorted) == 0 {
		return LevelRuleView{}
	}
	current := sorted[0]
	for _, r := range sorted {
		if totalXP >= r.MinXP {
			current = r
		} else {
			break
		}
	}
	return current
}

// NextLevel mengembalikan level terdekat yang belum dicapai.
func NextLevel(rules []LevelRuleView, totalXP int) (LevelRuleView, bool) {
	for _, r := range sortedByMinXP(rules) {
		if r.MinXP > totalXP {
			return r, true
		}
	}
	return LevelRuleView{}, false
}

// ProgressToNext mengembalikan persentase perjalanan menuju level berikutnya
// dan sisa XP yang dibutuhkan. Bila sudah di level tertinggi, hasilnya 100/0.
func ProgressToNext(rules []LevelRuleView, totalXP int) (percent, remaining int) {
	current := ResolveLevel(rules, totalXP)
	next, ok := NextLevel(rules, totalXP)
	if !ok {
		return 100, 0
	}

	remaining = next.MinXP - totalXP
	if remaining < 0 {
		remaining = 0
	}

	span := next.MinXP - current.MinXP
	// Terjadi bila totalXP berada di bawah ambang level terendah: level
	// berikutnya adalah level terendah itu sendiri, jadi belum ada rentang
	// yang bisa dipersentasekan.
	if span <= 0 {
		return 0, remaining
	}

	done := totalXP - current.MinXP
	if done < 0 {
		done = 0
	}
	percent = done * 100 / span
	if percent > 100 {
		percent = 100
	}
	return percent, remaining
}
