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

// SiteType 网站类型
const (
	SiteTypeStatic = "static" // 纯静态：由网关直接托管文件，无需容器
	SiteTypePHP    = "php"    // PHP：php-fpm 容器 + 网关 fastcgi
	SiteTypeProxy  = "proxy"  // 反代：任意镜像容器 + 网关 proxy_pass
)

// Website 网站实例
type Website struct {
	Domain     string    `json:"domain"`
	Aliases    []string  `json:"aliases"`
	Type       string    `json:"type"`
	PHPVersion string    `json:"phpVersion"`
	Image      string    `json:"image"`
	Port       int       `json:"port"`
	Status     string    `json:"status"`
	Path       string    `json:"path"`
	RootDir    string    `json:"rootDir"`
	SSLEnabled bool      `json:"sslEnabled"`
	ForceSSL   bool      `json:"forceSSL"`
	Remark     string    `json:"remark"`
	CreatedAt  time.Time `json:"createdAt"`
}

// CreateWebsiteRequest 创建网站请求
type CreateWebsiteRequest struct {
	Domain     string   `json:"domain" binding:"required"`
	Aliases    []string `json:"aliases"`
	Type       string   `json:"type"`       // static / php / proxy
	PHPVersion string   `json:"phpVersion"` // 8.2 / 8.1 / 7.4
	Image      string   `json:"image"`      // proxy 类型使用的镜像
	Port       int      `json:"port"`       // proxy 类型容器内端口
	Remark     string   `json:"remark"`
	// 兼容旧字段
	CreateDB   bool   `json:"create_db"`
	DBName     string `json:"db_name"`
	DBPassword string `json:"db_password"`
}

// WebsiteService 网站管理服务
type WebsiteService struct {
	rt  *Runtime
	cfg *config.Config
	gw  *GatewayService
}

// NewWebsiteService 创建网站服务
func NewWebsiteService() (*WebsiteService, error) {
	rt, err := NewRuntime()
	if err != nil {
		return nil, err
	}
	cfg := config.Current()
	if err := os.MkdirAll(cfg.WebsitePath, 0755); err != nil {
		rt.Close()
		return nil, fmt.Errorf("创建网站目录失败: %w", err)
	}
	return &WebsiteService{
		rt:  rt,
		cfg: cfg,
		gw:  NewGatewayServiceWith(rt),
	}, nil
}

func (s *WebsiteService) Close() error {
	return s.rt.Close()
}

// phpContainerName PHP-FPM 容器名
func phpContainerName(domain string) string {
	return "upanel-php-" + Slug(domain)
}

// siteContainerName 反代站点容器名
func siteContainerName(domain string) string {
	return "upanel-site-" + Slug(domain)
}

var domainRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]*[a-zA-Z0-9])?)*$`)

// ValidateDomain 校验域名合法性，支持 *.example.com 形式的通配域名
func ValidateDomain(domain string) error {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return fmt.Errorf("域名不能为空")
	}
	if len(domain) > 253 {
		return fmt.Errorf("域名过长")
	}
	if strings.Contains(domain, "/") || strings.Contains(domain, " ") {
		return fmt.Errorf("域名包含非法字符")
	}

	// 通配域名：仅允许前缀 *.
	target := domain
	if strings.HasPrefix(domain, "*.") {
		target = strings.TrimPrefix(domain, "*.")
	}
	if target == "" || strings.HasPrefix(target, ".") || strings.HasSuffix(target, ".") {
		return fmt.Errorf("域名格式不合法: %s", domain)
	}
	if !domainRe.MatchString(target) {
		return fmt.Errorf("域名格式不合法: %s", domain)
	}
	return nil
}

// SiteDir 站点数据目录
func (s *WebsiteService) SiteDir(domain string) string {
	return filepath.Join(s.cfg.WebsitePath, domain)
}

// RootDir 站点根目录（网站文件存放处）
func (s *WebsiteService) RootDir(domain string) string {
	return filepath.Join(s.SiteDir(domain), "html")
}

// CreateWebsite 创建网站
func (s *WebsiteService) CreateWebsite(req *CreateWebsiteRequest) (*Website, error) {
	domain := strings.ToLower(strings.TrimSpace(req.Domain))
	if err := ValidateDomain(domain); err != nil {
		return nil, err
	}

	siteDir := s.SiteDir(domain)
	rootDir := s.RootDir(domain)

	// 站点类型归一
	siteType := strings.ToLower(strings.TrimSpace(req.Type))
	if siteType == "" {
		// 兼容旧调用：带 php_version 的默认走 PHP
		if req.PHPVersion != "" {
			siteType = SiteTypePHP
		} else {
			siteType = SiteTypeStatic
		}
	}
	switch siteType {
	case SiteTypeStatic, SiteTypePHP, SiteTypeProxy:
	default:
		return nil, fmt.Errorf("不支持的网站类型: %s（可选 static / php / proxy）", siteType)
	}

	if _, err := os.Stat(filepath.Join(siteDir, "site.json")); err == nil {
		return nil, fmt.Errorf("网站 %s 已存在", domain)
	}

	// 1. 目录结构
	logDir := filepath.Join(siteDir, "logs")
	for _, d := range []string{rootDir, logDir, filepath.Join(siteDir, "conf")} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("创建站点目录失败: %w", err)
		}
	}

	// 2. 默认首页
	if err := s.writeDefaultIndex(siteType, rootDir, domain); err != nil {
		return nil, err
	}

	// 3. 确保网关就绪
	if err := s.gw.Ensure(nil); err != nil {
		return nil, err
	}

	site := &Website{
		Domain:  domain,
		Aliases: req.Aliases,
		Type:    siteType,
		Path:    siteDir,
		RootDir: rootDir,
		Remark:  req.Remark,
	}

	// 4. 按类型准备运行时
	switch siteType {
	case SiteTypePHP:
		version := strings.TrimSpace(req.PHPVersion)
		if version == "" {
			version = phpVersionFromImage(s.cfg.PHPImage)
		}
		image := "php:" + version + "-fpm"
		if err := s.rt.PullImage(image, nil); err != nil {
			return nil, err
		}
		if _, err := s.rt.CreateAndStart(ContainerSpec{
			Name:  phpContainerName(domain),
			Image: image,
			Volumes: map[string]string{
				rootDir: "/var/www/html",
			},
		}); err != nil {
			return nil, err
		}
		site.PHPVersion = version
		site.Image = image

	case SiteTypeProxy:
		image := strings.TrimSpace(req.Image)
		if image == "" {
			return nil, fmt.Errorf("反代类型必须提供镜像名称")
		}
		port := req.Port
		if port == 0 {
			port = 80
		}
		if err := s.rt.PullImage(image, nil); err != nil {
			return nil, err
		}
		if _, err := s.rt.CreateAndStart(ContainerSpec{
			Name:  siteContainerName(domain),
			Image: image,
			Volumes: map[string]string{
				rootDir: "/var/www/html",
			},
		}); err != nil {
			return nil, err
		}
		site.Image = image
		site.Port = port
	}

	site.Status = "running"
	site.CreatedAt = time.Now()

	// 5. 写入网关配置
	if err := s.gw.WriteSiteConf(domain, s.renderConf(site)); err != nil {
		return nil, err
	}

	// 6. 落盘元数据（必须最后一步，避免半初始化状态被当成正常站点）
	if err := s.saveMeta(site); err != nil {
		return nil, err
	}

	return site, nil
}

func phpVersionFromImage(image string) string {
	// php:8.2-fpm -> 8.2
	img := strings.TrimPrefix(image, "php:")
	img = strings.TrimSuffix(img, "-fpm")
	parts := strings.Split(img, "-")
	if len(parts) > 0 && parts[0] != "" {
		return parts[0]
	}
	return "8.2"
}

func (s *WebsiteService) writeDefaultIndex(siteType, rootDir, domain string) error {
	var name, content string
	switch siteType {
	case SiteTypePHP:
		name = "index.php"
		content = fmt.Sprintf(`<?php
echo "<h1>Welcome to %s</h1>";
echo "<p>Powered by UPanel</p>";
echo "<p>PHP Version: " . phpversion() . "</p>";
echo "<p>Server Time: " . date('Y-m-d H:i:s') . "</p>";
`, domain)
	default:
		name = "index.html"
		content = fmt.Sprintf(`<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Welcome to %s</title>
