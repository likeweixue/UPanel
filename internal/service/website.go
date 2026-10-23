package service

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
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
	mu  sync.Mutex
}

// NewWebsiteService 创建网站服务。
// Docker 不可用时仍返回只读服务（可列出已有站点），写操作会返回明确错误。
func NewWebsiteService() (*WebsiteService, error) {
	cfg := config.Current()
	if err := os.MkdirAll(cfg.WebsitePath, 0755); err != nil {
		return nil, fmt.Errorf("创建网站目录失败: %w", err)
	}
	s := &WebsiteService{cfg: cfg}
	rt, err := NewRuntime()
	if err != nil {
		return s, err
	}
	s.rt = rt
	s.gw = NewGatewayServiceWith(rt)
	return s, nil
}

func (s *WebsiteService) dockerReady() error {
	if s.rt == nil || s.gw == nil {
		return fmt.Errorf("Docker 未连接，无法执行该操作")
	}
	return nil
}

func (s *WebsiteService) Close() error {
	if s.rt == nil {
		return nil
	}
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

// domainRe 域名标签校验：允许普通域名与通配域名（*.example.com）
var domainRe = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?(\.[a-zA-Z0-9]([a-zA-Z0-9\-]{0,61}[a-zA-Z0-9])?)*$`)

var phpVersionRe = regexp.MustCompile(`^[0-9]{1,2}\.[0-9]{1,2}$`)
var imageRefRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._/:@\-]{0,200}$`)

// ValidateDomain 校验域名合法性，支持 *.example.com 形式的通配域名
func ValidateDomain(domain string) error {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		return fmt.Errorf("域名不能为空")
	}
	if len(domain) > 253 {
		return fmt.Errorf("域名过长")
	}
	if strings.ContainsAny(domain, "/\\ \t\r\n;'\"`$") || strings.Contains(domain, "..") {
		return fmt.Errorf("域名包含非法字符")
	}
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return fmt.Errorf("域名格式不合法: %s", domain)
	}

	// 通配域名：仅允许前缀 *.
	target := domain
	if strings.HasPrefix(domain, "*.") {
		target = strings.TrimPrefix(domain, "*.")
		if !strings.Contains(target, ".") {
			return fmt.Errorf("通配域名必须包含至少一个点号: %s", domain)
		}
	} else if strings.Contains(domain, "*") {
		return fmt.Errorf("域名格式不合法: %s", domain)
	}
	if !domainRe.MatchString(target) {
		return fmt.Errorf("域名格式不合法: %s", domain)
	}
	for _, label := range strings.Split(target, ".") {
		if label == "" || len(label) > 63 {
			return fmt.Errorf("域名格式不合法: %s", domain)
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return fmt.Errorf("域名格式不合法: %s", domain)
		}
	}
	return nil
}

func normalizeDomain(domain string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(domain), "."))
}

// ValidatePHPVersion 校验 PHP 版本号
func ValidatePHPVersion(v string) error {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	if !phpVersionRe.MatchString(v) {
		return fmt.Errorf("PHP 版本格式不合法（示例: 8.2）")
	}
	return nil
}

func validateImageRef(image string) error {
	if strings.TrimSpace(image) == "" {
		return fmt.Errorf("镜像名称不能为空")
	}
	if !imageRefRe.MatchString(image) {
		return fmt.Errorf("镜像名称包含非法字符")
	}
	return nil
}

