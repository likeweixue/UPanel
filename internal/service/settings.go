package service

import (
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"upanel/internal/config"
)

// Settings 面板偏好设置（持久化在 data/settings.json）
type Settings struct {
	// 面板
	PanelName     string `json:"panelName"`
	PanelPort     string `json:"panelPort"`
	BindIP        string `json:"bindIP"`
	SecurityEntry string `json:"securityEntry"`

	// 路径设置
	WebsitePath  string `json:"websitePath"`
	DatabasePath string `json:"databasePath"`
	AppPath      string `json:"appPath"`

	// 网关
	GatewayImage     string `json:"gatewayImage"`
	GatewayHTTPPort  string `json:"gatewayHttpPort"`
	GatewayHTTPSPort string `json:"gatewayHttpsPort"`

	// 镜像
	RegistryMirrors []string `json:"registryMirrors"`

	// 界面
	Theme             string  `json:"theme"`
	Language          string  `json:"language"`
	Density           string  `json:"density"`
	CornerRadius      *int    `json:"cornerRadius"`
	FooterVisible     *bool   `json:"footerVisible"`
	FooterText        *string `json:"footerText"`
	FooterShowVersion *bool   `json:"footerShowVersion"`
	FooterShowGithub  *bool   `json:"footerShowGithub"`
}

// SettingsService 设置服务
type SettingsService struct {
	path     string
	settings *Settings
}

var settingsMu sync.Mutex

// NewSettingsService 创建设置服务
func NewSettingsService() *SettingsService {
	cfg := config.Current()
	p := filepath.Join(cfg.DataPath, "settings.json")

	s := &SettingsService{path: p, settings: defaultSettings(cfg)}
	s.load()
	// 服务器相关配置以 config.json 为唯一权威来源，避免 CLI 修改后与 settings.json 不一致。
	s.settings.PanelPort = cfg.Port
	s.settings.BindIP = cfg.BindIP
	s.settings.SecurityEntry = cfg.GetSecurityEntry()
	s.settings.WebsitePath = cfg.WebsitePath
	s.settings.DatabasePath = cfg.DatabasePath
	s.settings.AppPath = cfg.AppPath
	s.settings.GatewayImage = cfg.GatewayImage
	s.settings.GatewayHTTPPort = cfg.GatewayHTTPPort
	s.settings.GatewayHTTPSPort = cfg.GatewayHTTPSPort
	s.settings.RegistryMirrors = append([]string{}, cfg.RegistryMirrors...)
	return s
}

func defaultSettings(cfg *config.Config) *Settings {
	return &Settings{
		PanelName:         "UPanel",
		PanelPort:         cfg.Port,
		BindIP:            cfg.BindIP,
		SecurityEntry:     cfg.GetSecurityEntry(),
		WebsitePath:       cfg.WebsitePath,
		DatabasePath:      cfg.DatabasePath,
		AppPath:           cfg.AppPath,
		GatewayImage:      cfg.GatewayImage,
		GatewayHTTPPort:   cfg.GatewayHTTPPort,
		GatewayHTTPSPort:  cfg.GatewayHTTPSPort,
		RegistryMirrors:   append([]string{}, cfg.RegistryMirrors...),
		Theme:             "light",
		Language:          "zh-CN",
		Density:           "comfortable",
		CornerRadius:      ptrInt(4),
		FooterVisible:     ptrBool(true),
		FooterText:        ptrString(""),
		FooterShowVersion: ptrBool(true),
		FooterShowGithub:  ptrBool(true),
	}
}

func ptrBool(v bool) *bool       { return &v }
func ptrInt(v int) *int          { return &v }
func ptrString(v string) *string { return &v }

