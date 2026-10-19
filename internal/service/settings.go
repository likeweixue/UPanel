package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"upanel/internal/config"
)

// Settings 面板偏好设置（持久化在 data/settings.json）
type Settings struct {
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
	Theme    string `json:"theme"`
	Language string `json:"language"`
}

// SettingsService 设置服务
type SettingsService struct {
	mu       sync.Mutex
	path     string
	settings *Settings
}

// NewSettingsService 创建设置服务
func NewSettingsService() *SettingsService {
	cfg := config.Current()
	p := filepath.Join(cfg.DataPath, "settings.json")

	s := &SettingsService{path: p, settings: defaultSettings(cfg)}
	s.load()
	return s
}

func defaultSettings(cfg *config.Config) *Settings {
	return &Settings{
		WebsitePath:      cfg.WebsitePath,
		DatabasePath:     cfg.DatabasePath,
		AppPath:          cfg.AppPath,
		GatewayImage:     cfg.GatewayImage,
		GatewayHTTPPort:  cfg.GatewayHTTPPort,
		GatewayHTTPSPort: cfg.GatewayHTTPSPort,
		RegistryMirrors:  cfg.RegistryMirrors,
		Theme:            "light",
		Language:         "zh-CN",
	}
}

func (s *SettingsService) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	var disk Settings
	if err := json.Unmarshal(data, &disk); err != nil {
		return
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
}

func (s *SettingsService) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

// GetAll 获取全部设置
func (s *SettingsService) GetAll() *Settings {
	return s.settings
}

// Update 更新设置；网关与路径类改动同步写回主配置并生效
func (s *SettingsService) Update(patch *Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := config.Current()
	needGatewayRestart := false

	if patch.WebsitePath != "" {
		s.settings.WebsitePath = patch.WebsitePath
		cfg.WebsitePath = patch.WebsitePath
	}
	if patch.DatabasePath != "" {
		s.settings.DatabasePath = patch.DatabasePath
		cfg.DatabasePath = patch.DatabasePath
	}
	if patch.AppPath != "" {
		s.settings.AppPath = patch.AppPath
		cfg.AppPath = patch.AppPath
	}
	if patch.GatewayImage != "" && patch.GatewayImage != s.settings.GatewayImage {
		s.settings.GatewayImage = patch.GatewayImage
		cfg.GatewayImage = patch.GatewayImage
		needGatewayRestart = true
	}
	if patch.GatewayHTTPPort != "" && patch.GatewayHTTPPort != s.settings.GatewayHTTPPort {
		s.settings.GatewayHTTPPort = patch.GatewayHTTPPort
		cfg.GatewayHTTPPort = patch.GatewayHTTPPort
		needGatewayRestart = true
	}
	if patch.GatewayHTTPSPort != "" && patch.GatewayHTTPSPort != s.settings.GatewayHTTPSPort {
		s.settings.GatewayHTTPSPort = patch.GatewayHTTPSPort
		cfg.GatewayHTTPSPort = patch.GatewayHTTPSPort
		needGatewayRestart = true
	}
	if len(patch.RegistryMirrors) > 0 {
		s.settings.RegistryMirrors = patch.RegistryMirrors
		cfg.RegistryMirrors = patch.RegistryMirrors
	}
	if patch.Theme != "" {
		s.settings.Theme = patch.Theme
	}
	if patch.Language != "" {
		s.settings.Language = patch.Language
	}

	if err := cfg.Save(); err != nil {
		return err
	}
	if err := s.save(); err != nil {
		return err
	}
	if err := cfg.EnsureDirs(); err != nil {
		return err
	}

	if needGatewayRestart {
		gw, err := NewGatewayService()
		if err != nil {
			return err
		}
		defer gw.Close()
		// 重建网关容器以应用新镜像 / 新端口
		if err := gw.rt.Remove(GatewayName); err != nil {
			return err
		}
		return gw.Ensure(nil)
	}
	return nil
}

// SetTheme 更新主题
func (s *SettingsService) SetTheme(theme string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings.Theme = theme
	return s.save()
}

// SetRegistryMirrors 更新镜像加速
func (s *SettingsService) SetRegistryMirrors(mirrors []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.settings.RegistryMirrors = mirrors
	cfg := config.Current()
	cfg.RegistryMirrors = mirrors
	_ = cfg.Save()
	return s.save()
}
