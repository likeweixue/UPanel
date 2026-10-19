package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"upanel/internal/config"
)

// App 应用商店条目
type App struct {
	Key          string   `json:"key"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	Icon         string   `json:"icon"`
	Versions     []string `json:"versions"`
	DefaultVer   string   `json:"defaultVersion"`
	Port         int      `json:"port"`      // 容器内端口
	DefaultHostPort int    `json:"hostPort"` // 建议宿主机端口
	Installed    bool     `json:"installed"`
	Status       string   `json:"status"`
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
	Name       string                 `json:"name"`
	AppKey     string                 `json:"appKey"`
	Version    string                 `json:"version"`
	Image      string                 `json:"image"`
	Container  string                 `json:"container"`
	HostPort   int                    `json:"hostPort"`
	Port       int                    `json:"port"`
	Status     string                 `json:"status"`
	Path       string                 `json:"path"`
	Config     map[string]interface{} `json:"config"`
	InstalledAt time.Time             `json:"installedAt"`
}

// AppService 应用商店服务
type AppService struct {
	rt  *Runtime
	cfg *config.Config
}

// NewAppService 创建应用服务
func NewAppService() (*AppService, error) {
	rt, err := NewRuntime()
	if err != nil {
		return nil, err
	}
	cfg := config.Current()
	if err := os.MkdirAll(cfg.AppPath, 0755); err != nil {
		rt.Close()
		return nil, fmt.Errorf("创建应用目录失败: %w", err)
	}
	return &AppService{rt: rt, cfg: cfg}, nil
}

func (s *AppService) Close() error {
	return s.rt.Close()
}

// catalog 内置应用目录
func (s *AppService) catalog() []App {
	return []App{
		{Key: "nginx", Name: "Nginx", Category: "web", Description: "高性能 HTTP 与反向代理服务器", Icon: "Monitor", Versions: []string{"1.27-alpine", "1.25-alpine", "latest"}, DefaultVer: "1.27-alpine", Port: 80, DefaultHostPort: 8081},
		{Key: "openresty", Name: "OpenResty", Category: "web", Description: "基于 Nginx 与 Lua 的高性能 Web 平台", Icon: "Monitor", Versions: []string{"alpine", "latest"}, DefaultVer: "alpine", Port: 80, DefaultHostPort: 8082},
		{Key: "mysql", Name: "MySQL", Category: "database", Description: "流行的关系型数据库", Icon: "Coin", Versions: []string{"8.0", "5.7"}, DefaultVer: "8.0", Port: 3306, DefaultHostPort: 3306},
		{Key: "postgres", Name: "PostgreSQL", Category: "database", Description: "强大的开源关系型数据库", Icon: "Coin", Versions: []string{"15", "14"}, DefaultVer: "15", Port: 5432, DefaultHostPort: 5432},
		{Key: "redis", Name: "Redis", Category: "database", Description: "高性能键值数据库", Icon: "Connection", Versions: []string{"7-alpine", "6-alpine"}, DefaultVer: "7-alpine", Port: 6379, DefaultHostPort: 6379},
		{Key: "mariadb", Name: "MariaDB", Category: "database", Description: "MySQL 的开源分支", Icon: "Coin", Versions: []string{"11", "10.11"}, DefaultVer: "11", Port: 3306, DefaultHostPort: 3307},
		{Key: "phpmyadmin", Name: "phpMyAdmin", Category: "tools", Description: "MySQL 可视化管理工具", Icon: "Tools", Versions: []string{"latest"}, DefaultVer: "latest", Port: 80, DefaultHostPort: 8888},
		{Key: "wordpress", Name: "WordPress", Category: "website", Description: "流行的博客与内容管理系统", Icon: "Document", Versions: []string{"latest", "6.4-php8.2-fpm"}, DefaultVer: "latest", Port: 80, DefaultHostPort: 8083},
		{Key: "portainer", Name: "Portainer", Category: "tools", Description: "Docker 可视化管理工具", Icon: "Tools", Versions: []string{"latest"}, DefaultVer: "latest", Port: 9443, DefaultHostPort: 9443},
		{Key: "minio", Name: "MinIO", Category: "storage", Description: "高性能对象存储服务", Icon: "Folder", Versions: []string{"latest"}, DefaultVer: "latest", Port: 9000, DefaultHostPort: 9000},
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

// containerName 应用容器名
func (s *AppService) containerName(name string) string {
	return "upanel-app-" + Slug(name)
}

// InstallApp 安装应用
func (s *AppService) InstallApp(req *InstallAppRequest) (*InstalledApp, error) {
	app, ok := s.find(strings.TrimSpace(req.AppKey))
	if !ok {
		return nil, fmt.Errorf("未知应用: %s", req.AppKey)
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = app.Key
	}
	if strings.ContainsAny(name, "/ .") {
		return nil, fmt.Errorf("应用名称不能包含空格、斜杠或点号")
	}

	if _, err := os.Stat(s.metaPath(name)); err == nil {
		return nil, fmt.Errorf("应用 %s 已安装", name)
	}

	version := strings.TrimSpace(req.Version)
	if version == "" {
		version = app.DefaultVer
	}
	image := app.Key + ":" + version
	if app.Key == "openresty" {
		image = "openresty/openresty:" + version
	}

	// 宿主机端口
	hostPort := app.DefaultHostPort
	if req.Config != nil {
		if v, ok := req.Config["host_port"]; ok {
			hostPort = toInt(v, hostPort)
		} else if v, ok := req.Config["port"]; ok {
			hostPort = toInt(v, hostPort)
		}
	}
	if p := s.rt.HostPortOccupied(hostPort); p && hostPort > 0 {
		hostPort = s.rt.FindFreePort(hostPort)
	}

	appPath := filepath.Join(s.cfg.AppPath, name)
	dataPath := filepath.Join(appPath, "data")
	confPath := filepath.Join(appPath, "conf")
	for _, d := range []string{dataPath, confPath} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("创建应用目录失败: %w", err)
		}
	}

	env := s.buildEnv(app.Key, req.Config)

	if err := s.rt.PullImage(image, nil); err != nil {
		return nil, err
	}

	spec := ContainerSpec{
		Name:    s.containerName(name),
		Image:   image,
		Env:     env,
		Ports:   map[string]string{fmt.Sprintf("%d/tcp", app.Port): fmt.Sprintf("%d", hostPort)},
		Volumes: s.buildVolumes(app.Key, dataPath, confPath),
	}
	if app.Key == "portainer" {
		spec.Volumes = map[string]string{"/var/run/docker.sock": "/var/run/docker.sock", dataPath: "/data"}
	}

	if _, err := s.rt.CreateAndStart(spec); err != nil {
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
		Config:      req.Config,
		InstalledAt: time.Now(),
	}
	if err := s.saveMeta(installed); err != nil {
		return nil, err
	}
	return installed, nil
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

func (s *AppService) buildEnv(key string, cfg map[string]interface{}) []string {
	get := func(k string) string {
		if cfg == nil {
			return ""
		}
		if v, ok := cfg[k]; ok {
			if s, ok := v.(string); ok {
				return strings.TrimSpace(s)
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

	switch key {
	case "mysql", "mariadb":
		return []string{"MYSQL_ROOT_PASSWORD=" + orDefault(get("password"), "upanel123")}
	case "postgres":
		return []string{
			"POSTGRES_PASSWORD=" + orDefault(get("password"), "upanel123"),
			"POSTGRES_USER=" + orDefault(get("username"), "postgres"),
			"POSTGRES_DB=" + orDefault(get("database"), "postgres"),
		}
	case "redis":
		return nil
	case "minio":
		return []string{
			"MINIO_ROOT_USER=" + orDefault(get("username"), "minioadmin"),
			"MINIO_ROOT_PASSWORD=" + orDefault(get("password"), "minio123456"),
		}
	case "phpmyadmin":
		return []string{"PMA_ARBITRARY=1", "UPLOAD_LIMIT=256M"}
	case "wordpress":
		return []string{
			"WORDPRESS_DB_HOST=" + orDefault(get("db_host"), ""),
			"WORDPRESS_DB_USER=" + orDefault(get("db_user"), "wordpress"),
			"WORDPRESS_DB_PASSWORD=" + orDefault(get("db_password"), "wordpress123"),
			"WORDPRESS_DB_NAME=" + orDefault(get("db_name"), "wordpress"),
		}
	}
	return nil
}

func (s *AppService) buildVolumes(key, dataPath, confPath string) map[string]string {
	switch key {
	case "mysql", "mariadb":
		return map[string]string{dataPath: "/var/lib/mysql"}
	case "postgres":
		return map[string]string{dataPath: "/var/lib/postgresql/data"}
	case "redis":
		return map[string]string{dataPath: "/data"}
	case "minio":
		return map[string]string{dataPath: "/data"}
	case "nginx", "openresty":
		return map[string]string{dataPath: "/usr/share/nginx/html", confPath: "/etc/nginx/conf.d"}
	}
	return map[string]string{dataPath: "/data"}
}

func (s *AppService) metaPath(name string) string {
	return filepath.Join(s.cfg.AppPath, name, "app.json")
}

func (s *AppService) saveMeta(a *InstalledApp) error {
	data, err := json.MarshalIndent(a, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath(a.Name), data, 0644)
}

func (s *AppService) loadMeta(name string) (*InstalledApp, error) {
	data, err := os.ReadFile(s.metaPath(name))
	if err != nil {
		return nil, err
	}
	var a InstalledApp
	if err := json.Unmarshal(data, &a); err != nil {
		return nil, err
	}
	return &a, nil
}

// ListInstalled 列出已安装应用
func (s *AppService) ListInstalled() ([]InstalledApp, error) {
	entries, err := os.ReadDir(s.cfg.AppPath)
	if err != nil {
		if os.IsNotExist(err) {
			return []InstalledApp{}, nil
		}
		return nil, err
	}
	var list []InstalledApp
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		a, err := s.loadMeta(e.Name())
		if err != nil {
			continue
		}
		a.Status = s.rt.ContainerState(a.Container)
		if a.Status == "" {
			a.Status = "missing"
		}
		list = append(list, *a)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list, nil
}

// StartApp / StopApp / RestartApp
func (s *AppService) StartApp(name string) error {
	a, err := s.loadMeta(name)
	if err != nil {
		return fmt.Errorf("应用 %s 未安装", name)
	}
	return s.rt.Start(a.Container)
}

func (s *AppService) StopApp(name string) error {
	a, err := s.loadMeta(name)
	if err != nil {
		return fmt.Errorf("应用 %s 未安装", name)
	}
	return s.rt.Stop(a.Container)
}

func (s *AppService) RestartApp(name string) error {
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
