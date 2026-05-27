package cmd

import "testing"

func TestAccessPolicyInputForMode(t *testing.T) {
	tests := []struct {
		name                 string
		mode                 string
		requirePlatformLogin bool
		allowlist            bool
	}{
		{name: "open", mode: "open", requirePlatformLogin: false},
		{name: "login", mode: "login", requirePlatformLogin: true},
		{name: "private", mode: "private", requirePlatformLogin: true, allowlist: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, err := accessPolicyInputForMode(tt.mode)
			if err != nil {
				t.Fatalf("accessPolicyInputForMode: %v", err)
			}
			if input.RequirePlatformLogin != tt.requirePlatformLogin {
				t.Fatalf("require platform login = %v, want %v", input.RequirePlatformLogin, tt.requirePlatformLogin)
			}
			allowlist := input.Credentials != nil && input.Credentials.Allowlist
			if allowlist != tt.allowlist {
				t.Fatalf("allowlist = %v, want %v", allowlist, tt.allowlist)
			}
		})
	}
}

func TestNormalizeDeployAccess(t *testing.T) {
	if got, err := normalizeDeployAccess(""); err != nil || got != "unchanged" {
		t.Fatalf("empty access = %q/%v, want unchanged", got, err)
	}
	if got, err := normalizeDeployAccess("OPEN"); err != nil || got != "open" {
		t.Fatalf("OPEN access = %q/%v, want open", got, err)
	}
	if _, err := normalizeDeployAccess("public"); err == nil {
		t.Fatal("expected invalid access mode")
	}
}
