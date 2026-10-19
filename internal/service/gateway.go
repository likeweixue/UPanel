package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"upanel/internal/config"
)

// GatewayName 统一网关容器名。所有站点通过它对外暴露，
// 站点容器本身不映射宿主机端口，从而彻底避免端口冲突与随机端口问题。
const GatewayName = "upanel-gateway"

// GatewayService 管理统一接入网关（Nginx / OpenResty）
type GatewayService struct {
	rt  *Runtime
	cfg *config.Config
}

// NewGatewayService 创建网关服务
func NewGatewayService() (*GatewayService, error) {
	rt, err := NewRuntime()
	if err != nil {
		return nil, err
	}
	return &GatewayService{rt: rt, cfg: config.Current()}, nil
}

// NewGatewayServiceWith 复用已有 Runtime（避免重复建立连接）
func NewGatewayServiceWith(rt *Runtime) *GatewayService {
	return &GatewayService{rt: rt, cfg: config.Current()}
}

func (g *GatewayService) Close() error {
	return g.rt.Close()
}

// mainConf 生成网关主配置。
// 关键点：使用 Docker 内嵌 DNS(127.0.0.11) 作为 resolver，
// 这样即使后端站点容器尚未启动，网关也能正常加载配置。
func (g *GatewayService) mainConf() string {
	return `user  nginx;
worker_processes  auto;

error_log  /var/log/nginx/error.log warn;
pid        /var/run/nginx.pid;

events {
    worker_connections  2048;
}

http {
    include       /etc/nginx/mime.types;
    default_type  application/octet-stream;

    log_format  main  '$remote_addr - $remote_user [$time_local] "$request" '
                      '$status $body_bytes_sent "$http_referer" '
                      '"$http_user_agent" "$http_x_forwarded_for"';

    access_log  /var/log/nginx/access.log  main;

    sendfile        on;
    tcp_nopush      on;
    keepalive_timeout  65;
    server_tokens   off;
    client_max_body_size 256m;

    # Docker 内嵌 DNS，保证后端容器可延迟解析
    resolver 127.0.0.11 valid=30s ipv6=off;

    gzip on;
    gzip_min_length 1k;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml image/svg+xml;

    include /etc/nginx/conf.d/*.conf;
}
`
}

// defaultSiteConf 兜底站点：访问 IP 或未匹配域名时展示 UPanel 提示页，
// 避免 80 端口出现 "no server" 的空白情况。
func (g *GatewayService) defaultSiteConf() string {
	return `server {
    listen 80 default_server;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    location / {
        try_files $uri $uri/ =404;
    }
}
`
}