<style>
  body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;
       background:#0f172a;color:#e2e8f0;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;}
  .box{text-align:center;padding:40px}
  h1{font-size:26px;margin:0 0 10px;color:#38bdf8;font-weight:600}
  p{margin:6px 0;color:#94a3b8;font-size:14px}
</style>
</head>
<body>
  <div class="box">
    <h1>Welcome to %s</h1>
    <p>站点已创建成功，把你的网页文件上传到网站根目录即可。</p>
    <p>Powered by UPanel</p>
  </div>
</body>
</html>
`, domain, domain)
	}
	return os.WriteFile(filepath.Join(rootDir, name), []byte(content), 0644)
}

// serverNames 生成 server_name 指令内容
func serverNames(site *Website) string {
	names := []string{site.Domain}
	for _, a := range site.Aliases {
		a = strings.TrimSpace(a)
		if a != "" && a != site.Domain {
			names = append(names, a)
		}
	}
	return strings.Join(names, " ")
}

// renderConf 渲染 Nginx 站点配置
func (s *WebsiteService) renderConf(site *Website) string {
	// 网关容器内的站点根路径
	root := fmt.Sprintf("/var/www/websites/%s/html", site.Domain)
	names := serverNames(site)
	logPrefix := fmt.Sprintf("/var/log/nginx/%s", site.Domain)

	var b strings.Builder

	// ---- HTTP ----
	if site.ForceSSL && site.SSLEnabled {
		b.WriteString(fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    return 301 https://$host$request_uri;
}
`, names))
	}

	listen := "listen 80;"
	if site.SSLEnabled && !site.ForceSSL {
		listen = "listen 80;"
	}

	b.WriteString(fmt.Sprintf(`
# UPanel managed site: %s
server {
    %s
    server_name %s;

    root %s;
    index %s;

    access_log %s.access.log;
    error_log  %s.error.log;
`, site.Domain, listen, names, root, s.indexLine(site), logPrefix, logPrefix))

	switch site.Type {
	case SiteTypePHP:
		b.WriteString(fmt.Sprintf(`
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        try_files $uri =404;
        fastcgi_split_path_info ^(.+\.php)(/.+)$;
        set $upanel_php "%s:9000";
        fastcgi_pass $upanel_php;
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_param PATH_INFO $fastcgi_path_info;
        fastcgi_read_timeout 300;
    }

    location ~ /\.ht {
        deny all;
    }
`, phpContainerName(site.Domain)))

	case SiteTypeProxy:
		b.WriteString(fmt.Sprintf(`
    location / {
        set $upanel_backend "http://%s:%d";
        proxy_pass $upanel_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 300;
    }
`, siteContainerName(site.Domain), site.Port))

	default: // static
		b.WriteString(`
    location / {
        try_files $uri $uri/ =404;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff2?)$ {
        expires 7d;
        access_log off;
    }
`)
	}

	b.WriteString("}\n")

	// ---- HTTPS ----
	if site.SSLEnabled {
		certDir := fmt.Sprintf("/etc/nginx/certs/%s", site.Domain)
		b.WriteString(fmt.Sprintf(`
server {
    listen 443 ssl http2;
    server_name %s;

    ssl_certificate     %s/fullchain.pem;
    ssl_certificate_key %s/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;

    root %s;
    index %s;

    access_log %s.ssl.access.log;
    error_log  %s.ssl.error.log;
`, names, certDir, certDir, root, s.indexLine(site), logPrefix, logPrefix))

		switch site.Type {
		case SiteTypePHP:
			b.WriteString(fmt.Sprintf(`
    location / {
        try_files $uri $uri/ /index.php?$query_string;
    }

    location ~ \.php$ {
        try_files $uri =404;
        fastcgi_split_path_info ^(.+\.php)(/.+)$;
        set $upanel_php_ssl "%s:9000";
        fastcgi_pass $upanel_php_ssl;
        fastcgi_index index.php;
        include fastcgi_params;
        fastcgi_param HTTPS on;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        fastcgi_param PATH_INFO $fastcgi_path_info;
        fastcgi_read_timeout 300;
    }

    location ~ /\.ht {
        deny all;
    }
`, phpContainerName(site.Domain)))
		case SiteTypeProxy:
			b.WriteString(fmt.Sprintf(`
    location / {
        set $upanel_backend_ssl "http://%s:%d";
        proxy_pass $upanel_backend_ssl;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_read_timeout 300;
    }
`, siteContainerName(site.Domain), site.Port))
		default:
			b.WriteString(`
    location / {
        try_files $uri $uri/ =404;
    }
`)
		}
		b.WriteString("}\n")
	}

	return b.String()
}

