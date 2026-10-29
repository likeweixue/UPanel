package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThemeServiceDiscoverAndServe(t *testing.T) {
	dir := t.TempDir()
	themeDir := filepath.Join(dir, "flarum")
	if err := os.MkdirAll(filepath.Join(themeDir, "assets"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := `{"name":"Flarum","description":"demo","dark":false,"themeColor":"#f7f7f7","radius":4,"swatch":["#ffffff","#536f90"]}`
	if err := os.WriteFile(filepath.Join(themeDir, "theme.json"), []byte(manifest), 0600); err != nil {
		t.Fatal(err)
	}
	css := `html[data-theme="flarum"] { --color-primary: #536f90; }`
	if err := os.WriteFile(filepath.Join(themeDir, "theme.css"), []byte(css), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(themeDir, "assets", "font.woff2"), []byte("font"), 0600); err != nil {
		t.Fatal(err)
	}

	s := &ThemeService{dir: dir}
	themes := s.List()
	if len(themes) != 1 {
		t.Fatalf("themes = %d, want 1", len(themes))
	}
	got := themes[0]
	if got.ID != "flarum" || got.Name != "Flarum" || got.Dark || got.Radius == nil || *got.Radius != 4 {
		t.Fatalf("unexpected theme: %+v", got)
	}
	if data, err := s.CSS("flarum"); err != nil || string(data) != css {
		t.Fatalf("css = %q, err = %v", data, err)
	}
	if path, err := s.Asset("flarum", "assets/font.woff2"); err != nil || !strings.HasSuffix(path, "font.woff2") {
		t.Fatalf("asset path = %q, err = %v", path, err)
	}
	if _, err := s.Asset("flarum", "../secret"); err == nil {
		t.Fatal("asset path traversal should fail")
	}
	if links := s.IndexLinks(); !strings.Contains(links, "data-upanel-theme=\"flarum\"") {
		t.Fatalf("index links = %q", links)
	}

	// 缺少 theme.css 的目录不应出现在列表中
	if err := os.MkdirAll(filepath.Join(dir, "empty"), 0755); err != nil {
		t.Fatal(err)
	}
	if themes := s.List(); len(themes) != 1 {
		t.Fatalf("theme without css should be ignored, got %d", len(themes))
	}
	// 非法 ID 不应被发现
	if err := os.MkdirAll(filepath.Join(dir, "Bad_ID"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Bad_ID", "theme.css"), []byte("x"), 0600); err != nil {
		t.Fatal(err)
	}
	if themes := s.List(); len(themes) != 1 {
		t.Fatalf("invalid id should be ignored, got %d", len(themes))
	}
}