// welcomePage 写入网关默认欢迎页
func (g *GatewayService) welcomePage() string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>UPanel Gateway</title>
<style>
  body{margin:0;min-height:100vh;display:flex;align-items:center;justify-content:center;
       background:#0f172a;color:#e2e8f0;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;}
  .box{text-align:center;padding:40px}
  h1{font-size:28px;margin:0 0 12px;color:#38bdf8;font-weight:600}
  p{margin:6px 0;color:#94a3b8;font-size:14px}
  code{background:#1e293b;padding:2px 8px;border-radius:4px;color:#f8fafc}
</style>
</head>
<body>
  <div class="box">
    <h1>UPanel Gateway 运行中</h1>
    <p>请通过已创建的网站域名访问，或在面板中创建新网站。</p>
    <p><code>UPanel</code></p>
  </div>
</body>
</html>
`
}

// Ensure 确保网关已就绪：写入主配置 -> 拉取镜像 -> 启动容器。
// 返回错误时附带可操作的排查建议。
func (g *GatewayService) Ensure(logFn func(string)) error {
	confDir := g.cfg.GatewayConfDir()
	if err := os.MkdirAll(confDir, 0755); err != nil {
		return fmt.Errorf("创建网关配置目录失败: %w", err)
	}

	// 主配置
	if err := os.WriteFile(g.cfg.GatewayNginxConf(), []byte(g.mainConf()), 0644); err != nil {
		return fmt.Errorf("写入网关主配置失败: %w", err)
	}

	// 兜底站点（仅在没有任何站点配置时写入，避免覆盖用户自定义）
	entries, _ := filepath.Glob(filepath.Join(confDir, "*.conf"))
	if len(entries) == 0 {
		if err := os.WriteFile(filepath.Join(confDir, "default.conf"), []byte(g.defaultSiteConf()), 0644); err != nil {
			return fmt.Errorf("写入默认站点配置失败: %w", err)
		}
	}

	// 欢迎页
	htmlDir := filepath.Join(g.cfg.DataPath, "gateway", "html")
	if err := os.MkdirAll(htmlDir, 0755); err != nil {
		return fmt.Errorf("创建默认站点目录失败: %w", err)
	}
	_ = os.WriteFile(filepath.Join(htmlDir, "index.html"), []byte(g.welcomePage()), 0644)

	// 证书目录
	_ = os.MkdirAll(g.cfg.GatewayCertDir(), 0755)

	if err := g.rt.PullImage(g.cfg.GatewayImage, logFn); err != nil {
		return err
	}

	ports := map[string]string{
		"80/tcp": g.cfg.GatewayHTTPPort,
	}
	if g.cfg.GatewayHTTPSPort != "" {
		ports["443/tcp"] = g.cfg.GatewayHTTPSPort
	}

	volumes := map[string]string{
		g.cfg.GatewayNginxConf(): "/etc/nginx/nginx.conf",
		confDir:                  "/etc/nginx/conf.d",
		g.cfg.GatewayCertDir():   "/etc/nginx/certs",
		g.cfg.WebsitePath:        "/var/www/websites",
		htmlDir:                  "/usr/share/nginx/html",
	}

	_, err := g.rt.CreateAndStart(ContainerSpec{
		Name:    GatewayName,
		Image:   g.cfg.GatewayImage,
		Volumes: volumes,
		Ports:   ports,
	})
	if err != nil {
		if strings.Contains(err.Error(), "address already in use") ||
			strings.Contains(err.Error(), "port is already allocated") {
			return fmt.Errorf("网关端口 %s 或 %s 已被占用，请先停止占用该端口的服务（如系统自带 nginx/apache），"+
				"或在面板「设置」中更换网关端口: %w", g.cfg.GatewayHTTPPort, g.cfg.GatewayHTTPSPort, err)
		}
		return fmt.Errorf("启动网关容器失败: %w", err)
	}
	return nil
}

// Reload 重载网关配置。reload 失败时自动重启容器兜底。
func (g *GatewayService) Reload() error {
	if g.rt.ContainerState(GatewayName) != "running" {
		return g.Ensure(nil)
	}
	if _, err := g.rt.Exec(GatewayName, "nginx", "-s", "reload"); err != nil {
		// reload 失败多半是容器异常，直接重启兜底
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		timeout := 10 * time.Second
		if rerr := g.rt.Client().ContainerRestart(ctx, GatewayName, &timeout); rerr != nil {
			return fmt.Errorf("重载网关配置失败: %w", rerr)
		}
	}
	return nil
}

// Status 网关状态
func (g *GatewayService) Status() (state string, port string) {
	state = g.rt.ContainerState(GatewayName)
	if state == "" {
		return "not_installed", g.cfg.GatewayHTTPPort
	}
	return state, g.cfg.GatewayHTTPPort
}

// WriteSiteConf 写入站点配置并重载网关
func (g *GatewayService) WriteSiteConf(domain, content string) error {
	confDir := g.cfg.GatewayConfDir()
	path := filepath.Join(confDir, SiteConfName(domain))
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入站点配置失败: %w", err)
	}
	return g.Reload()
}

// RemoveSiteConf 删除站点配置并重载网关
func (g *GatewayService) RemoveSiteConf(domain string) error {
	path := filepath.Join(g.cfg.GatewayConfDir(), SiteConfName(domain))
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return g.Reload()
}

// SiteConfName 站点配置文件名
func SiteConfName(domain string) string {
	safe := strings.ReplaceAll(domain, "*", "_wildcard_")
	safe = strings.ReplaceAll(safe, "/", "_")
	return safe + ".conf"
}

// Slug 把域名转换为可用于容器名/目录名的安全字符串
func Slug(s string) string {
	r := strings.NewReplacer(
		".", "-",
		":", "-",
		"/", "-",
		"_", "-",
	)
	return strings.ToLower(r.Replace(s))
}
