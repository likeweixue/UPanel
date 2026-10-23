package service

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	mu  sync.Mutex
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
                      '"$http_user_agent" "$http_x_forwarded_for" '
                      'rt=$request_time';

    access_log  /var/log/nginx/access.log  main;

    sendfile        on;
    tcp_nopush      on;
    keepalive_timeout  65;
    server_tokens   off;
    client_max_body_size 256m;
    client_body_timeout 60s;
    send_timeout 60s;
    server_names_hash_bucket_size 128;

    # Docker 内嵌 DNS，保证后端容器可延迟解析
    resolver 127.0.0.11 valid=30s ipv6=off;

    # WebSocket 升级
    map $http_upgrade $connection_upgrade {
        default upgrade;
        ''      close;
    }

    gzip on;
    gzip_min_length 1k;
    gzip_comp_level 5;
    gzip_types text/plain text/css application/json application/javascript
               application/xml text/xml image/svg+xml application/wasm;

    # 所有站点统一继承的基础安全头
    add_header X-Content-Type-Options nosniff always;
    add_header X-Frame-Options SAMEORIGIN always;
    add_header Referrer-Policy strict-origin-when-cross-origin always;

    include /etc/nginx/conf.d/*.conf;
}
`
}

// defaultSiteConf 兜底站点：访问 IP 或未匹配域名时展示 UPanel 提示页。
// 这个配置必须始终存在，否则删除最后一个站点后 80 端口将不再监听。
func (g *GatewayService) defaultSiteConf() string {
	return `# Managed by UPanel. Do not edit.
server {
    listen 80 default_server;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    location = /_upanel/health {
        access_log off;
        default_type text/plain;
        return 200 "ok\n";
    }

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
       background:#f4f5f7;color:#1f2937;font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif;}
  .box{text-align:center;padding:40px;background:#fff;border:1px solid #e5e7eb;border-radius:12px;max-width:460px}
  h1{font-size:24px;margin:0 0 12px;color:#2f6fed;font-weight:600}
  p{margin:8px 0;color:#6b7280;font-size:14px;line-height:1.6}
  code{background:#f3f4f6;padding:2px 8px;border-radius:4px;color:#111827}
</style>
</head>
<body>
  <div class="box">
    <h1>UPanel Gateway 运行中</h1>
    <p>当前请求没有匹配到已创建的网站。</p>
    <p>请在面板中创建网站，并把域名解析到这台服务器。</p>
    <p><code>UPanel</code></p>
  </div>
</body>
</html>
`
}

// writeFileInPlace 原地写入，保留 inode。
// 网关的 nginx.conf 是以单文件 bind mount 挂进容器的；如果用 rename 原子替换，
// 容器仍会指向旧 inode，导致配置更新不生效。
func writeFileInPlace(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Chmod(path, perm)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	if err := os.Chmod(tmp, perm); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Ensure 确保网关已就绪：写入配置 -> 拉取镜像 -> 启动容器 -> 校验配置。
func (g *GatewayService) Ensure(logFn func(string)) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.ensureLocked(logFn)
}

func (g *GatewayService) ensureLocked(logFn func(string)) error {
	oldMain, hadMain := readFileIfExists(g.cfg.GatewayNginxConf())
	if err := g.writeBaseConfig(); err != nil {
		return err
	}

	wasRunning := g.rt.ContainerState(GatewayName) == "running"

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
		g.cfg.GatewayConfDir():   "/etc/nginx/conf.d",
		g.cfg.GatewayCertDir():   "/etc/nginx/certs",
		g.cfg.WebsitePath:        "/var/www/websites",
		g.cfg.GatewayHTMLDir():   "/usr/share/nginx/html",
		g.cfg.GatewayLogDir():    "/var/log/nginx/upanel",
	}

	_, err := g.rt.CreateAndStart(ContainerSpec{
		Name:            GatewayName,
		Image:           g.cfg.GatewayImage,
		Volumes:         volumes,
		ReadOnlyVolumes: []string{g.cfg.GatewayNginxConf(), g.cfg.GatewayConfDir(), g.cfg.GatewayCertDir(), g.cfg.WebsitePath, g.cfg.GatewayHTMLDir()},
		Ports:           ports,
	})
	if err != nil {
		if strings.Contains(err.Error(), "address already in use") ||
			strings.Contains(err.Error(), "port is already allocated") {
			return fmt.Errorf("网关端口 %s 或 %s 已被占用，请先停止占用该端口的服务（如系统自带 nginx/apache），"+
				"或在面板「设置 → 网关」中更换网关端口: %w",
				g.cfg.GatewayHTTPPort, g.cfg.GatewayHTTPSPort, err)
		}
		return fmt.Errorf("启动网关容器失败: %w", err)
	}

	if wasRunning {
		// 容器已在运行：主配置可能变化，校验后热重载，不重建容器。
		// 校验失败时恢复原主配置，避免把正在运行的网关留在不可重载状态。
		if err := g.testLocked(); err != nil {
			if hadMain {
				_ = writeFileInPlace(g.cfg.GatewayNginxConf(), oldMain, 0644)
			}
			return err
		}
		if err := g.reloadLocked(); err != nil {
			if hadMain {
				_ = writeFileInPlace(g.cfg.GatewayNginxConf(), oldMain, 0644)
				_ = g.reloadLocked()
			}
			return err
		}
		return nil
	}

	return g.waitReadyLocked()
}

// writeBaseConfig 写入主配置、默认站点、欢迎页与日志目录
func (g *GatewayService) writeBaseConfig() error {
	if err := os.MkdirAll(g.cfg.GatewayConfDir(), 0755); err != nil {
		return fmt.Errorf("创建网关配置目录失败: %w", err)
	}
	for _, d := range []string{g.cfg.GatewayCertDir(), g.cfg.GatewayLogDir(), g.cfg.GatewayHTMLDir()} {
		if err := os.MkdirAll(d, 0755); err != nil {
			return fmt.Errorf("创建网关目录失败: %w", err)
		}
	}
	if err := writeFileInPlace(g.cfg.GatewayNginxConf(), []byte(g.mainConf()), 0644); err != nil {
		return fmt.Errorf("写入网关主配置失败: %w", err)
	}
	if err := writeFileAtomic(filepath.Join(g.cfg.GatewayConfDir(), "default.conf"), []byte(g.defaultSiteConf()), 0644); err != nil {
		return fmt.Errorf("写入默认站点配置失败: %w", err)
	}
	if err := writeFileAtomic(filepath.Join(g.cfg.GatewayHTMLDir(), "index.html"), []byte(g.welcomePage()), 0644); err != nil {
		return fmt.Errorf("写入网关欢迎页失败: %w", err)
	}
	return nil
}

// waitReadyLocked 等待新容器中的 nginx 就绪并校验配置。
func (g *GatewayService) waitReadyLocked() error {
	var lastOut string
	for i := 0; i < 24; i++ {
		if g.rt.ContainerState(GatewayName) != "running" {
			return fmt.Errorf("网关容器启动后立即退出，请检查镜像 %s 是否可用", g.cfg.GatewayImage)
		}
		out, code, err := g.rt.ExecResult(GatewayName, "nginx", "-t")
		if err == nil && code == 0 {
			return nil
		}
		lastOut = out
		time.Sleep(250 * time.Millisecond)
	}
	if lastOut == "" {
		lastOut = "nginx -t 无输出"
	}
	return fmt.Errorf("网关配置校验失败: %s", lastOut)
}

// Reload 校验并热重载网关配置。配置错误时保留旧配置继续运行，
// 绝不通过重启容器来掩盖配置问题（否则会导致所有站点一起宕机）。
func (g *GatewayService) Reload() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.rt.ContainerState(GatewayName) != "running" {
		return g.ensureLocked(nil)
	}
	if err := g.testLocked(); err != nil {
		return err
	}
	return g.reloadLocked()
}

func (g *GatewayService) testLocked() error {
	out, code, err := g.rt.ExecResult(GatewayName, "nginx", "-t")
	if err != nil {
		return fmt.Errorf("网关不可执行 nginx -t: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("nginx 配置校验失败: %s", strings.TrimSpace(out))
	}
	return nil
}

func (g *GatewayService) reloadLocked() error {
	out, code, err := g.rt.ExecResult(GatewayName, "nginx", "-s", "reload")
	if err != nil {
		return fmt.Errorf("重载网关失败: %w", err)
	}
	if code != 0 {
		return fmt.Errorf("重载网关失败: %s", strings.TrimSpace(out))
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

// WriteSiteConf 写入站点配置。写入前备份，nginx -t 失败时回滚。
func (g *GatewayService) WriteSiteConf(domain, content string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	path := filepath.Join(g.cfg.GatewayConfDir(), SiteConfName(domain))
	backup, hadOld := readFileIfExists(path)

	if err := writeFileAtomic(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入站点配置失败: %w", err)
	}

	restore := func() {
		if hadOld {
			_ = writeFileAtomic(path, backup, 0644)
		} else {
			_ = os.Remove(path)
		}
	}

	if g.rt.ContainerState(GatewayName) != "running" {
		if err := g.ensureLocked(nil); err != nil {
			restore()
			return err
		}
		return nil
	}

	if err := g.testLocked(); err != nil {
		restore()
		// 恢复后重新加载，确保运行中的配置仍与磁盘一致
		if hadOld {
			_ = g.reloadLocked()
		}
		return err
	}
	if err := g.reloadLocked(); err != nil {
		restore()
		if hadOld {
			_ = g.reloadLocked()
		}
		return err
	}
	return nil
}

// RemoveSiteConf 删除站点配置并重载网关
func (g *GatewayService) RemoveSiteConf(domain string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	path := filepath.Join(g.cfg.GatewayConfDir(), SiteConfName(domain))
	backup, hadOld := readFileIfExists(path)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	if g.rt.ContainerState(GatewayName) != "running" {
		return nil
	}
	if err := g.testLocked(); err != nil {
		if hadOld {
			_ = writeFileAtomic(path, backup, 0644)
		}
		return err
	}
	if err := g.reloadLocked(); err != nil {
		if hadOld {
			_ = writeFileAtomic(path, backup, 0644)
			_ = g.reloadLocked()
		}
		return err
	}
	return nil
}

func readFileIfExists(path string) ([]byte, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// SiteConfName 站点配置文件名（只保留安全字符）
func SiteConfName(domain string) string {
	domain = strings.ReplaceAll(domain, "*", "_wildcard_")
	var b strings.Builder
	for _, r := range domain {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '.', r == '-', r == '_':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	name := strings.Trim(b.String(), "._-")
	if name == "" {
		name = "site"
	}
	return name + ".conf"
}

// Slug 把域名转换为可用于容器名的安全字符串。
// Docker 容器名只允许 [a-zA-Z0-9][a-zA-Z0-9_.-]*，因此通配符、冒号等都需要替换。
func Slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if strings.HasPrefix(s, "*.") {
		s = "w-" + strings.TrimPrefix(s, "*.")
	}
	s = strings.ReplaceAll(s, "*", "wildcard.")
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if len(out) > 48 {
		out = strings.Trim(out[:48], "-")
	}
	if out == "" {
		out = "site"
	}
	return out
}
