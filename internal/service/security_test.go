package service

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRenderConfForceSSLHasSingleHTTPBlock(t *testing.T) {
	s := testService()
	site := &Website{Domain: "ssl.demo.com", Type: SiteTypeStatic, SSLEnabled: true, ForceSSL: true}
	conf := s.renderConf(site)
	if got := strings.Count(conf, "listen 80;"); got != 1 {
		t.Fatalf("强制 HTTPS 时 listen 80 应只有 1 处，实际 %d\n%s", got, conf)
	}
	if !strings.Contains(conf, "return 301 https://$host$request_uri") {
		t.Fatalf("缺少 301 跳转")
	}
	if strings.Contains(conf, "root /var/www/websites/ssl.demo.com/html;") == false {
		t.Fatalf("HTTPS 站点缺少 root")
	}
	if got := strings.Count(conf, "server {"); got != 2 {
		t.Fatalf("应只有跳转与 HTTPS 两个 server 块，实际 %d", got)
	}
}

func TestRenderConfDeniesDotfiles(t *testing.T) {
	s := testService()
	conf := s.renderConf(&Website{Domain: "demo.com", Type: SiteTypeStatic})
	if !strings.Contains(conf, `location ~ /\.(?!well-known)`) {
		t.Fatalf("静态站点缺少隐藏文件保护\n%s", conf)
	}
}

func TestValidateDomainRejectsInjection(t *testing.T) {
	bad := []string{
		"example.com; include /etc/passwd;",
		"example.com\nserver_name evil",
		"..",
		"foo..bar.com",
		"*.*.example.com",
		"-bad.com",
		"bad-.com",
	}
	for _, domain := range bad {
		if err := ValidateDomain(domain); err == nil {
			t.Errorf("ValidateDomain(%q) 应失败", domain)
		}
	}
}

func TestValidateAliasesRejectsDuplicate(t *testing.T) {
	s := testService()
	_, err := s.validateAliases("demo.com", []string{"demo.com"}, "")
	if err == nil {
		t.Fatal("别名与主域名重复时应报错")
	}
	aliases, err := s.validateAliases("demo.com", []string{"WWW.Demo.com", "www.demo.com"}, "")
	if err == nil {
		t.Fatal("归一化后重复的别名应报错")
	}
	_ = aliases
	if _, err := s.validateAliases("demo.com", []string{"bad alias"}, ""); err == nil {
		t.Fatal("含空格别名应报错")
	}
}

func TestCronParseAndMatch(t *testing.T) {
	spec, err := parseCron("*/5 * * * *")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 10, 24, 10, 10, 0, 0, time.Local)
	if !spec.matches(base) {
		t.Fatal("10:10 应匹配每 5 分钟")
	}
	if spec.matches(base.Add(3 * time.Minute)) {
		t.Fatal("10:13 不应匹配每 5 分钟")
	}

	weekly, err := parseCron("0 3 * * 1")
	if err != nil {
		t.Fatal(err)
	}
	monday := time.Date(2026, 10, 26, 3, 0, 0, 0, time.Local) // 周一
	if !weekly.matches(monday) {
		t.Fatal("周一 03:00 应匹配")
	}
	if weekly.matches(monday.Add(24 * time.Hour)) {
		t.Fatal("周二不应匹配每周一")
	}

	for _, bad := range []string{"", "* * *", "60 * * * *", "*/0 * * * *", "a b c d e"} {
		if _, err := parseCron(bad); err == nil {
			t.Errorf("parseCron(%q) 应报错", bad)
		}
	}
}

func TestAppRuntimeRedisUsesRequirePass(t *testing.T) {
	s := &AppService{}
	app, _ := s.find("redis")
	env, cmd, creds, err := s.buildRuntime(app, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(env) != 0 {
		t.Fatalf("redis 不应设置环境变量密码，实际 %v", env)
	}
	joined := strings.Join(cmd, " ")
	if !strings.Contains(joined, "--requirepass") || !strings.Contains(joined, creds["password"]) {
		t.Fatalf("redis 启动命令缺少 requirepass: %v", cmd)
	}
	if creds["password"] == "" || len(creds["password"]) < 16 {
		t.Fatalf("redis 密码过弱: %q", creds["password"])
	}
}

func TestFormatDuration(t *testing.T) {
	cases := map[time.Duration]string{
		30 * time.Second:             "0 分钟",
		2 * time.Minute:              "2 分钟",
		3*time.Hour + 2*time.Minute:  "3 小时 2 分钟",
		25*time.Hour + 4*time.Minute: "1 天 1 小时 4 分钟",
	}
	for d, want := range cases {
		if got := formatDuration(d); got != want {
			t.Errorf("formatDuration(%v) = %q, want %q", d, got, want)
		}
	}
}

func TestTailFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, "line-"+string(rune('a'+i)))
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := tailFile(path, 3, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "line-r") || !strings.Contains(out, "line-t") {
		t.Fatalf("tail 内容不正确: %q", out)
	}
	if strings.Contains(out, "line-a") {
		t.Fatalf("tail 不应包含首行: %q", out)
	}
}

func TestPHPScriptFilenameUsesContainerMount(t *testing.T) {
	s := testService()
	conf := s.renderConf(&Website{Domain: "php.demo.com", Type: SiteTypePHP, PHPVersion: "8.2"})
	if !strings.Contains(conf, "fastcgi_param SCRIPT_FILENAME /var/www/html$fastcgi_script_name;") {
		t.Fatalf("PHP SCRIPT_FILENAME 必须指向 php 容器内的挂载路径\n%s", conf)
	}
	if strings.Contains(conf, "$document_root$fastcgi_script_name") {
		t.Fatalf("不能使用网关容器的 $document_root 作为 SCRIPT_FILENAME")
	}
}

func TestSlugWildcardDoesNotCollide(t *testing.T) {
	if got := Slug("*.example.com"); got != "w-example-com" {
		t.Fatalf("Slug(*.example.com) = %q", got)
	}
	if Slug("*.example.com") == Slug("wildcard.example.com") {
		t.Fatal("通配域名与同名普通域名不应生成相同容器名")
	}
}

func TestDefaultIndexDoesNotOverwriteExisting(t *testing.T) {
	s := testService()
	root := t.TempDir()
	existing := "<html>existing</html>"
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}
	if err := s.writeDefaultIndex(SiteTypeStatic, root, "demo.com"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(root, "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != existing {
		t.Fatalf("已有首页被覆盖: %q", string(data))
	}
}
