package config

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
	// SessionVersion 会话版本：登出时递增，使所有已签发令牌立即失效。
	// 该值持久化在 config.json 中，因此重启后已登出的令牌仍然无效。
	SessionVersion int64 `json:"session_version"`

	// 安全入口
	SecurityEntry string `json:"security_entry"`

	// 运行时目录（相对路径会基于 BaseDir 解析）
	DataPath     string `json:"data_path"`
	WebsitePath  string `json:"website_path"`
	DatabasePath string `json:"database_path"`
	AppPath      string `json:"app_path"`
	StaticPath   string `json:"static_path"`

	// 网关 / 镜像配置
	GatewayImage     string   `json:"gateway_image"`
	GatewayHTTPPort  string   `json:"gateway_http_port"`
	GatewayHTTPSPort string   `json:"gateway_https_port"`
	PHPImage         string   `json:"php_image"`
	RegistryMirrors  []string `json:"registry_mirrors"`
}

var (
	AppConfig *Config
	mu        sync.Mutex
)

// DefaultBaseDir 返回默认安装目录。
// 安装脚本与 systemd 单元都会显式设置 UPANEL_BASE_DIR；
// 命令行工具在未设置时会根据可执行文件位置推断。
func DefaultBaseDir() string {
	if v := strings.TrimSpace(os.Getenv("UPANEL_BASE_DIR")); v != "" {
		return filepath.Clean(v)
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		if filepath.Base(dir) == "bin" {
			return filepath.Dir(dir)
		}
		return dir
	}
	return "/opt/upanel"
}

// LoadConfig 加载配置：先读磁盘上的 config.json，再用环境变量覆盖。
func LoadConfig() error {
	mu.Lock()
	defer mu.Unlock()

	base := DefaultBaseDir()
	cfg := NewDefaultConfig(base)

	path := configFilePath(base)
	needSave := false

	// 读取已持久化的配置
	if data, err := os.ReadFile(path); err == nil {
		var disk Config
		if err := json.Unmarshal(data, &disk); err != nil {
			return fmt.Errorf("配置文件 %s 格式错误: %w", path, err)
		}
		_ = os.Chmod(path, 0600)
		if disk.JWTSecret == "" || len(disk.JWTSecret) < 32 {
			needSave = true
		}
		mergeConfig(cfg, &disk)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("读取配置文件失败: %w", err)
	} else {
		// 首次启动：生成并持久化 JWT 密钥，保证重启后会话仍然有效
		needSave = true
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
		if len(v) < 32 {
			log.Printf("警告: JWT_SECRET 少于 32 个字符，已忽略并使用自动生成的密钥")
		} else {
			cfg.JWTSecret = v
		}
	}
	if v := os.Getenv("PANEL_ENTRY"); v != "" {
		cfg.SecurityEntry = v
	}

	cfg.normalize()

	// 安全入口只允许字母、数字、- 与 _，防止路由歧义与路径穿越
	if entry, err := NormalizeSecurityEntry(cfg.SecurityEntry); err != nil {
		log.Printf("警告: 安全入口配置无效（%v），已禁用安全入口", err)
		cfg.SecurityEntry = ""
		needSave = true
	} else {
		if entry != cfg.SecurityEntry {
			needSave = true
		}
		cfg.SecurityEntry = entry
	}

	if cfg.Port == "" || ValidatePort(cfg.Port) != nil {
		log.Printf("警告: 面板端口 %q 无效，回退到 8080", cfg.Port)
		cfg.Port = "8080"
		needSave = true
	}
	if cfg.JWTSecret == "" {
		cfg.JWTSecret = randomString(48)
		needSave = true
	}

	if cfg.BindIP != "" && cfg.BindIP != "localhost" && net.ParseIP(cfg.BindIP) == nil {
		log.Printf("警告: 面板监听地址 %q 无效，回退到 0.0.0.0", cfg.BindIP)
		cfg.BindIP = "0.0.0.0"
		needSave = true
	}
	if ValidatePort(cfg.GatewayHTTPPort) != nil {
		log.Printf("警告: 网关 HTTP 端口 %q 无效，回退到 80", cfg.GatewayHTTPPort)
		cfg.GatewayHTTPPort = "80"
		needSave = true
	}
	if ValidatePort(cfg.GatewayHTTPSPort) != nil {
		log.Printf("警告: 网关 HTTPS 端口 %q 无效，回退到 443", cfg.GatewayHTTPSPort)
		cfg.GatewayHTTPSPort = "443"
		needSave = true
	}
	if cfg.GatewayHTTPPort == cfg.GatewayHTTPSPort {
		log.Printf("警告: 网关 HTTP/HTTPS 端口相同，HTTPS 已回退到 443")
		cfg.GatewayHTTPSPort = "443"
		needSave = true
	}

	AppConfig = cfg

	if err := cfg.EnsureDirs(); err != nil {
		return err
	}
	if needSave {
		if err := cfg.saveLocked(); err != nil {
			return fmt.Errorf("写入配置文件失败: %w", err)
		}
	}
	return nil
}

