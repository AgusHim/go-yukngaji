// Package authz memusatkan pemetaan role dan izin.
//
// Nilai role sengaja dipertahankan sama dengan data lama supaya baris yang
// sudah ada tetap valid. "user" dan "jamaah" diperlakukan sebagai satu tier
// anggota (lihat IsMember); tidak ada penulisan ulang data lama.
package authz

// Role yang dikenal sistem.
const (
	RoleAdmin  = "admin"
	RolePJ     = "pj"
	RoleRanger = "ranger"
	// RoleMember adalah nilai kanonik untuk anggota baru.
	RoleMember = "user"
	// RoleJamaah adalah nilai lama untuk anggota yang dibuat dari alur tamu.
	// Dipertahankan agar data lama tetap terbaca; keduanya satu tier.
	RoleJamaah = "jamaah"
)

// Permission adalah izin bertipe untuk pekerjaan lintas modul.
// Moderator belum menjadi role tersendiri; lihat docs/roles-and-permissions.md.
type Permission string

const (
	PermissionMissionManage Permission = "mission:manage"
	PermissionFundsManage   Permission = "funds:manage"
	PermissionProductManage Permission = "product:manage"
	PermissionModerate      Permission = "moderation:moderate"
	// PermissionMetricsView hanya membaca laporan agregat. Ia dipisah dari
	// izin tulis karena melihat angka ringkasan tidak sama dengan mengubah
	// apa pun; kalau nanti ada role yang hanya perlu membaca, ia cukup
	// mendapat izin ini.
	PermissionMetricsView Permission = "metrics:view"
	// PermissionUsersView membaca daftar akun. Ia dipisah dari izin tulis
	// karena daftar itu memuat data pribadi (email, telepon, alamat),
	// sedangkan mengubah akun adalah tindakan lain sama sekali.
	PermissionUsersView Permission = "users:view"
)

// rolePermissions memetakan role ke izin yang dimilikinya.
// Role yang tidak terdaftar tidak memiliki izin apa pun.
var rolePermissions = map[string]map[Permission]bool{
	RoleAdmin: {
		PermissionMissionManage: true,
		PermissionFundsManage:   true,
		PermissionProductManage: true,
		PermissionModerate:      true,
		PermissionMetricsView:   true,
		PermissionUsersView:     true,
	},
	RolePJ: {
		PermissionMissionManage: true,
		PermissionFundsManage:   true,
		PermissionProductManage: true,
		PermissionModerate:      true,
		PermissionMetricsView:   true,
		PermissionUsersView:     true,
	},
	RoleRanger: {
		PermissionModerate: true,
	},
}

// IsMember melaporkan apakah role termasuk tier anggota.
// Nilai lama "jamaah" dan nilai kanonik "user" sama-sama anggota.
func IsMember(role string) bool {
	return role == RoleMember || role == RoleJamaah
}

// HasRole melaporkan apakah role cocok dengan salah satu roles yang diminta.
func HasRole(role string, roles ...string) bool {
	for _, r := range roles {
		if role == r {
			return true
		}
	}
	return false
}

// Can melaporkan apakah role memiliki izin p.
func Can(role string, p Permission) bool {
	perms, ok := rolePermissions[role]
	if !ok {
		return false
	}
	return perms[p]
}

// IsStaff melaporkan apakah role termasuk pengurus (pj/admin) yang boleh
// mengelola data lintas event.
func IsStaff(role string) bool {
	return role == RoleAdmin || role == RolePJ
}

// IsRanger melaporkan apakah role boleh menjalankan tugas ranger.
func IsRanger(role string) bool {
	return role == RoleRanger || role == RolePJ || role == RoleAdmin
}
