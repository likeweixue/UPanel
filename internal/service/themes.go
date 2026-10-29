package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"upanel/internal/config"
)

const (
	themeManifestName = "theme.json"
	themeCSSName      = "theme.css"
	maxThemeFileSize  = 512 * 1024
)

var themeIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,31}$`)
var themeColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{3}([0-9a-fA-F]{3})?$`)

// ThemeInfo 是 themes 目录中一个可插拔主题的公开元数据。
type ThemeInfo struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author,omitempty"`
	Dark        bool     `json:"dark"`
	ThemeColor  string   `json:"themeColor,omitempty"`
	Radius      *int     `json:"radius,omitempty"`
	Swatch      []string `json:"swatch,omitempty"`
}

type themeManifest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Version     string   `json:"version"`
	Author      string   `json:"author"`
	Dark        bool     `json:"dark"`
	ThemeColor  string   `json:"themeColor"`
	Radius      *int     `json:"radius"`
	Swatch      []string `json:"swatch"`
}

// ThemeService 读取 <base>/themes/<id>/ 下的主题包。
// 每个主题至少包含一个 theme.css；theme.json 可选。
type ThemeService struct {
	dir string
}

func NewThemeService() *ThemeService {
	cfg := config.Current()
	dir := filepath.Join(cfg.BaseDir, "themes")
	if err := os.MkdirAll(dir, 0755); err != nil {
		// 目录创建失败时 List 会返回空列表，不阻塞面板启动。
		_ = err
	}
	return &ThemeService{dir: dir}
}

func (s *ThemeService) Dir() string {
	return s.dir
}

func (s *ThemeService) List() []ThemeInfo {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return []ThemeInfo{}
	}
	out := make([]ThemeInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || !themeIDPattern.MatchString(entry.Name()) {
			continue
		}
		info, err := s.load(entry.Name())
		if err != nil {
			continue
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out
}

func (s *ThemeService) load(id string) (ThemeInfo, error) {
	if !themeIDPattern.MatchString(id) {
		return ThemeInfo{}, fmt.Errorf("主题 ID 不合法")
	}
	root := filepath.Join(s.dir, id)
	cssPath := filepath.Join(root, themeCSSName)
	stat, err := os.Stat(cssPath)
	if err != nil || !stat.Mode().IsRegular() || stat.Size() <= 0 || stat.Size() > maxThemeFileSize {
		return ThemeInfo{}, fmt.Errorf("主题 %s 缺少有效的 theme.css", id)
	}

	info := ThemeInfo{
		ID:          id,
		Name:        id,
		Description: "本地主题",
		Version:     "1.0.0",
	}
	manifestPath := filepath.Join(root, themeManifestName)
	if data, err := os.ReadFile(manifestPath); err == nil && len(data) <= 64*1024 {
		var m themeManifest
		if json.Unmarshal(data, &m) == nil {
			if strings.TrimSpace(m.Name) != "" {
				info.Name = strings.TrimSpace(m.Name)
			}
			if strings.TrimSpace(m.Description) != "" {
				info.Description = strings.TrimSpace(m.Description)
			}
			if strings.TrimSpace(m.Version) != "" {
				info.Version = strings.TrimSpace(m.Version)
			}
			info.Author = strings.TrimSpace(m.Author)
			info.Dark = m.Dark
			info.ThemeColor = strings.TrimSpace(m.ThemeColor)
			info.Radius = m.Radius
			info.Swatch = append([]string{}, m.Swatch...)
		}
	}

	if info.ThemeColor != "" && !themeColorPattern.MatchString(info.ThemeColor) {
		info.ThemeColor = ""
	}
	if info.ThemeColor == "" {
		if info.Dark {
			info.ThemeColor = "#171a21"
		} else {
			info.ThemeColor = "#ffffff"
		}
	}
	if len(info.Swatch) != 2 || !themeColorPattern.MatchString(info.Swatch[0]) || !themeColorPattern.MatchString(info.Swatch[1]) {
		if info.Dark {
			info.Swatch = []string{"#171a21", "#536f90"}
		} else {
			info.Swatch = []string{"#ffffff", "#536f90"}
		}
	}
	if info.Radius != nil && (*info.Radius < 0 || *info.Radius > 16) {
		info.Radius = nil
	}
	return info, nil
}

func (s *ThemeService) CSS(id string) ([]byte, error) {
	if _, err := s.load(id); err != nil {
		return nil, err
	}
	data, err := os.ReadFile(filepath.Join(s.dir, id, themeCSSName))
	if err != nil {
		return nil, err
	}
	if len(data) > maxThemeFileSize {
		return nil, fmt.Errorf("主题 CSS 超过大小限制")
	}
	return data, nil
}

// Asset 返回主题目录中的静态资源路径，并阻止目录穿越。
func (s *ThemeService) Asset(id, rel string) (string, error) {
	if _, err := s.load(id); err != nil {
		return "", err
	}
	rel = strings.TrimPrefix(strings.TrimSpace(rel), "/")
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) || filepath.IsAbs(clean) {
		return "", fmt.Errorf("资源路径不合法")
	}
	root := filepath.Join(s.dir, id)
	full := filepath.Join(root, clean)
	rootPrefix := filepath.Clean(root) + string(filepath.Separator)
	if !strings.HasPrefix(filepath.Clean(full), rootPrefix) {
		return "", fmt.Errorf("资源路径不合法")
	}
	stat, err := os.Stat(full)
	if err != nil || !stat.Mode().IsRegular() {
		return "", fmt.Errorf("资源不存在")
	}
	return full, nil
}

// IndexLinks 返回注入 index.html 的主题 CSS 链接。
// 路径相对 <base>，在安全入口下也会自动带上前缀。
func (s *ThemeService) IndexLinks() string {
	var b strings.Builder
	for _, theme := range s.List() {
		fmt.Fprintf(&b, `<link rel="stylesheet" href="api/themes/%s/theme.css" data-upanel-theme="%s">`+"\n", theme.ID, theme.ID)
	}
	return b.String()
}