func (s *WebsiteService) indexLine(site *Website) string {
	if site.Type == SiteTypePHP {
		return "index.php index.html index.htm"
	}
	return "index.html index.htm index.php"
}

// metaPath 元数据文件路径
func (s *WebsiteService) metaPath(domain string) string {
	return filepath.Join(s.SiteDir(domain), "site.json")
}

func (s *WebsiteService) saveMeta(site *Website) error {
	data, err := json.MarshalIndent(site, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath(site.Domain), data, 0644)
}

func (s *WebsiteService) loadMeta(domain string) (*Website, error) {
	data, err := os.ReadFile(s.metaPath(domain))
	if err != nil {
		return nil, err
	}
	var w Website
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

// GetWebsite 获取单个网站（含实时状态）
func (s *WebsiteService) GetWebsite(domain string) (*Website, error) {
	w, err := s.loadMeta(domain)
	if err != nil {
		return nil, fmt.Errorf("网站 %s 不存在", domain)
	}
	s.fillStatus(w)
	return w, nil
}

// fillStatus 计算实时运行状态
func (s *WebsiteService) fillStatus(w *Website) {
	gwState := s.rt.ContainerState(GatewayName)
	confExists := fileExists(filepath.Join(s.cfg.GatewayConfDir(), SiteConfName(w.Domain)))

	switch w.Type {
	case SiteTypeStatic:
		if gwState == "running" && confExists {
			w.Status = "running"
		} else {
			w.Status = "stopped"
		}
	default:
		name := w.containerName()
		state := s.rt.ContainerState(name)
		if state == "running" && gwState == "running" && confExists {
			w.Status = "running"
		} else if state == "" {
			w.Status = "stopped"
		} else {
			w.Status = "stopped"
		}
	}
}

func (w *Website) containerName() string {
	if w.Type == SiteTypePHP {
		return phpContainerName(w.Domain)
	}
	if w.Type == SiteTypeProxy {
		return siteContainerName(w.Domain)
	}
	return ""
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// ListWebsites 列出所有网站
func (s *WebsiteService) ListWebsites() ([]Website, error) {
	entries, err := os.ReadDir(s.cfg.WebsitePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Website{}, nil
		}
		return nil, err
	}

	// 网关可能尚未启动，避免每个站点都重复查询
	var list []Website
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		w, err := s.loadMeta(e.Name())
		if err != nil {
			continue
		}
		s.fillStatus(w)
		list = append(list, *w)
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].Domain < list[j].Domain
	})
	return list, nil
}

// StartWebsite 启动网站
func (s *WebsiteService) StartWebsite(domain string) error {
	w, err := s.loadMeta(domain)
	if err != nil {
		return fmt.Errorf("网站 %s 不存在", domain)
	}

	if err := s.gw.Ensure(nil); err != nil {
		return err
	}

	if name := w.containerName(); name != "" {
		state := s.rt.ContainerState(name)
		if state == "" {
			// 容器被外部删除，按元数据重建
			if _, err := s.rt.CreateAndStart(s.buildSpec(w)); err != nil {
				return err
			}
		} else if state != "running" {
			if err := s.rt.Start(name); err != nil {
				return fmt.Errorf("启动容器失败: %w", err)
			}
		}
	}

	if err := s.gw.WriteSiteConf(domain, s.renderConf(w)); err != nil {
		return err
	}
	w.Status = "running"
	return s.saveMeta(w)
}

// StopWebsite 停止网站（移除网关配置，避免访问时出现 502）
func (s *WebsiteService) StopWebsite(domain string) error {
	w, err := s.loadMeta(domain)
	if err != nil {
		return fmt.Errorf("网站 %s 不存在", domain)
	}

	if err := s.gw.RemoveSiteConf(domain); err != nil {
		return err
	}
	if name := w.containerName(); name != "" {
		if err := s.rt.Stop(name); err != nil {
			return fmt.Errorf("停止容器失败: %w", err)
		}
	}
	w.Status = "stopped"
	return s.saveMeta(w)
}

