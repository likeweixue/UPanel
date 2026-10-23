package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"upanel/internal/config"
)

// App 应用商店条目
type App struct {
	Key             string   `json:"key"`
	Name            string   `json:"name"`
	Category        string   `json:"category"`
	Description     string   `json:"description"`
	Icon            string   `json:"icon"`
	Versions        []string `json:"versions"`
	DefaultVer      string   `json:"defaultVersion"`
	Port            int      `json:"port"`     // 容器内端口
	DefaultHostPort int      `json:"hostPort"` // 建议宿主机端口
	Installed       bool     `json:"installed"`
	Status          string   `json:"status"`
}

// InstallAppRequest 安装应用请求
type InstallAppRequest struct {
	AppKey  string                 `json:"app_key"`
	Version string                 `json:"version"`
	Name    string                 `json:"name"`
	Config  map[string]interface{} `json:"config"`
}

// InstalledApp 已安装应用
type InstalledApp struct {
	Name        string                 `json:"name"`
	AppKey      string                 `json:"appKey"`
	Version     string                 `json:"version"`
	Image       string                 `json:"image"`
	Container   string                 `json:"container"`
	HostPort    int                    `json:"hostPort"`
	Port        int                    `json:"port"`
	Status      string                 `json:"status"`
	Path        string                 `json:"path"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Credentials map[string]string      `json:"credentials,omitempty"`
	InstalledAt time.Time              `json:"installedAt"`
}

// AppService 应用商店服务
type AppService struct {
	rt  *Runtime
	cfg *config.Config
}

// NewAppService 创建应用服务
func NewAppService() (*AppService, error) {
	cfg := config.Current()
	if err := os.MkdirAll(cfg.AppPath, 0755); err != nil {
		return nil, fmt.Errorf("创建应用目录失败: %w", err)
	}
	s := &AppService{cfg: cfg}
	rt, err := NewRuntime()
	if err != nil {
		return s, err
	}
	s.rt = rt
	return s, nil
}

func (s *AppService) dockerReady() error {
	if s.rt == nil {
		return fmt.Errorf("Docker 未连接，无法执行该操作")
	}
	return nil
}

func (s *AppService) Close() error {
	if s.rt == nil {
		return nil
	}
	return s.rt.Close()
}

// catalog 内置应用目录
func (s *AppService) catalog() []App {
	return []App{
		{Key: "nginx", Name: "Nginx", Category: "web", Description: "高性能 HTTP 与反向代理服务器", Icon: "server", Versions: []string{"1.27-alpine", "1.25-alpine", "latest"}, DefaultVer: "1.27-alpine", Port: 80, DefaultHostPort: 8081},
		{Key: "openresty", Name: "OpenResty", Category: "web", Description: "基于 Nginx 与 Lua 的高性能 Web 平台", Icon: "server", Versions: []string{"alpine", "latest"}, DefaultVer: "alpine", Port: 80, DefaultHostPort: 8082},
		{Key: "mysql", Name: "MySQL", Category: "database", Description: "流行的关系型数据库", Icon: "database", Versions: []string{"8.0", "5.7"}, DefaultVer: "8.0", Port: 3306, DefaultHostPort: 3306},
		{Key: "postgres", Name: "PostgreSQL", Category: "database", Description: "强大的开源关系型数据库", Icon: "database", Versions: []string{"15", "14"}, DefaultVer: "15", Port: 5432, DefaultHostPort: 5432},
		{Key: "redis", Name: "Redis", Category: "database", Description: "高性能键值数据库", Icon: "database", Versions: []string{"7-alpine", "6-alpine"}, DefaultVer: "7-alpine", Port: 6379, DefaultHostPort: 6379},
		{Key: "mariadb", Name: "MariaDB", Category: "database", Description: "MySQL 的开源分支", Icon: "database", Versions: []string{"11", "10.11"}, DefaultVer: "11", Port: 3306, DefaultHostPort: 3307},
		{Key: "phpmyadmin", Name: "phpMyAdmin", Category: "tools", Description: "MySQL 可视化管理工具", Icon: "tool", Versions: []string{"latest"}, DefaultVer: "latest", Port: 80, DefaultHostPort: 8888},
		{Key: "wordpress", Name: "WordPress", Category: "website", Description: "流行的博客与内容管理系统（需要已有 MySQL）", Icon: "globe", Versions: []string{"latest"}, DefaultVer: "latest", Port: 80, DefaultHostPort: 8083},
		{Key: "portainer", Name: "Portainer", Category: "tools", Description: "Docker 可视化管理工具（仅本机访问）", Icon: "tool", Versions: []string{"latest"}, DefaultVer: "latest", Port: 9443, DefaultHostPort: 9443},
		{Key: "minio", Name: "MinIO", Category: "storage", Description: "高性能对象存储服务", Icon: "folder", Versions: []string{"latest"}, DefaultVer: "latest", Port: 9000, DefaultHostPort: 9000},
	}
}

// GetApps 获取应用列表（含安装状态）
func (s *AppService) GetApps() []App {
	installed, _ := s.ListInstalled()
	byKey := map[string]*InstalledApp{}
	for i := range installed {
		byKey[installed[i].AppKey] = &installed[i]
	}
	list := s.catalog()
	for i := range list {
		if a, ok := byKey[list[i].Key]; ok {
			list[i].Installed = true
			list[i].Status = a.Status
		} else {
			list[i].Status = "not_installed"
		}
	}
	return list
}

func (s *AppService) find(key string) (App, bool) {
	for _, a := range s.catalog() {
		if a.Key == key {
			return a, true
		}
	}
	return App{}, false
}

func appImage(key, version string) string {
	switch key {
	case "openresty":
		return "openresty/openresty:" + version
	case "minio":
		return "minio/minio:" + version
	case "portainer":
		return "portainer/portainer-ce:" + version
	default:
		return key + ":" + version
	}
}

// containerName 应用容器名
func (s *AppService) containerName(name string) string {
	return "upanel-app-" + Slug(name)
}

var appNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,31}$`)