// NewDefaultConfig 生成一份带默认值的配置
func NewDefaultConfig(base string) *Config {
	if base == "" {
		base = DefaultBaseDir()
	}
	return &Config{
		Port:             "8080",
		BindIP:           "0.0.0.0",
		Mode:             "release",
		BaseDir:          base,
		Username:         "admin",
		JWTSecret:        randomString(48),
		SecurityEntry:    "",
		DataPath:         "data",
		WebsitePath:      "data/websites",
		DatabasePath:     "data/databases",
		AppPath:          "data/apps",
		StaticPath:       "web",
		GatewayImage:     "nginx:alpine",
		GatewayHTTPPort:  "80",
		GatewayHTTPSPort: "443",
		PHPImage:         "php:8.2-fpm",
		RegistryMirrors:  []string{},
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
	if src.SessionVersion > 0 {
		dst.SessionVersion = src.SessionVersion
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
	if c.BaseDir == "" {
		c.BaseDir = DefaultBaseDir()
	}
	c.BaseDir = filepath.Clean(c.BaseDir)
	c.Port = strings.TrimSpace(c.Port)
	c.BindIP = strings.TrimSpace(c.BindIP)
	c.GatewayHTTPPort = strings.TrimSpace(c.GatewayHTTPPort)
	c.GatewayHTTPSPort = strings.TrimSpace(c.GatewayHTTPSPort)
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
	if c.Mode != "debug" {
		c.Mode = "release"
	}
	if c.Username == "" {
		c.Username = "admin"
	}
	if c.JWTSecret == "" || len(c.JWTSecret) < 32 {
		c.JWTSecret = randomString(48)
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
		filepath.Join(c.DataPath, "gateway", "logs"),
		filepath.Join(c.DataPath, "gateway", "html"),
		filepath.Join(c.DataPath, "logs"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", d, err)
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

// GatewayLogDir 网关站点日志目录（宿主机侧）
func (c *Config) GatewayLogDir() string {
	return filepath.Join(c.DataPath, "gateway", "logs")
}

// GatewayHTMLDir 网关默认站点目录（宿主机侧）
func (c *Config) GatewayHTMLDir() string {
	return filepath.Join(c.DataPath, "gateway", "html")
}

// GatewayNginxConf 网关主配置文件（宿主机侧）
func (c *Config) GatewayNginxConf() string {
	return filepath.Join(c.DataPath, "gateway", "nginx.conf")
}

// InitialPasswordFile 首次启动自动生成的管理员密码（仅首次生成时写入）
func (c *Config) InitialPasswordFile() string {
	return filepath.Join(c.DataPath, "initial-password.txt")
}

// Save 持久化配置到磁盘
func (c *Config) Save() error {
	mu.Lock()
	defer mu.Unlock()
	return c.saveLocked()
}

func (c *Config) saveLocked() error {
	c.normalize()
	if err := os.MkdirAll(filepath.Dir(c.ConfigFilePath()), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	// 原子写入，避免断电/并发导致配置损坏
	tmp := c.ConfigFilePath() + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, c.ConfigFilePath()); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// SaveWith 在锁内修改配置并原子落盘
func (c *Config) SaveWith(mutate func(*Config)) error {
	mu.Lock()
	defer mu.Unlock()
	mutate(c)
	return c.saveLocked()
}

// SetPasswordHash 更新密码哈希并落盘
func (c *Config) SetPasswordHash(hash string) error {
	mu.Lock()
	defer mu.Unlock()
	c.PasswordHash = hash
	return c.saveLocked()
}

// SetPort 更新端口并落盘
func (c *Config) SetPort(port string) error {
	if err := ValidatePort(port); err != nil {
		return err
	}
	mu.Lock()
	defer mu.Unlock()
	c.Port = port
	return c.saveLocked()
}

// GetSecurityEntry 获取安全入口路径（已归一化，带前导斜杠）
func (c *Config) GetSecurityEntry() string {
	entry, err := NormalizeSecurityEntry(c.SecurityEntry)
	if err != nil {
		return ""
	}
	return entry
}

// NormalizeSecurityEntry 校验并归一化安全入口。
// 返回值形如 "/mypanel" 或 "/a/b"，空字符串表示未启用。
func NormalizeSecurityEntry(entry string) (string, error) {
	e := strings.TrimSpace(entry)
	e = strings.Trim(e, "/")
	if e == "" {
		return "", nil
	}
	if len(e) > 128 {
		return "", fmt.Errorf("安全入口过长（最多 128 个字符）")
	}
	lower := strings.ToLower(e)
	if lower == "api" || strings.HasPrefix(lower, "api/") || lower == "assets" || strings.HasPrefix(lower, "assets/") {
		return "", fmt.Errorf("安全入口不能使用 api / assets 等保留路径")
	}
	for _, seg := range strings.Split(e, "/") {
		if !securityEntrySegmentRe.MatchString(seg) {
			return "", fmt.Errorf("安全入口只能包含字母、数字、- 和 _，且不能以 - 开头")
		}
	}
	return "/" + e, nil
}

var securityEntrySegmentRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// ValidatePort 校验端口字符串
func ValidatePort(port string) error {
	n, err := strconv.Atoi(strings.TrimSpace(port))
	if err != nil || n < 1 || n > 65535 {
		return fmt.Errorf("端口必须是 1-65535 之间的数字")
	}
	return nil
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