func (s *SettingsService) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var disk Settings
	if err := json.Unmarshal(data, &disk); err != nil {
		return
	}
	if disk.PanelName != "" {
		s.settings.PanelName = disk.PanelName
	}
	if disk.PanelPort != "" {
		s.settings.PanelPort = disk.PanelPort
	}
	if disk.BindIP != "" {
		s.settings.BindIP = disk.BindIP
	}
	if disk.SecurityEntry != "" {
		s.settings.SecurityEntry = disk.SecurityEntry
	}
	if disk.WebsitePath != "" {
		s.settings.WebsitePath = disk.WebsitePath
	}
	if disk.DatabasePath != "" {
		s.settings.DatabasePath = disk.DatabasePath
	}
	if disk.AppPath != "" {
		s.settings.AppPath = disk.AppPath
	}
	if disk.GatewayImage != "" {
		s.settings.GatewayImage = disk.GatewayImage
	}
	if disk.GatewayHTTPPort != "" {
		s.settings.GatewayHTTPPort = disk.GatewayHTTPPort
	}
	if disk.GatewayHTTPSPort != "" {
		s.settings.GatewayHTTPSPort = disk.GatewayHTTPSPort
	}
	if len(disk.RegistryMirrors) > 0 {
		s.settings.RegistryMirrors = disk.RegistryMirrors
	}
	if disk.Theme != "" {
		s.settings.Theme = disk.Theme
	}
	if disk.Language != "" {
		s.settings.Language = disk.Language
	}
	if disk.Density != "" {
		s.settings.Density = disk.Density
	}
	if disk.CornerRadius != nil {
		s.settings.CornerRadius = disk.CornerRadius
	}
	if disk.FooterVisible != nil {
		s.settings.FooterVisible = disk.FooterVisible
	}
	if disk.FooterText != nil {
		s.settings.FooterText = disk.FooterText
	}
	if disk.FooterShowVersion != nil {
		s.settings.FooterShowVersion = disk.FooterShowVersion
	}
	if disk.FooterShowGithub != nil {
		s.settings.FooterShowGithub = disk.FooterShowGithub
	}
}

func (s *SettingsService) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return err
	}
	return writeFileAtomic(s.path, data, 0600)
}

// GetAll 获取全部设置（返回副本，避免调用方修改内部状态）
func (s *SettingsService) GetAll() *Settings {
	cp := *s.settings
	cp.RegistryMirrors = append([]string{}, s.settings.RegistryMirrors...)
	cp.CornerRadius = ptrInt(4)
	if s.settings.CornerRadius != nil {
		cp.CornerRadius = ptrInt(*s.settings.CornerRadius)
	}
	cp.FooterVisible = ptrBool(true)
	if s.settings.FooterVisible != nil {
		cp.FooterVisible = ptrBool(*s.settings.FooterVisible)
	}
	cp.FooterText = ptrString("")
	if s.settings.FooterText != nil {
		cp.FooterText = ptrString(*s.settings.FooterText)
	}
	cp.FooterShowVersion = ptrBool(true)
	if s.settings.FooterShowVersion != nil {
		cp.FooterShowVersion = ptrBool(*s.settings.FooterShowVersion)
	}
	cp.FooterShowGithub = ptrBool(true)
	if s.settings.FooterShowGithub != nil {
		cp.FooterShowGithub = ptrBool(*s.settings.FooterShowGithub)
	}
	return &cp
}

func validateBindIP(ip string) error {
	if ip == "" {
		return nil
	}
	if ip == "localhost" {
		return nil
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return fmt.Errorf("监听地址必须是合法的 IPv4/IPv6 地址")
	}
	return nil
}

// validateManagedPath 限制面板数据目录必须位于安装目录之内，
// 防止把 / 或 /etc 之类的宿主机路径挂载进网关容器。
func validateManagedPath(base, label, p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", nil
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(base, p)
	}
	p = filepath.Clean(p)
	rel, err := filepath.Rel(base, p)
	if err != nil {
		return "", fmt.Errorf("%s 路径无效: %w", label, err)
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("%s 必须位于安装目录 %s 的子目录中", label, base)
	}
	return p, nil
}

func validateMirrors(mirrors []string) ([]string, error) {
	if len(mirrors) > 10 {
		return nil, fmt.Errorf("镜像加速地址最多 10 个")
	}
	out := make([]string, 0, len(mirrors))
	for _, m := range mirrors {
		m = strings.TrimSpace(m)
		if m == "" {
			continue
		}
		u, err := url.Parse(m)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return nil, fmt.Errorf("镜像加速地址格式不合法: %s", m)
		}
		out = append(out, strings.TrimRight(m, "/"))
	}
	return out, nil
}