// InstallApp 安装应用
func (s *AppService) InstallApp(req *InstallAppRequest) (*InstalledApp, error) {
	if err := s.dockerReady(); err != nil {
		return nil, err
	}
	app, ok := s.find(strings.TrimSpace(req.AppKey))
	if !ok {
		return nil, fmt.Errorf("未知应用: %s", req.AppKey)
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = app.Key
	}
	if !appNameRe.MatchString(name) {
		return nil, fmt.Errorf("应用名称只能包含字母、数字、- 和 _，且必须以字母开头（最多 32 位）")
	}

	if _, err := os.Stat(s.metaPath(name)); err == nil {
		return nil, fmt.Errorf("应用 %s 已安装", name)
	}

	version := strings.TrimSpace(req.Version)
	if version == "" {
		version = app.DefaultVer
	}
	versionOK := false
	for _, v := range app.Versions {
		if v == version {
			versionOK = true
			break
		}
	}
	if !versionOK {
		return nil, fmt.Errorf("不支持的版本 %s，可选: %s", version, strings.Join(app.Versions, ", "))
	}
	image := appImage(app.Key, version)

	// 宿主机端口：只绑定 127.0.0.1，避免数据库/管理工具直接暴露公网
	hostPort := app.DefaultHostPort
	if req.Config != nil {
		if v, ok := req.Config["host_port"]; ok {
			hostPort = toInt(v, hostPort)
		} else if v, ok := req.Config["port"]; ok {
			hostPort = toInt(v, hostPort)
		}
	}
	if hostPort < 1 || hostPort > 65535 {
		return nil, fmt.Errorf("宿主机端口必须是 1-65535 之间的数字")
	}
	if s.rt.HostPortOccupied(hostPort) {
		hostPort = s.rt.FindFreePort(hostPort)
		if hostPort == 0 {
			return nil, fmt.Errorf("找不到可用的宿主机端口")
		}
	}

	appPath := filepath.Join(s.cfg.AppPath, name)
	appPathExisted := dirExists(appPath)
	dataPath := filepath.Join(appPath, "data")
	confPath := filepath.Join(appPath, "conf")
	for _, d := range []string{dataPath, confPath} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("创建应用目录失败: %w", err)
		}
	}
	switch app.Key {
	case "redis":
		prepareRedisDir(dataPath)
	case "mysql", "mariadb":
		prepareMySQLDir(dataPath)
	case "postgres":
		preparePostgresDir(dataPath)
	case "minio":
		prepareMinioDir(dataPath)
	case "wordpress":
		prepareWordpressDir(dataPath)
	}
	if app.Key == "nginx" || app.Key == "openresty" {
		if err := ensureWebAppDefaults(dataPath, confPath); err != nil {
			if !appPathExisted {
				_ = os.RemoveAll(appPath)
			}
			return nil, err
		}
	}

	env, cmd, credentials, err := s.buildRuntime(app, req.Config)
	if err != nil {
		if !appPathExisted {
			_ = os.RemoveAll(appPath)
		}
		return nil, err
	}

	if err := s.rt.PullImage(image, nil); err != nil {
		if !appPathExisted {
			_ = os.RemoveAll(appPath)
		}
		return nil, err
	}

	spec := ContainerSpec{
		Name:          s.containerName(name),
		Image:         image,
		Env:           env,
		Cmd:           cmd,
		Ports:         map[string]string{fmt.Sprintf("%d/tcp", app.Port): fmt.Sprintf("%d", hostPort)},
		Volumes:       s.buildVolumes(app.Key, dataPath, confPath),
		BindLocalhost: true,
		Labels:        map[string]string{"managed-by": "upanel", "upanel-app": name},
	}
	if app.Key == "portainer" {
		spec.Volumes = map[string]string{"/var/run/docker.sock": "/var/run/docker.sock", dataPath: "/data"}
	}
	if app.Key == "minio" {
		spec.Cmd = []string{"server", "/data", "--address", ":9000"}
	}

	if _, err := s.rt.CreateAndStart(spec); err != nil {
		if !appPathExisted {
			_ = os.RemoveAll(appPath)
		}
		return nil, err
	}

	installed := &InstalledApp{
		Name:        name,
		AppKey:      app.Key,
		Version:     version,
		Image:       image,
		Container:   s.containerName(name),
		HostPort:    hostPort,
		Port:        app.Port,
		Status:      "running",
		Path:        appPath,
		Config:      sanitizeAppConfig(req.Config),
		Credentials: credentials,
		InstalledAt: time.Now(),
	}
	if err := s.saveMeta(installed); err != nil {
		_ = s.rt.Remove(spec.Name)
		if !appPathExisted {
			_ = os.RemoveAll(appPath)
		}
		return nil, err
	}
	return installed, nil
}