func validateProxyPort(port int) error {
	if port < 1 || port > 65535 {
		return fmt.Errorf("端口必须是 1-65535 之间的数字")
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

// ErrSiteNotFound 网站不存在
var ErrSiteNotFound = errors.New("网站不存在")

// checkDomainConflict 校验主域名/别名是否已被其它站点占用
func (s *WebsiteService) checkDomainConflict(names []string, exclude string) error {
	entries, err := os.ReadDir(s.cfg.WebsitePath)
	if err != nil {
		return nil
	}
	used := map[string]string{}
	for _, e := range entries {
		if !e.IsDir() || e.Name() == exclude {
			continue
		}
		w, err := s.loadMeta(e.Name())
		if err != nil {
			continue
		}
		used[strings.ToLower(w.Domain)] = w.Domain
		for _, a := range w.Aliases {
			used[strings.ToLower(a)] = w.Domain
		}
	}
	for _, n := range names {
		if owner, ok := used[strings.ToLower(n)]; ok && owner != exclude {
			return fmt.Errorf("域名 %s 已被站点 %s 使用", n, owner)
		}
	}
	return nil
}

// validateAliases 归一化并校验别名列表
func (s *WebsiteService) validateAliases(domain string, aliases []string, exclude string) ([]string, error) {
	if len(aliases) > 20 {
		return nil, fmt.Errorf("别名域名最多 20 个")
	}
	out := make([]string, 0, len(aliases))
	seen := map[string]bool{strings.ToLower(domain): true}
	for _, raw := range aliases {
		a := normalizeDomain(raw)
		if a == "" {
			continue
		}
		if err := ValidateDomain(a); err != nil {
			return nil, fmt.Errorf("别名 %s 无效: %w", raw, err)
		}
		if seen[a] {
			return nil, fmt.Errorf("别名 %s 重复", a)
		}
		seen[a] = true
		out = append(out, a)
	}
	if err := s.checkDomainConflict(append([]string{domain}, out...), exclude); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateWebsite 创建网站
func (s *WebsiteService) CreateWebsite(req *CreateWebsiteRequest) (*Website, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.dockerReady(); err != nil {
		return nil, err
	}

	domain := normalizeDomain(req.Domain)
	if err := ValidateDomain(domain); err != nil {
		return nil, err
	}
	aliases, err := s.validateAliases(domain, req.Aliases, "")
	if err != nil {
		return nil, err
	}

	if len([]rune(strings.TrimSpace(req.Remark))) > 200 {
		return nil, fmt.Errorf("备注最多 200 个字符")
	}

	siteType := strings.ToLower(strings.TrimSpace(req.Type))
	if siteType == "" {
		if strings.TrimSpace(req.PHPVersion) != "" {
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

	phpVersion := strings.TrimSpace(req.PHPVersion)
	proxyImage := strings.TrimSpace(req.Image)
	proxyPort := req.Port
	if siteType == SiteTypePHP {
		if err := ValidatePHPVersion(phpVersion); err != nil {
			return nil, err
		}
		if phpVersion == "" {
			phpVersion = phpVersionFromImage(s.cfg.PHPImage)
		}
	}
	if siteType == SiteTypeProxy {
		if err := validateImageRef(proxyImage); err != nil {
			return nil, err
		}
		if proxyPort == 0 {
			proxyPort = 80
		}
		if err := validateProxyPort(proxyPort); err != nil {
			return nil, err
		}
	}

	siteDir := s.SiteDir(domain)
	rootDir := s.RootDir(domain)
	siteDirExisted := dirExists(siteDir)
	if _, err := os.Stat(filepath.Join(siteDir, "site.json")); err == nil {
		return nil, fmt.Errorf("网站 %s 已存在", domain)
	}
	// 目录存在但没有 site.json：可能是删除时保留了文件，或上次半途失败。
	// 允许复用目录，但不会覆盖已有首页文件。

	// 1. 先确保网关就绪，避免创建一半目录后才发现 Docker 不可用
	if err := s.gw.Ensure(nil); err != nil {
		return nil, err
	}

	// 2. 目录结构
	logDir := filepath.Join(siteDir, "logs")
	for _, d := range []string{rootDir, logDir, filepath.Join(siteDir, "conf")} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("创建站点目录失败: %w", err)
		}
	}
	if siteType == SiteTypePHP {
		preparePHPDir(rootDir)
	}
	if siteType == SiteTypeProxy {
		// 常见 Node/Go 镜像以 UID 1000 运行；让应用可以写自己的工作目录
		prepareWritableDir(rootDir, 1000, 1000)
	}

	site := &Website{
		Domain:    domain,
		Aliases:   aliases,
		Type:      siteType,
		Path:      siteDir,
		RootDir:   rootDir,
		Remark:    strings.TrimSpace(req.Remark),
		CreatedAt: time.Now(),
	}

	cleanup := func() {
		if name := site.containerName(); name != "" {
			_ = s.rt.Remove(name)
		}
		_ = s.gw.RemoveSiteConf(domain)
		if !siteDirExisted {
			_ = os.RemoveAll(siteDir)
		}
	}

	// 3. 默认首页
	if err := s.writeDefaultIndex(siteType, rootDir, domain); err != nil {
		cleanup()
		return nil, err
	}

	// 4. 按类型准备运行时
	switch siteType {
	case SiteTypePHP:
		image := "php:" + phpVersion + "-fpm"
		if err := s.rt.PullImage(image, nil); err != nil {
			cleanup()
			return nil, err
		}
		if _, err := s.rt.CreateAndStart(ContainerSpec{
			Name:    phpContainerName(domain),
			Image:   image,
			Volumes: map[string]string{rootDir: "/var/www/html"},
			Labels:  map[string]string{"managed-by": "upanel", "upanel-site": domain},
		}); err != nil {
			cleanup()
			return nil, fmt.Errorf("创建 PHP 容器失败: %w", err)
		}
		site.PHPVersion = phpVersion
		site.Image = image

	case SiteTypeProxy:
		if err := s.rt.PullImage(proxyImage, nil); err != nil {
			cleanup()
			return nil, err
		}
		if _, err := s.rt.CreateAndStart(ContainerSpec{
			Name:    siteContainerName(domain),
			Image:   proxyImage,
			Volumes: map[string]string{rootDir: "/var/www/html"},
			Labels:  map[string]string{"managed-by": "upanel", "upanel-site": domain},
		}); err != nil {
			cleanup()
			return nil, fmt.Errorf("创建站点容器失败: %w", err)
		}
		site.Image = proxyImage
		site.Port = proxyPort
	}

	site.Status = "running"

	// 5. 写入网关配置（内部会执行 nginx -t 并回滚）
	if err := s.gw.WriteSiteConf(domain, s.renderConf(site)); err != nil {
		cleanup()
		return nil, fmt.Errorf("站点配置未通过校验，已回滚: %w", err)
	}

	// 6. 落盘元数据（必须最后一步）
	if err := s.saveMeta(site); err != nil {
		cleanup()
		return nil, err
	}

	s.fillStatus(site)
	return site, nil
}

func phpVersionFromImage(image string) string {
	img := strings.TrimPrefix(image, "php:")
	img = strings.TrimSuffix(img, "-fpm")
	parts := strings.Split(img, "-")
	if len(parts) > 0 && phpVersionRe.MatchString(parts[0]) {
		return parts[0]
	}
	return "8.2"
}

func (s *WebsiteService) writeDefaultIndex(siteType, rootDir, domain string) error {
	domain = html.EscapeString(domain)
	var name, content string
	switch siteType {
	case SiteTypePHP:
		name = "index.php"
		content = fmt.Sprintf(`<?php
echo "<h1>Welcome to %s</h1>";
echo "<p>Powered by UPanel</p>";
echo "<p>PHP Version: " . htmlspecialchars(phpversion()) . "</p>";
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
       background:#f4f5f7;color:#1f2937;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;}
  .box{text-align:center;padding:40px;background:#fff;border:1px solid #e5e7eb;border-radius:12px;max-width:520px}
  h1{font-size:24px;margin:0 0 10px;color:#2f6fed;font-weight:600}
  p{margin:6px 0;color:#6b7280;font-size:14px;line-height:1.6}
  code{background:#f3f4f6;padding:2px 8px;border-radius:4px;color:#111827}
</style>
</head>
<body>
  <div class="box">
    <h1>Welcome to %s</h1>
    <p>站点已创建成功，把你的网页文件上传到网站根目录即可。</p>
    <p><code>%s</code></p>
    <p>Powered by UPanel</p>
  </div>
</body>
</html>
`, domain, domain, domain)
	}
	target := filepath.Join(rootDir, name)
	if _, err := os.Stat(target); err == nil {
		// 保留用户已有首页，避免“删除站点但保留文件”后重建时覆盖内容
		return nil
	}
	return os.WriteFile(target, []byte(content), 0644)
}

// serverNames 生成 server_name 指令内容
func serverNames(site *Website) string {
	names := []string{site.Domain}
	seen := map[string]bool{strings.ToLower(site.Domain): true}
	for _, a := range site.Aliases {
		a = strings.ToLower(strings.TrimSpace(a))
		if a == "" || seen[a] || ValidateDomain(a) != nil {
			continue
		}
		seen[a] = true
		names = append(names, a)
	}
	return strings.Join(names, " ")
}

// renderConf 渲染 Nginx 站点配置
func (s *WebsiteService) renderConf(site *Website) string {
	root := fmt.Sprintf("/var/www/websites/%s/html", site.Domain)
	names := serverNames(site)
	logPrefix := fmt.Sprintf("/var/log/nginx/upanel/%s", site.Domain)

	var b strings.Builder

	// ---- 强制 HTTPS：只保留 80 跳转，避免与内容 server 块重复 ----
	if site.SSLEnabled && site.ForceSSL {
		fmt.Fprintf(&b, `# UPanel managed site: %s
server {
    listen 80;
    server_name %s;
    return 301 https://$host$request_uri;
}
`, site.Domain, names)
	} else {
		fmt.Fprintf(&b, `# UPanel managed site: %s
server {
    listen 80;
    server_name %s;

    root %s;
    index %s;

    access_log %s.access.log;
    error_log  %s.error.log;
`, site.Domain, names, root, s.indexLine(site), logPrefix, logPrefix)
		s.writeLocationBlock(&b, site, false)
		b.WriteString("}\n")
	}

	// ---- HTTPS ----
	if site.SSLEnabled {
		certDir := fmt.Sprintf("/etc/nginx/certs/%s", site.Domain)
		fmt.Fprintf(&b, `
server {
    listen 443 ssl;
    server_name %s;

    ssl_certificate     %s/fullchain.pem;
    ssl_certificate_key %s/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:10m;
    ssl_session_timeout 1d;
    ssl_session_tickets off;
    add_header Strict-Transport-Security "max-age=31536000" always;
    add_header X-Content-Type-Options nosniff always;
    add_header X-Frame-Options SAMEORIGIN always;
    add_header Referrer-Policy strict-origin-when-cross-origin always;

    root %s;
    index %s;

    access_log %s.ssl.access.log;
    error_log  %s.ssl.error.log;
`, names, certDir, certDir, root, s.indexLine(site), logPrefix, logPrefix)
		s.writeLocationBlock(&b, site, true)
		b.WriteString("}\n")
	}

	return b.String()
}

// writeLocationBlock 写入类型相关的 location 配置
func (s *WebsiteService) writeLocationBlock(b *strings.Builder, site *Website, ssl bool) {
	// 禁止访问隐藏文件（保留 ACME 验证目录）与常见敏感/备份文件
	b.WriteString(`
    location ~ /\.(?!well-known) {
        deny all;
        access_log off;
        log_not_found off;
    }

    location ~* \.(sql|bak|old|orig|swp|swo)$ {
        deny all;
        access_log off;
    }
`)

	switch site.Type {
	case SiteTypePHP:
		fmt.Fprintf(b, `
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
        # 注意：php-fpm 运行在独立容器中，站点目录挂载在 /var/www/html。
        # 不能使用 $document_root（那是网关容器内的路径），否则 php-fpm 会报 Primary script unknown。
        fastcgi_param SCRIPT_FILENAME /var/www/html$fastcgi_script_name;
        fastcgi_param DOCUMENT_ROOT /var/www/html;
        fastcgi_param PATH_INFO $fastcgi_path_info;
        fastcgi_param HTTP_PROXY "";
        %s
        fastcgi_read_timeout 300;
    }
`, phpContainerName(site.Domain), httpsFastCGIParam(ssl))

	case SiteTypeProxy:
		fmt.Fprintf(b, `
    location / {
        set $upanel_backend "http://%s:%d";
        proxy_pass $upanel_backend;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto %s;
        proxy_set_header X-Forwarded-Host $host;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;
        proxy_read_timeout 300;
    }
`, siteContainerName(site.Domain), site.Port, scheme(ssl))

	default: // static
		b.WriteString(`
    # 静态站点不应执行或暴露 PHP 源码
    location ~ \.php$ {
        return 404;
    }

    location / {
        try_files $uri $uri/ =404;
    }

    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|webp|woff2?|ttf|eot)$ {
        expires 7d;
        access_log off;
    }
`)
	}
}

func httpsFastCGIParam(ssl bool) string {
	if ssl {
		return "fastcgi_param HTTPS on;"
	}
	return ""
}

func scheme(ssl bool) string {
	if ssl {
		return "https"
	}
	return "http"
}

func (s *WebsiteService) indexLine(site *Website) string {
	if site.Type == SiteTypePHP {
		return "index.php index.html index.htm"
	}
	// 静态站点绝不能把 index.php 作为默认索引，否则会以纯文本形式泄露 PHP 源码
	return "index.html index.htm"
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
	if err := os.MkdirAll(s.SiteDir(site.Domain), 0755); err != nil {
		return err
	}
	return writeFileAtomic(s.metaPath(site.Domain), data, 0600)
}

func (s *WebsiteService) loadMeta(domain string) (*Website, error) {
	if err := ValidateDomain(domain); err != nil {
		return nil, ErrSiteNotFound
	}
	data, err := os.ReadFile(s.metaPath(domain))
	if err != nil {
		return nil, ErrSiteNotFound
	}
	var w Website
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, ErrSiteNotFound
	}
	// 目录名是唯一权威来源，防止手工篡改 site.json 注入非法域名
	w.Domain = domain
	if w.Aliases == nil {
		w.Aliases = []string{}
	}
	cleanAliases := make([]string, 0, len(w.Aliases))
	seen := map[string]bool{domain: true}
	for _, a := range w.Aliases {
		a = normalizeDomain(a)
		if a == "" || seen[a] || ValidateDomain(a) != nil {
			continue
		}
		seen[a] = true
		cleanAliases = append(cleanAliases, a)
	}
	w.Aliases = cleanAliases
	switch w.Type {
	case SiteTypeStatic, SiteTypePHP, SiteTypeProxy:
	default:
		w.Type = SiteTypeStatic
	}
	if w.Type == SiteTypePHP && ValidatePHPVersion(w.PHPVersion) != nil {
		w.PHPVersion = "8.2"
	}
	if w.Type == SiteTypePHP && w.Image != "" && !imageRefRe.MatchString(w.Image) {
		w.Image = ""
	}
	if w.Type == SiteTypeProxy {
		if w.Port < 1 || w.Port > 65535 {
			w.Port = 80
		}
		if w.Image != "" && !imageRefRe.MatchString(w.Image) {
			w.Image = ""
		}
	}
	return &w, nil
}

// GetWebsite 获取单个网站（含实时状态）
func (s *WebsiteService) GetWebsite(domain string) (*Website, error) {
	domain = normalizeDomain(domain)
	w, err := s.loadMeta(domain)
	if err != nil {
		return nil, fmt.Errorf("网站 %s 不存在", domain)
	}
	s.fillStatus(w)
	return w, nil
}

// fillStatus 计算实时运行状态
func (s *WebsiteService) fillStatus(w *Website) {
	if s.rt == nil {
		w.Status = "unknown"
		return
	}
	gwState := s.rt.ContainerState(GatewayName)
	confExists := fileExists(filepath.Join(s.cfg.GatewayConfDir(), SiteConfName(w.Domain)))
	containerOK := true
	if name := w.containerName(); name != "" {
		containerOK = s.rt.ContainerState(name) == "running"
	}
	if gwState == "running" && confExists && containerOK {
		w.Status = "running"
		return
	}
	w.Status = "stopped"
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
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
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

	list := make([]Website, 0, len(entries))
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.dockerReady(); err != nil {
		return err
	}

	domain = normalizeDomain(domain)
	w, err := s.loadMeta(domain)
	if err != nil {
		return fmt.Errorf("网站 %s 不存在", domain)
	}

	if err := s.gw.Ensure(nil); err != nil {
		return err
	}

	if name := w.containerName(); name != "" {
		if w.Type == SiteTypePHP {
			preparePHPDir(s.RootDir(w.Domain))
		}
		state := s.rt.ContainerState(name)
		if state == "" {
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
	s.fillStatus(w)
	return s.saveMeta(w)
}

// StopWebsite 停止网站（移除网关配置，避免访问时出现 502）
func (s *WebsiteService) StopWebsite(domain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.dockerReady(); err != nil {
		return err
	}

	domain = normalizeDomain(domain)
	w, err := s.loadMeta(domain)
	if err != nil {
		return fmt.Errorf("网站 %s 不存在", domain)
	}

	if err := s.gw.RemoveSiteConf(domain); err != nil {
		return err
	}
	if name := w.containerName(); name != "" && s.rt.ContainerState(name) != "" {
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
		image := w.Image
		if image == "" {
			image = "php:" + w.PHPVersion + "-fpm"
		}
		return ContainerSpec{
			Name:    phpContainerName(w.Domain),
			Image:   image,
			Volumes: map[string]string{rootDir: "/var/www/html"},
			Labels:  map[string]string{"managed-by": "upanel", "upanel-site": w.Domain},
		}
	}
	return ContainerSpec{
		Name:    siteContainerName(w.Domain),
		Image:   w.Image,
		Volumes: map[string]string{rootDir: "/var/www/html"},
		Labels:  map[string]string{"managed-by": "upanel", "upanel-site": w.Domain},
	}
}

// DeleteWebsite 删除网站
func (s *WebsiteService) DeleteWebsite(domain string, keepFiles bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.dockerReady(); err != nil {
		return err
	}

	domain = normalizeDomain(domain)
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
	return os.Remove(s.metaPath(domain))
}

// UpdateWebsite 更新网站配置（别名 / 备注 / PHP 版本 / 反代端口）
func (s *WebsiteService) UpdateWebsite(domain string, patch *Website) (*Website, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.dockerReady(); err != nil {
		return nil, err
	}

	domain = normalizeDomain(domain)
	w, err := s.loadMeta(domain)
	if err != nil {
		return nil, fmt.Errorf("网站 %s 不存在", domain)
	}
	if patch.Type != "" && patch.Type != w.Type {
		return nil, fmt.Errorf("暂不支持修改网站类型，请删除后重新创建")
	}

	if patch.Aliases != nil {
		aliases, err := s.validateAliases(w.Domain, patch.Aliases, w.Domain)
		if err != nil {
			return nil, err
		}
		w.Aliases = aliases
	}
	if len([]rune(strings.TrimSpace(patch.Remark))) > 200 {
		return nil, fmt.Errorf("备注最多 200 个字符")
	}
	w.Remark = strings.TrimSpace(patch.Remark)
	if patch.PHPVersion != "" && w.Type == SiteTypePHP && patch.PHPVersion != w.PHPVersion {
		if err := ValidatePHPVersion(patch.PHPVersion); err != nil {
			return nil, err
		}
		image := "php:" + patch.PHPVersion + "-fpm"
		if err := s.rt.PullImage(image, nil); err != nil {
			return nil, err
		}
		// 同名容器正在运行时 CreateAndStart 会直接返回，必须显式删除才能切换镜像
		oldImage := w.Image
		if oldImage == "" {
			oldImage = "php:" + w.PHPVersion + "-fpm"
		}
		if err := s.rt.Remove(phpContainerName(domain)); err != nil {
			return nil, err
		}
		if _, err := s.rt.CreateAndStart(ContainerSpec{
			Name:    phpContainerName(domain),
			Image:   image,
			Volumes: map[string]string{s.RootDir(domain): "/var/www/html"},
			Labels:  map[string]string{"managed-by": "upanel", "upanel-site": domain},
		}); err != nil {
			// 回滚到旧镜像，避免一次失败的版本切换让站点直接不可用
			_, _ = s.rt.CreateAndStart(ContainerSpec{
				Name:    phpContainerName(domain),
				Image:   oldImage,
				Volumes: map[string]string{s.RootDir(domain): "/var/www/html"},
				Labels:  map[string]string{"managed-by": "upanel", "upanel-site": domain},
			})
			return nil, err
		}
		w.PHPVersion = patch.PHPVersion
		w.Image = image
	}
	if w.Type == SiteTypeProxy && patch.Port > 0 && patch.Port != w.Port {
		if err := validateProxyPort(patch.Port); err != nil {
			return nil, err
		}
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
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.dockerReady(); err != nil {
		return err
	}

	domain = normalizeDomain(domain)
	w, err := s.loadMeta(domain)
	if err != nil {
		return fmt.Errorf("网站 %s 不存在", domain)
	}
	if len(fullchain) > 2<<20 || len(privkey) > 2<<20 {
		return fmt.Errorf("证书或私钥文件过大")
	}
	cert, err := tls.X509KeyPair([]byte(fullchain), []byte(privkey))
	if err != nil {
		return fmt.Errorf("证书与私钥无效或不匹配: %w", err)
	}
	if len(cert.Certificate) == 0 {
		return fmt.Errorf("证书内容无效")
	}
	leaf, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("证书解析失败: %w", err)
	}
	if time.Now().After(leaf.NotAfter) {
		return fmt.Errorf("证书已过期（到期时间 %s）", leaf.NotAfter.Format("2006-01-02"))
	}

	dir := filepath.Join(s.cfg.GatewayCertDir(), domain)
	newDir := dir + ".new"
	oldDir := dir + ".old"
	_ = os.RemoveAll(newDir)
	_ = os.RemoveAll(oldDir)
	if err := os.MkdirAll(newDir, 0700); err != nil {
		return err
	}
	if err := writeFileAtomic(filepath.Join(newDir, "fullchain.pem"), []byte(fullchain), 0644); err != nil {
		_ = os.RemoveAll(newDir)
		return err
	}
	if err := writeFileAtomic(filepath.Join(newDir, "privkey.pem"), []byte(privkey), 0600); err != nil {
		_ = os.RemoveAll(newDir)
		return err
	}
	if _, err := os.Stat(dir); err == nil {
		if err := os.Rename(dir, oldDir); err != nil {
			_ = os.RemoveAll(newDir)
			return err
		}
	}
	if err := os.Rename(newDir, dir); err != nil {
		if _, statErr := os.Stat(oldDir); statErr == nil {
			_ = os.Rename(oldDir, dir)
		}
		_ = os.RemoveAll(newDir)
		return err
	}

	prevSSL, prevForce := w.SSLEnabled, w.ForceSSL
	w.SSLEnabled = true
	w.ForceSSL = forceSSL
	if err := s.gw.WriteSiteConf(domain, s.renderConf(w)); err != nil {
		w.SSLEnabled, w.ForceSSL = prevSSL, prevForce
		_ = os.RemoveAll(dir)
		if _, statErr := os.Stat(oldDir); statErr == nil {
			_ = os.Rename(oldDir, dir)
		}
		return fmt.Errorf("证书已写入但站点配置未通过校验，已回滚: %w", err)
	}
	_ = os.RemoveAll(oldDir)
	return s.saveMeta(w)
}

// DisableSSL 关闭 SSL
func (s *WebsiteService) DisableSSL(domain string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.dockerReady(); err != nil {
		return err
	}

	domain = normalizeDomain(domain)
	w, err := s.loadMeta(domain)
	if err != nil {
		return fmt.Errorf("网站 %s 不存在", domain)
	}
	w.SSLEnabled = false
	w.ForceSSL = false
	if err := s.gw.WriteSiteConf(domain, s.renderConf(w)); err != nil {
		return err
	}
	if err := s.saveMeta(w); err != nil {
		return err
	}
	_ = os.RemoveAll(filepath.Join(s.cfg.GatewayCertDir(), domain))
	return nil
}

// SiteLogKind 站点日志类型
type SiteLogKind = string

const (
	SiteLogAccess SiteLogKind = "access"
	SiteLogError  SiteLogKind = "error"
)

// GetSiteLogs 读取站点访问/错误日志的最后若干行
func (s *WebsiteService) GetSiteLogs(domain string, kind string, lines int) (string, error) {
	domain = normalizeDomain(domain)
	if err := ValidateDomain(domain); err != nil {
		return "", err
	}
	if _, err := s.loadMeta(domain); err != nil {
		return "", fmt.Errorf("网站 %s 不存在", domain)
	}
	if kind == "" {
		kind = SiteLogAccess
	}
	var path string
	switch strings.ToLower(kind) {
	case SiteLogAccess:
		path = filepath.Join(s.cfg.GatewayLogDir(), domain+".access.log")
	case SiteLogError:
		path = filepath.Join(s.cfg.GatewayLogDir(), domain+".error.log")
	default:
		return "", fmt.Errorf("未知的日志类型")
	}
	if lines <= 0 || lines > 2000 {
		lines = 200
	}
	return tailFile(path, lines, 2<<20)
}

func tailFile(path string, maxLines int, maxBytes int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	size := info.Size()
	start := int64(0)
	if size > maxBytes {
		start = size - maxBytes
	}
	buf := make([]byte, size-start)
	if _, err := f.ReadAt(buf, start); err != nil && err != io.EOF {
		return "", err
	}
	text := string(buf)
	all := strings.Split(text, "\n")
	if start > 0 {
		all = all[1:] // 丢弃被截断的首行
	}
	if len(all) > maxLines {
		all = all[len(all)-maxLines:]
	}
	return strings.Join(all, "\n"), nil
}

// GatewayStatus 供前端展示网关状态
func (s *WebsiteService) GatewayStatus() (string, string) {
	if s.gw == nil {
		return "docker_unavailable", config.Current().GatewayHTTPPort
	}
	return s.gw.Status()
}

// EnsureGateway 手动确保网关运行
func (s *WebsiteService) EnsureGateway() error {
	if err := s.dockerReady(); err != nil {
		return err
	}
	return s.gw.Ensure(nil)
}
