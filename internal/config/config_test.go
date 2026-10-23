package config

import "testing"

func TestNormalizeSecurityEntry(t *testing.T) {
	valid := map[string]string{
		"":           "",
		"panel":      "/panel",
		"/panel/":    "/panel",
		"a/b":        "/a/b",
		"my-panel_1": "/my-panel_1",
	}
	for in, want := range valid {
		got, err := NormalizeSecurityEntry(in)
		if err != nil {
			t.Errorf("NormalizeSecurityEntry(%q) 报错: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizeSecurityEntry(%q) = %q, want %q", in, got, want)
		}
	}

	invalid := []string{"../etc", "a b", "-bad", "a//b", "a?b", "中文"}
	for _, in := range invalid {
		if _, err := NormalizeSecurityEntry(in); err == nil {
			t.Errorf("NormalizeSecurityEntry(%q) 应报错", in)
		}
	}
}

func TestValidatePort(t *testing.T) {
	for _, p := range []string{"1", "80", "65535"} {
		if err := ValidatePort(p); err != nil {
			t.Errorf("端口 %s 应有效: %v", p, err)
		}
	}
	for _, p := range []string{"", "0", "65536", "-1", "abc", "80.5"} {
		if err := ValidatePort(p); err == nil {
			t.Errorf("端口 %s 应无效", p)
		}
	}
}

func TestNormalizeSecurityEntryReserved(t *testing.T) {
	for _, in := range []string{"api", "api/v1", "assets", "assets/x"} {
		if _, err := NormalizeSecurityEntry(in); err == nil {
			t.Errorf("保留路径 %q 应被拒绝", in)
		}
	}
}