func (s *WebsiteService) buildSpec(w *Website) ContainerSpec {
	rootDir := s.RootDir(w.Domain)
	if w.Type == SiteTypePHP {
		return ContainerSpec{
			Name:    phpContainerName(w.Domain),
			Image:   "php:" + w.PHPVersion + "-fpm",
			Volumes: map[string]string{rootDir: "/var/www/html"},
		}
	}
	return ContainerSpec{
		Name:    siteContainerName(w.Domain),
		Image:   w.Image,
		Volumes: map[string]string{rootDir: "/var/www/html"},
	}
}

// DeleteWebsite 删除网站
func (s *WebsiteService) DeleteWebsite(domain string, keepFiles bool) error {
	w, err := s.loadMeta(domain)
	if err != nil {
		return fmt.Errorf("网站 %s 不存在", domain)
	}

	if name := w.containerName(); name != "" {
		if err := s.rt.Remove(name); err != nil {
			return fmt.Errorf("删除容器失败: %w", err)
		}
	}
	if err := s.gw.RemoveSiteConf(domain); err != nil {
		return err
	}
	_ = os.RemoveAll(filepath.Join(s.cfg.GatewayCertDir(), domain))

	if !keepFiles {
		return os.RemoveAll(s.SiteDir(domain))
	}
	// 保留文件：仅删除元数据与配置，站点目录保留为普通目录
	return os.Remove(s.metaPath(domain))
}

// UpdateWebsite 更新网站配置（别名 / 备注 / PHP 版本）
func (s *WebsiteService) UpdateWebsite(domain string, patch *Website) (*Website, error) {
	w, err := s.loadMeta(domain)
	if err != nil {
		return nil, fmt.Errorf("网站 %s 不存在", domain)
	}

	if patch.Aliases != nil {
		w.Aliases = patch.Aliases
	}
	if patch.Remark != "" {
		w.Remark = patch.Remark
	}
	if patch.PHPVersion != "" && w.Type == SiteTypePHP && patch.PHPVersion != w.PHPVersion {
		image := "php:" + patch.PHPVersion + "-fpm"
		if err := s.rt.PullImage(image, nil); err != nil {
			return nil, err
		}
		if err := s.rt.Remove(phpContainerName(domain)); err != nil {
			return nil, err
		}
		if _, err := s.rt.CreateAndStart(ContainerSpec{
			Name:    phpContainerName(domain),
			Image:   image,
			Volumes: map[string]string{s.RootDir(domain): "/var/www/html"},
		}); err != nil {
			return nil, err
		}
		w.PHPVersion = patch.PHPVersion
		w.Image = image
	}
	if patch.Type == SiteTypeProxy && patch.Port > 0 {
		w.Port = patch.Port
	}

	if err := s.gw.WriteSiteConf(domain, s.renderConf(w)); err != nil {
		return nil, err
	}
	if err := s.saveMeta(w); err != nil {
		return nil, err
	}
	s.fillStatus(w)
	return w, nil
}

// ApplySSL 为站点部署证书
func (s *WebsiteService) ApplySSL(domain, fullchain, privkey string, forceSSL bool) error {
	w, err := s.loadMeta(domain)
	if err != nil {
		return fmt.Errorf("网站 %s 不存在", domain)
	}
	dir := filepath.Join(s.cfg.GatewayCertDir(), domain)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "fullchain.pem"), []byte(fullchain), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "privkey.pem"), []byte(privkey), 0600); err != nil {
		return err
	}

	w.SSLEnabled = true
	w.ForceSSL = forceSSL
	if err := s.gw.WriteSiteConf(domain, s.renderConf(w)); err != nil {
		return err
	}
	return s.saveMeta(w)
}

// DisableSSL 关闭 SSL
func (s *WebsiteService) DisableSSL(domain string) error {
	w, err := s.loadMeta(domain)
	if err != nil {
		return fmt.Errorf("网站 %s 不存在", domain)
	}
	w.SSLEnabled = false
	w.ForceSSL = false
	if err := s.gw.WriteSiteConf(domain, s.renderConf(w)); err != nil {
		return err
	}
	_ = os.RemoveAll(filepath.Join(s.cfg.GatewayCertDir(), domain))
	return s.saveMeta(w)
}

// GatewayStatus 供前端展示网关状态
func (s *WebsiteService) GatewayStatus() (string, string) {
	return s.gw.Status()
}

// EnsureGateway 手动确保网关运行
func (s *WebsiteService) EnsureGateway() error {
	return s.gw.Ensure(nil)
}
