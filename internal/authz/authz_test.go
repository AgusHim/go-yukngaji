package authz

import "testing"

func TestIsMember(t *testing.T) {
	tests := []struct {
		role string
		want bool
	}{
		{RoleMember, true},
		{RoleJamaah, true},
		{RoleRanger, false},
		{RolePJ, false},
		{RoleAdmin, false},
		{"", false},
		{"superadmin", false},
	}
	for _, tt := range tests {
		if got := IsMember(tt.role); got != tt.want {
			t.Errorf("IsMember(%q) = %v, want %v", tt.role, got, tt.want)
		}
	}
}

func TestCan(t *testing.T) {
	tests := []struct {
		name string
		role string
		perm Permission
		want bool
	}{
		{"admin boleh kelola misi", RoleAdmin, PermissionMissionManage, true},
		{"admin boleh moderasi", RoleAdmin, PermissionModerate, true},
		{"pj boleh kelola dana", RolePJ, PermissionFundsManage, true},
		{"ranger hanya moderasi", RoleRanger, PermissionModerate, true},
		{"ranger tidak kelola produk", RoleRanger, PermissionProductManage, false},
		{"anggota tidak punya izin", RoleMember, PermissionModerate, false},
		{"jamaah tidak punya izin", RoleJamaah, PermissionMissionManage, false},
		{"role tak dikenal", "superadmin", PermissionMissionManage, false},
		{"izin tak dikenal", RoleAdmin, Permission("unknown:perm"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Can(tt.role, tt.perm); got != tt.want {
				t.Errorf("Can(%q, %q) = %v, want %v", tt.role, tt.perm, got, tt.want)
			}
		})
	}
}

func TestIsRangerAndIsStaff(t *testing.T) {
	tests := []struct {
		role        string
		wantRanger  bool
		wantIsStaff bool
	}{
		{RoleRanger, true, false},
		{RolePJ, true, true},
		{RoleAdmin, true, true},
		{RoleMember, false, false},
		{RoleJamaah, false, false},
		{"", false, false},
	}
	for _, tt := range tests {
		if got := IsRanger(tt.role); got != tt.wantRanger {
			t.Errorf("IsRanger(%q) = %v, want %v", tt.role, got, tt.wantRanger)
		}
		if got := IsStaff(tt.role); got != tt.wantIsStaff {
			t.Errorf("IsStaff(%q) = %v, want %v", tt.role, got, tt.wantIsStaff)
		}
	}
}

func TestHasRole(t *testing.T) {
	if !HasRole(RoleMember, RoleAdmin, RoleMember) {
		t.Error("HasRole harus true bila salah satu role cocok")
	}
	if HasRole(RoleMember, RoleAdmin, RoleRanger) {
		t.Error("HasRole harus false bila tidak ada yang cocok")
	}
	if HasRole(RoleMember) {
		t.Error("HasRole tanpa daftar role harus false")
	}
}