// sanitizeAppConfig 移除不应持久化的敏感字段
func sanitizeAppConfig(cfg map[string]interface{}) map[string]interface{} {
	if cfg == nil {
		return nil
	}
	out := map[string]interface{}{}
	for k, v := range cfg {
		lk := strings.ToLower(k)
		if strings.Contains(lk, "password") || strings.Contains(lk, "secret") || strings.Contains(lk, "token") {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func toInt(v interface{}, def int) int {
	switch t := v.(type) {
	case float64:
		if t > 0 {
			return int(t)
		}
	case int:
		if t > 0 {
			return t
		}
	case string:
		var i int
		if _, err := fmt.Sscanf(t, "%d", &i); err == nil && i > 0 {
			return i
		}
	}
	return def
}

// buildRuntime 生成环境变量/命令与自动生成的凭据
func (s *AppService) buildRuntime(app App, cfg map[string]interface{}) ([]string, []string, map[string]string, error) {
	get := func(k string) string {
		if cfg == nil {
			return ""
		}
		if v, ok := cfg[k]; ok {
			if str, ok := v.(string); ok {
				return strings.TrimSpace(str)
			}
		}
		return ""
	}
	orDefault := func(v, def string) string {
		if v != "" {
			return v
		}
		return def
	}
	random := config.RandomPassword

	credentials := map[string]string{}
	switch app.Key {
	case "mysql", "mariadb":
		password := orDefault(get("password"), random(20))
		credentials["username"] = "root"
		credentials["password"] = password
		return []string{"MYSQL_ROOT_PASSWORD=" + password}, nil, credentials, nil
	case "postgres":
		password := orDefault(get("password"), random(20))
		user := orDefault(get("username"), "postgres")
		credentials["username"] = user
		credentials["password"] = password
		return []string{
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_USER=" + user,
			"POSTGRES_DB=" + orDefault(get("database"), "postgres"),
		}, nil, credentials, nil
	case "redis":
		password := orDefault(get("password"), random(20))
		credentials["password"] = password
		return nil, []string{"redis-server", "--requirepass", password, "--appendonly", "yes"}, credentials, nil
	case "minio":
		user := orDefault(get("username"), "minioadmin")
		password := orDefault(get("password"), random(20))
		credentials["username"] = user
		credentials["password"] = password
		return []string{
			"MINIO_ROOT_USER=" + user,
			"MINIO_ROOT_PASSWORD=" + password,
		}, nil, credentials, nil
	case "phpmyadmin":
		return []string{"PMA_ARBITRARY=1", "UPLOAD_LIMIT=256M"}, nil, credentials, nil
	case "wordpress":
		dbHost := get("db_host")
		if dbHost == "" {
			return nil, nil, nil, fmt.Errorf("WordPress 需要填写 db_host（MySQL 容器名或地址）")
		}
		dbPassword := get("db_password")
		if dbPassword == "" {
			return nil, nil, nil, fmt.Errorf("WordPress 需要填写已有 MySQL 的数据库密码")
		}
		credentials["db_password"] = dbPassword
		return []string{
			"WORDPRESS_DB_HOST=" + dbHost,
			"WORDPRESS_DB_USER=" + orDefault(get("db_user"), "wordpress"),
			"WORDPRESS_DB_PASSWORD=" + dbPassword,
			"WORDPRESS_DB_NAME=" + orDefault(get("db_name"), "wordpress"),
		}, nil, credentials, nil
	}
	return nil, nil, credentials, nil
}

func (s *AppService) buildVolumes(key, dataPath, confPath string) map[string]string {
	switch key {
	case "mysql", "mariadb":
		return map[string]string{dataPath: "/var/lib/mysql"}
	case "postgres":
		return map[string]string{dataPath: "/var/lib/postgresql/data"}
	case "redis", "minio":
		return map[string]string{dataPath: "/data"}
	case "nginx", "openresty":
		return map[string]string{dataPath: "/usr/share/nginx/html", confPath: "/etc/nginx/conf.d"}
	case "wordpress":
		return map[string]string{dataPath: "/var/www/html"}
	}
	return map[string]string{dataPath: "/data"}
}

func (s *AppService) metaPath(name string) string {
	if !appNameRe.MatchString(name) {
		return ""
	}
	return filepath.Join(s.cfg.AppPath, name, "app.json")
}

func (s *AppService) saveMeta(a *InstalledApp) error {
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(s.cfg.AppPath, a.Name), 0755); err != nil {
		return err
	}
	return writeFileAtomic(s.metaPath(a.Name), data, 0600)
}

func (s *AppService) loadMeta(name string) (*InstalledApp, error) {
	p := s.metaPath(name)
	if p == "" {
		return nil, fmt.Errorf("应用名称不合法")
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, err
	}
	var a InstalledApp
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// ListInstalled 列出已安装应用（不返回凭据）
func (s *AppService) ListInstalled() ([]InstalledApp, error) {
	entries, err := os.ReadDir(s.cfg.AppPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []InstalledApp{}, nil
		}
		return nil, err
	}
	list := make([]InstalledApp, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		a, err := s.loadMeta(e.Name())
		if err != nil {
			continue
		}
		if s.rt == nil {
			a.Status = "docker_unavailable"
		} else {
			a.Status = s.rt.ContainerState(a.Container)
			if a.Status == "" {
				a.Status = "missing"
			}
		}
		a.Credentials = nil
		list = append(list, *a)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list, nil
}

// GetCredentials 读取应用凭据（仅登录后可访问）
func (s *AppService) GetCredentials(name string) (map[string]string, error) {
	a, err := s.loadMeta(name)
	if err != nil {
		return nil, fmt.Errorf("应用 %s 未安装", name)
	}
	if a.Credentials == nil {
		a.Credentials = map[string]string{}
	}
	return a.Credentials, nil
}

// StartApp / StopApp / RestartApp
func (s *AppService) StartApp(name string) error {
	if err := s.dockerReady(); err != nil {
		return err
	}
	a, err := s.loadMeta(name)
	if err != nil {
		return fmt.Errorf("应用 %s 未安装", name)
	}
	if s.rt.ContainerState(a.Container) == "" {
		return fmt.Errorf("容器 %s 已不存在，请卸载后重新安装", a.Container)
	}
	return s.rt.Start(a.Container)
}

func (s *AppService) StopApp(name string) error {
	if err := s.dockerReady(); err != nil {
		return err
	}
	a, err := s.loadMeta(name)
	if err != nil {
		return fmt.Errorf("应用 %s 未安装", name)
	}
	if s.rt.ContainerState(a.Container) == "" {
		return fmt.Errorf("容器 %s 已不存在", a.Container)
	}
	return s.rt.Stop(a.Container)
}

func (s *AppService) RestartApp(name string) error {
	if err := s.dockerReady(); err != nil {
		return err
	}
	a, err := s.loadMeta(name)
	if err != nil {
		return fmt.Errorf("应用 %s 未安装", name)
	}
	if err := s.rt.Stop(a.Container); err != nil {
		return err
	}
	return s.rt.Start(a.Container)
}

// UninstallApp 卸载应用（keepData=true 保留数据目录）
func (s *AppService) UninstallApp(name string, keepData bool) error {
	if err := s.dockerReady(); err != nil {
		return err
	}
	a, err := s.loadMeta(name)
	if err != nil {
		return fmt.Errorf("应用 %s 未安装", name)
	}
	if err := s.rt.Remove(a.Container); err != nil {
		return fmt.Errorf("删除容器失败: %w", err)
	}
	if keepData {
		return os.Remove(s.metaPath(name))
	}
	return os.RemoveAll(filepath.Join(s.cfg.AppPath, name))
}

// ensureWebAppDefaults 为独立的 Nginx/OpenResty 实例写入最小可用配置，
// 否则空的 conf.d 目录会导致容器虽然启动但并不监听 80 端口。
func ensureWebAppDefaults(dataPath, confPath string) error {
	indexPath := filepath.Join(dataPath, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		page := `<!DOCTYPE html><html lang="zh-CN"><head><meta charset="utf-8">
<title>UPanel App</title></head><body style="font-family:sans-serif;background:#f4f5f7;color:#1f2937">
<div style="max-width:520px;margin:12vh auto;padding:32px;background:#fff;border:1px solid #e5e7eb;border-radius:12px;text-align:center">
<h1 style="color:#2f6fed">Nginx 应用已就绪</h1>
<p>把静态文件上传到 data/apps/&lt;实例名&gt;/data 目录即可。</p>
</div></body></html>`
		if err := os.WriteFile(indexPath, []byte(page), 0644); err != nil {
			return fmt.Errorf("写入默认首页失败: %w", err)
		}
	}
	confFile := filepath.Join(confPath, "default.conf")
	if _, err := os.Stat(confFile); os.IsNotExist(err) {
		conf := `server {
    listen 80 default_server;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;
    location / { try_files $uri $uri/ =404; }
}
`
		if err := os.WriteFile(confFile, []byte(conf), 0644); err != nil {
			return fmt.Errorf("写入默认站点配置失败: %w", err)
		}
	}
	return nil
}

// ValidateAppName 供 handler 校验路径中的应用名
func ValidateAppName(name string) error {
	if !appNameRe.MatchString(name) {
		return fmt.Errorf("应用名称只能包含字母、数字、- 和 _，且必须以字母开头")
	}
	return nil
}
