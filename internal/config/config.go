package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Config 面板运行配置。所有路径均由 BaseDir 推导，不再硬编码任何开发机路径。
type Config struct {
	// 服务器配置
	Port     string `json:"port"`
	BindIP   string `json:"bind_ip"`
	Mode     string `json:"mode"` // debug / release
	BaseDir  string `json:"base_dir"`
	Username string `json:"username"`

	// 认证配置（密码以 bcrypt 哈希存储，不保存明文）
	JWTSecret    string `json:"jwt_secret"`
	PasswordHash string `json:"password_hash"`

	// 安全入口
	SecurityEntry string `json:"security_entry"`

	// 运行时目录（相对路径会基于 BaseDir 解析）
	DataPath     string `json:"data_path"`
	WebsitePath  string `json:"website_path"`
	DatabasePath string `json:"database_path"`
	AppPath      string `json:"app_path"`
	StaticPath   string `json:"static_path"`

	// 网关 / 镜像配置
	GatewayImage    string   `json:"gateway_image"`
	GatewayHTTPPort  string   `json:"gateway_http_port"`
	GatewayHTTPSPort string   `json:"gateway_https_port"`
	PHPImage        string   `json:"php_image"`
	RegistryMirrors []string `json:"registry_mirrors"`
}

var (
	AppConfig *Config
	mu        sync.Mutex
)

// DefaultBaseDir 返回默认安装目录
func DefaultBaseDir() string {
	if v := os.Getenv("UPANEL_BASE_DIR"); v != "" {
		return v
	}
	return "/opt/upanel"
}

// LoadConfig 加载配置：先读磁盘上的 config.json，再用环境变量覆盖。
func LoadConfig() error {
	mu.Lock()
	defer mu.Unlock()

	base := DefaultBaseDir()
	cfg := NewDefaultConfig(base)

	// 读取已持久化的配置
	if data, err := os.ReadFile(configFilePath(base)); err == nil {
		var disk Config
		if err := json.Unmarshal(data, &disk); err == nil {
			mergeConfig(cfg, &disk)
		}
	}

	// 环境变量优先级最高（systemd / 容器场景）
	if v := os.Getenv("PANEL_PORT"); v != "" {
		cfg.Port = v
	}
	if v := os.Getenv("PANEL_BIND_IP"); v != "" {
		cfg.BindIP = v
	}
	if v := os.Getenv("GIN_MODE"); v != "" {
		cfg.Mode = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.JWTSecret = v
	}
	if v := os.Getenv("PANEL_ENTRY"); v != "" {
		cfg.SecurityEntry = v
	}

	cfg.normalize()
	AppConfig = cfg
	return cfg.EnsureDirs()
}

// NewDefaultConfig 生成一份带默认值的配置
func NewDefaultConfig(base string) *Config {
	if base == "" {
		base = DefaultBaseDir()
	}
	return &Config{
		Port:          "8080",
		BindIP:        "0.0.0.0",
		Mode:          "release",
		BaseDir:       base,
		Username:      "admin",
		JWTSecret:     randomString(32),
		SecurityEntry: "",
		DataPath:      "data",
		WebsitePath:   "data/websites",
		DatabasePath:  "data/databases",
		AppPath:       "data/apps",
		StaticPath:    "web",
		GatewayImage:  "nginx:alpine",
		GatewayHTTPPort:  "80",
		GatewayHTTPSPort: "443",
		PHPImage:      "php:8.2-fpm",
		RegistryMirrors: []string{},
	}
}

func mergeConfig(dst, src *Config) {
	if src.Port != "" {
		dst.Port = src.Port
	}
	if src.BindIP != "" {
		dst.BindIP = src.BindIP
	}
	if src.Mode != "" {
		dst.Mode = src.Mode
	}
	if src.Username != "" {
		dst.Username = src.Username
	}
	if src.JWTSecret != "" {
		dst.JWTSecret = src.JWTSecret
	}
	if src.PasswordHash != "" {
		dst.PasswordHash = src.PasswordHash
	}
	if src.SecurityEntry != "" {
		dst.SecurityEntry = src.SecurityEntry
	}
	if src.DataPath != "" {
		dst.DataPath = src.DataPath
	}
	if src.WebsitePath != "" {
		dst.WebsitePath = src.WebsitePath
	}
	if src.DatabasePath != "" {
		dst.DatabasePath = src.DatabasePath
	}
	if src.AppPath != "" {
		dst.AppPath = src.AppPath
	}
	if src.StaticPath != "" {
		dst.StaticPath = src.StaticPath
	}
	if src.GatewayImage != "" {
		dst.GatewayImage = src.GatewayImage
	}
	if src.GatewayHTTPPort != "" {
		dst.GatewayHTTPPort = src.GatewayHTTPPort
	}
	if src.GatewayHTTPSPort != "" {
		dst.GatewayHTTPSPort = src.GatewayHTTPSPort
	}
	if src.PHPImage != "" {
		dst.PHPImage = src.PHPImage
	}
	if len(src.RegistryMirrors) > 0 {
		dst.RegistryMirrors = src.RegistryMirrors
	}
}

