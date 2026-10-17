package org

import "testing"

func TestParseRole(t *testing.T) {
	cases := []struct {
		in     string
		want   Role
		wantOK bool
	}{
		{"OWNER", RoleOwner, true},
		{"ADMIN", RoleAdmin, true},
		{"MEMBER", RoleMember, true},
		{"owner", "", false},
		{"GUEST", "", false},
		{"", "", false},
	}

	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got, ok := ParseRole(tc.in)
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("ParseRole(%q) = (%q, %v), want (%q, %v)", tc.in, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}

func TestRoleRank(t *testing.T) {
	if RoleOwner.Rank() <= RoleAdmin.Rank() {
		t.Fatal("OWNER should rank higher than ADMIN")
	}
	if RoleAdmin.Rank() <= RoleMember.Rank() {
		t.Fatal("ADMIN should rank higher than MEMBER")
	}
	if Role("").Rank() != 0 {
		t.Fatal("unknown role rank should be 0")
	}
}

func TestRoleCanManageMembers(t *testing.T) {
	if !RoleOwner.CanManageMembers() {
		t.Fatal("OWNER should manage members")
	}
	if !RoleAdmin.CanManageMembers() {
		t.Fatal("ADMIN should manage members")
	}
	if RoleMember.CanManageMembers() {
		t.Fatal("MEMBER should not manage members")
	}
}
