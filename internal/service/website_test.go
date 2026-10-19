package service

import (
	"strings"
	"testing"

	"upanel/internal/config"
)

func testService() *WebsiteService {
	cfg := config.NewDefaultConfig("/tmp/upanel-test")
	_ = cfg.EnsureDirs()
	return &WebsiteService{cfg: cfg}
}

func TestValidateDomain(t *testing.T) {
	cases := []struct {
		domain string
		ok     bool
	}{
		{"example.com", true},
		{"test.example.com", true},
		{"my-site.com", true},
		{"*.example.com", true},
		{"", false},
		{"http://example.com", false},
		{"example.com/path", false},
		{"bad domain", false},
	}
	for _, c := range cases {
		err := ValidateDomain(c.domain)
		if (err == nil) != c.ok {
			t.Errorf("ValidateDomain(%q) = %v, want ok=%v", c.domain, err, c.ok)
		}
	}
}

func TestSlug(t *testing.T) {
	if got := Slug("example.com"); got != "example-com" {
		t.Errorf("Slug(example.com) = %q", got)
	}
	if got := Slug("WWW.Example.COM"); got != "www-example-com" {
		t.Errorf("Slug(WWW.Example.COM) = %q", got)
	}
}

// 静态站点：网关直接托管文件，不应出现 fastcgi / proxy_pass
func TestRenderConfStatic(t *testing.T) {
	s := testService()
	site := &Website{Domain: "demo.com", Type: SiteTypeStatic, Aliases: []string{"www.demo.com"}}
	conf := s.renderConf(site)

	must := []string{
		"listen 80",
		"server_name demo.com www.demo.com",
		"/var/www/websites/demo.com/html",
		"try_files",
	}
	for _, m := range must {
		if !strings.Contains(conf, m) {
			t.Errorf("静态站点配置缺少 %q\n%s", m, conf)
		}
	}
	if strings.Contains(conf, "fastcgi_pass") || strings.Contains(conf, "proxy_pass") {
		t.Errorf("静态站点不应包含 fastcgi_pass / proxy_pass")
	}
}

// PHP 站点：必须 fastcgi 到 php-fpm 容器，且使用变量以支持延迟解析
func TestRenderConfPHP(t *testing.T) {
	s := testService()
	site := &Website{Domain: "php.demo.com", Type: SiteTypePHP, PHPVersion: "8.2"}
	conf := s.renderConf(site)

	must := []string{
		"listen 80",
		"set $upanel_php \"upanel-php-php-demo-com:9000\"",
		"fastcgi_pass $upanel_php",
		"SCRIPT_FILENAME",
		"index.php",
	}
	for _, m := range must {
		if !strings.Contains(conf, m) {
			t.Errorf("PHP 站点配置缺少 %q\n%s", m, conf)
		}
	}
}

// 反代站点：proxy_pass 到站点容器
func TestRenderConfProxy(t *testing.T) {
	s := testService()
	site := &Website{Domain: "app.demo.com", Type: SiteTypeProxy, Port: 3000}
	conf := s.renderConf(site)

	must := []string{
		"set $upanel_backend \"http://upanel-site-app-demo-com:3000\"",
		"proxy_pass $upanel_backend",
		"X-Forwarded-For",
	}
	for _, m := range must {
		if !strings.Contains(conf, m) {
			t.Errorf("反代站点配置缺少 %q\n%s", m, conf)
		}
	}
}

// SSL：开启后应同时有 443 与证书指令；强制跳转时 80 返回 301
func TestRenderConfSSL(t *testing.T) {
	s := testService()

	site := &Website{Domain: "ssl.demo.com", Type: SiteTypeStatic, SSLEnabled: true}
	conf := s.renderConf(site)
	if !strings.Contains(conf, "listen 443 ssl") {
		t.Errorf("未生成 443 监听")
	}
	if !strings.Contains(conf, "/etc/nginx/certs/ssl.demo.com/fullchain.pem") {
		t.Errorf("未指向证书文件")
	}

	site.ForceSSL = true
	conf = s.renderConf(site)
	if !strings.Contains(conf, "return 301 https://$host$request_uri") {
		t.Errorf("强制 HTTPS 时未生成 301 跳转")
	}
}

// 网关主配置必须包含 Docker 内嵌 DNS 解析器，否则后端容器未启动时 nginx 会加载失败
func TestGatewayMainConf(t *testing.T) {
	g := &GatewayService{cfg: config.NewDefaultConfig("/tmp/upanel-test")}
	conf := g.mainConf()
	if !strings.Contains(conf, "resolver 127.0.0.11") {
		t.Errorf("网关主配置缺少 resolver，后端容器未启动时会加载失败")
	}
	if !strings.Contains(conf, "include /etc/nginx/conf.d/*.conf") {
		t.Errorf("网关主配置未包含 conf.d")
	}
}