func configFilePath(base string) string {
	return filepath.Join(base, "data", "config.json")
}

// ConfigFilePath 配置文件落盘位置
func (c *Config) ConfigFilePath() string {
	return configFilePath(c.BaseDir)
}

// resolve 把相对路径解析为基于 BaseDir 的绝对路径
func (c *Config) resolve(p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Join(c.BaseDir, p)
}

// normalize 归一化所有路径
func (c *Config) normalize() {
	c.DataPath = c.resolve(c.DataPath)
	c.WebsitePath = c.resolve(c.WebsitePath)
	c.DatabasePath = c.resolve(c.DatabasePath)
	c.AppPath = c.resolve(c.AppPath)
	c.StaticPath = c.resolve(c.StaticPath)
	if c.Port == "" {
		c.Port = "8080"
	}
	if c.BindIP == "" {
		c.BindIP = "0.0.0.0"
	}
	if c.Mode == "" {
		c.Mode = "release"
	}
	if c.Username == "" {
		c.Username = "admin"
	}
	if c.JWTSecret == "" {
		c.JWTSecret = randomString(32)
	}
	if c.GatewayImage == "" {
		c.GatewayImage = "nginx:alpine"
	}
	if c.GatewayHTTPPort == "" {
		c.GatewayHTTPPort = "80"
	}
	if c.GatewayHTTPSPort == "" {
		c.GatewayHTTPSPort = "443"
	}
	if c.PHPImage == "" {
		c.PHPImage = "php:8.2-fpm"
	}
	if c.RegistryMirrors == nil {
		c.RegistryMirrors = []string{}
	}
}

// EnsureDirs 创建所有需要的目录
func (c *Config) EnsureDirs() error {
	dirs := []string{
		c.DataPath,
		c.WebsitePath,
		c.DatabasePath,
		c.AppPath,
		filepath.Join(c.DataPath, "gateway", "conf.d"),
		filepath.Join(c.DataPath, "gateway", "certs"),
		filepath.Join(c.DataPath, "logs"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return err
		}
	}
	return nil
}

// GatewayConfDir 网关站点配置目录（宿主机侧）
func (c *Config) GatewayConfDir() string {
	return filepath.Join(c.DataPath, "gateway", "conf.d")
}

// GatewayCertDir 网关证书目录（宿主机侧）
func (c *Config) GatewayCertDir() string {
	return filepath.Join(c.DataPath, "gateway", "certs")
}

// GatewayNginxConf 网关主配置文件（宿主机侧）
func (c *Config) GatewayNginxConf() string {
	return filepath.Join(c.DataPath, "gateway", "nginx.conf")
}

// Save 持久化配置到磁盘
func (c *Config) Save() error {
	mu.Lock()
	defer mu.Unlock()

	c.normalize()
	if err := os.MkdirAll(filepath.Dir(c.ConfigFilePath()), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.ConfigFilePath(), data, 0600)
}

// SetPasswordHash 更新密码哈希并落盘
func (c *Config) SetPasswordHash(hash string) error {
	c.PasswordHash = hash
	return c.Save()
}

// SetPort 更新端口并落盘
func (c *Config) SetPort(port string) error {
	c.Port = port
	return c.Save()
}

// GetSecurityEntry 获取安全入口路径
func (c *Config) GetSecurityEntry() string {
	entry := strings.TrimSpace(c.SecurityEntry)
	if entry == "" {
		return ""
	}
	if !strings.HasPrefix(entry, "/") {
		entry = "/" + entry
	}
	return strings.TrimRight(entry, "/")
}

// Current 返回当前配置，未加载时返回默认配置
func Current() *Config {
	if AppConfig != nil {
		return AppConfig
	}
	c := NewDefaultConfig(DefaultBaseDir())
	c.normalize()
	return c
}