// Update 更新设置。
// 返回 restartRequired 表示面板端口/监听地址/安全入口发生变化，需要重启服务。
func (s *SettingsService) Update(patch *Settings) (bool, error) {
	settingsMu.Lock()
	defer settingsMu.Unlock()

	cfg := config.Current()
	restartRequired := false
	gatewayChanged := false

	// ---- 先完整校验，全部通过后才落盘 ----
	panelPort := cfg.Port
	if patch.PanelPort != "" && patch.PanelPort != s.settings.PanelPort {
		if err := config.ValidatePort(patch.PanelPort); err != nil {
			return false, fmt.Errorf("面板端口无效: %w", err)
		}
		panelPort = strings.TrimSpace(patch.PanelPort)
		restartRequired = true
	}
	bindIP := cfg.BindIP
	if patch.BindIP != "" && patch.BindIP != s.settings.BindIP {
		if err := validateBindIP(patch.BindIP); err != nil {
			return false, err
		}
		bindIP = strings.TrimSpace(patch.BindIP)
		restartRequired = true
	}
	entry := cfg.GetSecurityEntry()
	if patch.SecurityEntry != s.settings.SecurityEntry {
		normalized, err := config.NormalizeSecurityEntry(patch.SecurityEntry)
		if err != nil {
			return false, err
		}
		if normalized != entry {
			entry = normalized
			restartRequired = true
		}
	}

	websitePath, err := validateManagedPath(cfg.BaseDir, "网站目录", patch.WebsitePath)
	if err != nil {
		return false, err
	}
	databasePath, err := validateManagedPath(cfg.BaseDir, "数据库目录", patch.DatabasePath)
	if err != nil {
		return false, err
	}
	appPath, err := validateManagedPath(cfg.BaseDir, "应用目录", patch.AppPath)
	if err != nil {
		return false, err
	}

	gatewayImage := cfg.GatewayImage
	if patch.GatewayImage != "" && patch.GatewayImage != s.settings.GatewayImage {
		if err := validateImageRef(patch.GatewayImage); err != nil {
			return false, err
		}
		gatewayImage = patch.GatewayImage
		gatewayChanged = true
	}
	gatewayHTTP := cfg.GatewayHTTPPort
	if patch.GatewayHTTPPort != "" && patch.GatewayHTTPPort != s.settings.GatewayHTTPPort {
		if err := config.ValidatePort(patch.GatewayHTTPPort); err != nil {
			return false, fmt.Errorf("网关 HTTP 端口无效: %w", err)
		}
		gatewayHTTP = strings.TrimSpace(patch.GatewayHTTPPort)
		gatewayChanged = true
	}
	gatewayHTTPS := cfg.GatewayHTTPSPort
	if patch.GatewayHTTPSPort != "" && patch.GatewayHTTPSPort != s.settings.GatewayHTTPSPort {
		if err := config.ValidatePort(patch.GatewayHTTPSPort); err != nil {
			return false, fmt.Errorf("网关 HTTPS 端口无效: %w", err)
		}
		gatewayHTTPS = strings.TrimSpace(patch.GatewayHTTPSPort)
		gatewayChanged = true
	}
	if gatewayHTTP != "" && gatewayHTTP == gatewayHTTPS {
		return false, fmt.Errorf("网关 HTTP 与 HTTPS 端口不能相同")
	}
	if panelPort != cfg.Port {
		addr := net.JoinHostPort(bindIP, panelPort)
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			return false, fmt.Errorf("面板监听地址 %s 不可用: %w", addr, err)
		}
		_ = ln.Close()
	}
	if gatewayChanged {
		for _, port := range []string{gatewayHTTP, gatewayHTTPS} {
			if port == "" || port == cfg.GatewayHTTPPort || port == cfg.GatewayHTTPSPort {
				continue
			}
			ln, err := net.Listen("tcp", "0.0.0.0:"+port)
			if err != nil {
				return false, fmt.Errorf("网关端口 %s 已被占用，请先停止占用该端口的服务", port)
			}
			_ = ln.Close()
		}
	}
	mirrors, err := validateMirrors(patch.RegistryMirrors)
	if err != nil {
		return false, err
	}
	if patch.PanelName != "" {
		name := strings.TrimSpace(patch.PanelName)
		if name == "" || len(name) > 64 || strings.ContainsAny(name, "\r\n\x00") {
			return false, fmt.Errorf("面板名称不能为空且最多 64 个字符")
		}
	}
	if patch.Language != "" && patch.Language != "zh-CN" && patch.Language != "en-US" {
		return false, fmt.Errorf("不支持的语言: %s", patch.Language)
	}
	if patch.Theme != "" {
		switch patch.Theme {
		case "light", "dark", "system", "retro-yellow", "green-paper", "black-gold", "ember":
		default:
			return false, fmt.Errorf("不支持的主题: %s", patch.Theme)
		}
	}
	if patch.Density != "" && patch.Density != "comfortable" && patch.Density != "compact" {
		return false, fmt.Errorf("不支持的密度: %s", patch.Density)
	}

	// ---- 应用 ----
	old := *cfg
	old.RegistryMirrors = append([]string{}, cfg.RegistryMirrors...)

	if websitePath != "" {
		cfg.WebsitePath = websitePath
		s.settings.WebsitePath = websitePath
	}
	if databasePath != "" {
		cfg.DatabasePath = databasePath
		s.settings.DatabasePath = databasePath
	}
	if appPath != "" {
		cfg.AppPath = appPath
		s.settings.AppPath = appPath
	}
	if gatewayImage != cfg.GatewayImage {
		cfg.GatewayImage = gatewayImage
		s.settings.GatewayImage = gatewayImage
	}
	if gatewayHTTP != cfg.GatewayHTTPPort {
		cfg.GatewayHTTPPort = gatewayHTTP
		s.settings.GatewayHTTPPort = gatewayHTTP
	}
	if gatewayHTTPS != cfg.GatewayHTTPSPort {
		cfg.GatewayHTTPSPort = gatewayHTTPS
		s.settings.GatewayHTTPSPort = gatewayHTTPS
	}
	if len(mirrors) > 0 || patch.RegistryMirrors != nil {
		cfg.RegistryMirrors = mirrors
		s.settings.RegistryMirrors = mirrors
	}
	cfg.Port = panelPort
	cfg.BindIP = bindIP
	cfg.SecurityEntry = strings.TrimPrefix(entry, "/")
	s.settings.PanelPort = panelPort
	s.settings.BindIP = bindIP
	s.settings.SecurityEntry = entry
	if patch.PanelName != "" {
		s.settings.PanelName = strings.TrimSpace(patch.PanelName)
	}
	if patch.Theme != "" {
		s.settings.Theme = patch.Theme
	}
	if patch.Language != "" {
		s.settings.Language = patch.Language
	}
	if patch.Density != "" {
		s.settings.Density = patch.Density
	}
	if patch.CornerRadius != nil {
		if *patch.CornerRadius < 0 || *patch.CornerRadius > 16 {
			return false, fmt.Errorf("界面圆角必须在 0-16px 之间")
		}
		s.settings.CornerRadius = ptrInt(*patch.CornerRadius)
	}
	if patch.FooterVisible != nil {
		s.settings.FooterVisible = ptrBool(*patch.FooterVisible)
	}
	if patch.FooterText != nil {
		text := strings.TrimSpace(*patch.FooterText)
		if len([]rune(text)) > 80 || strings.ContainsAny(text, "\r\n\x00") {
			return false, fmt.Errorf("底部栏文字最多 80 个字符，且不能包含换行")
		}
		s.settings.FooterText = ptrString(text)
	}
	if patch.FooterShowVersion != nil {
		s.settings.FooterShowVersion = ptrBool(*patch.FooterShowVersion)
	}
	if patch.FooterShowGithub != nil {
		s.settings.FooterShowGithub = ptrBool(*patch.FooterShowGithub)
	}

	if err := cfg.Save(); err != nil {
		*cfg = old
		return false, err
	}
	if err := cfg.EnsureDirs(); err != nil {
		*cfg = old
		_ = cfg.Save()
		return false, err
	}
	if err := s.save(); err != nil {
		*cfg = old
		_ = cfg.Save()
		return false, err
	}

	pathsChanged := (websitePath != "" && websitePath != old.WebsitePath) ||
		(databasePath != "" && databasePath != old.DatabasePath) ||
		(appPath != "" && appPath != old.AppPath)
	if gatewayChanged || pathsChanged {
		if err := restartGateway(cfg); err != nil {
			*cfg = old
			_ = cfg.Save()
			_ = restartGateway(cfg)
			return false, fmt.Errorf("网关重启失败，已恢复原配置: %w", err)
		}
	}

	return restartRequired, nil
}

// restartGateway 重建网关容器以应用新镜像/端口/挂载路径
func restartGateway(cfg *config.Config) error {
	rt, err := NewRuntime()
	if err != nil {
		return err
	}
	defer rt.Close()
	g := NewGatewayServiceWith(rt)
	if err := rt.Remove(GatewayName); err != nil {
		return err
	}
	return g.Ensure(nil)
}

// SetTheme 更新主题
func (s *SettingsService) SetTheme(theme string) error {
	switch theme {
	case "light", "dark", "system", "retro-yellow", "green-paper", "black-gold", "ember":
	default:
		return fmt.Errorf("不支持的主题: %s", theme)
	}
	settingsMu.Lock()
	defer settingsMu.Unlock()
	s.settings.Theme = theme
	return s.save()
}

// SetRegistryMirrors 更新镜像加速
func (s *SettingsService) SetRegistryMirrors(mirrors []string) error {
	settingsMu.Lock()
	defer settingsMu.Unlock()
	out, err := validateMirrors(mirrors)
	if err != nil {
		return err
	}
	s.settings.RegistryMirrors = out
	cfg := config.Current()
	cfg.RegistryMirrors = out
	_ = cfg.Save()
	return s.save()
}
