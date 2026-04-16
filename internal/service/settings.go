package service

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Settings struct {
	// Docker 镜像加速
	RegistryMirrors []string `json:"registry_mirrors"`
	
	// 网站设置
	WebsitePath string `json:"website_path"`
	
	// 数据库设置
	DatabasePath string `json:"database_path"`
	
	// 主题设置
	Theme string `json:"theme"` // light, dark
	
	// 其他设置
	Language string `json:"language"`
}

type SettingsService struct {
	configPath string
	settings   *Settings
}

func NewSettingsService() *SettingsService {
	// 配置文件路径
	configPath := filepath.Join(os.Getenv("HOME"), ".upanel", "settings.json")
	os.MkdirAll(filepath.Dir(configPath), 0755)
	
	service := &SettingsService{
		configPath: configPath,
		settings:   &Settings{},
	}
	
	// 加载配置
	service.load()
	
	// 设置默认值
	if service.settings.WebsitePath == "" {
		service.settings.WebsitePath = "/Users/machangsheng/Downloads/Upanel/www"
	}
	if service.settings.DatabasePath == "" {
		service.settings.DatabasePath = "/Users/machangsheng/Downloads/Upanel/databases"
	}
	if service.settings.Theme == "" {
		service.settings.Theme = "light"
	}
	if service.settings.Language == "" {
		service.settings.Language = "zh-CN"
	}
	
	return service
}

// 加载配置
func (s *SettingsService) load() {
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		return
	}
	json.Unmarshal(data, s.settings)
}

// 保存配置
func (s *SettingsService) save() error {
	data, err := json.MarshalIndent(s.settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.configPath, data, 0644)
}

// 获取所有设置
func (s *SettingsService) GetAll() *Settings {
	return s.settings
}

// 更新 Docker 镜像加速
func (s *SettingsService) SetRegistryMirrors(mirrors []string) error {
	s.settings.RegistryMirrors = mirrors
	return s.save()
}

// 更新主题
func (s *SettingsService) SetTheme(theme string) error {
	s.settings.Theme = theme
	return s.save()
}

// 更新网站路径
func (s *SettingsService) SetWebsitePath(path string) error {
	s.settings.WebsitePath = path
	return s.save()
}

// 更新数据库路径
func (s *SettingsService) SetDatabasePath(path string) error {
	s.settings.DatabasePath = path
	return s.save()
}
