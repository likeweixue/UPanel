package service

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SSHStatus SSH 服务只读状态
type SSHStatus struct {
	Supported              bool     `json:"supported"`
	Installed              bool     `json:"installed"`
	ServiceActive          bool     `json:"serviceActive"`
	ServiceName            string   `json:"serviceName"`
	Port                   string   `json:"port"`
	PermitRootLogin        string   `json:"permitRootLogin"`
	PasswordAuthentication string   `json:"passwordAuthentication"`
	PubkeyAuthentication   string   `json:"pubkeyAuthentication"`
	ConfigPath             string   `json:"configPath"`
	Warnings               []string `json:"warnings"`
	Error                  string   `json:"error,omitempty"`
}

// GetSSHStatus 读取 sshd 配置与服务状态（不修改任何配置）
func GetSSHStatus() SSHStatus {
	st := SSHStatus{Port: "22", Supported: runtime.GOOS == "linux"}
	if runtime.GOOS != "linux" {
		st.Error = "SSH 状态仅在 Linux 服务器上可用"
		return st
	}

	configPath := "/etc/ssh/sshd_config"
	if _, err := os.Stat(configPath); err != nil {
		st.Error = "未检测到 /etc/ssh/sshd_config"
		return st
	}
	st.Installed = true
	st.ConfigPath = configPath

	values := parseSSHDConfig(configPath)
	if v, ok := values["Port"]; ok {
		st.Port = v
	}
	st.PermitRootLogin = firstValue(values, "PermitRootLogin", "未设置")
	st.PasswordAuthentication = firstValue(values, "PasswordAuthentication", "未设置")
	st.PubkeyAuthentication = firstValue(values, "PubkeyAuthentication", "未设置")

	for _, name := range []string{"sshd", "ssh"} {
		if serviceActive(name) {
			st.ServiceActive = true
			st.ServiceName = name
			break
		}
	}

	if strings.EqualFold(st.PasswordAuthentication, "yes") && !strings.EqualFold(st.PermitRootLogin, "no") {
		st.Warnings = append(st.Warnings, "SSH 允许密码登录，建议改用密钥登录并禁用 PermitRootLogin")
	}
	if strings.EqualFold(st.PermitRootLogin, "yes") {
		st.Warnings = append(st.Warnings, "SSH 允许 root 直接登录，建议设置为 prohibit-password")
	}
	if st.Port == "22" {
		st.Warnings = append(st.Warnings, "SSH 使用默认 22 端口，公网暴露时建议修改端口或限制来源 IP")
	}
	return st
}

func firstValue(m map[string]string, key, def string) string {
	if v, ok := m[key]; ok && v != "" {
		return v
	}
	return def
}

// parseSSHDConfig 解析 sshd_config 及 conf.d 目录（首个出现的值生效）
func parseSSHDConfig(path string) map[string]string {
	values := map[string]string{}
	files := []string{path}
	if matches, err := filepath.Glob("/etc/ssh/sshd_config.d/*.conf"); err == nil {
		sort.Strings(matches)
		files = append(matches, files...)
	}
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		for _, raw := range strings.Split(string(data), "\n") {
			line := strings.TrimSpace(raw)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			key := fields[0]
			val := strings.Join(fields[1:], " ")
			if _, exists := values[key]; !exists {
				values[key] = val
			}
		}
	}
	return values
}

func serviceActive(name string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "systemctl", "is-active", name).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}

// FirewallStatus 防火墙只读状态
type FirewallStatus struct {
	Supported bool     `json:"supported"`
	Backend   string   `json:"backend"`
	Active    bool     `json:"active"`
	Rules     []string `json:"rules"`
	Summary   string   `json:"summary"`
	Error     string   `json:"error,omitempty"`
}

// GetFirewallStatus 读取 ufw / firewalld 状态。只读，不修改任何规则。
func GetFirewallStatus() FirewallStatus {
	st := FirewallStatus{Supported: runtime.GOOS == "linux", Rules: []string{}}
	if runtime.GOOS != "linux" {
		st.Error = "防火墙状态仅在 Linux 服务器上可用"
		return st
	}

	if _, err := exec.LookPath("ufw"); err == nil {
		st.Backend = "ufw"
		out, err := runCmd(5*time.Second, "ufw", "status", "numbered")
		if err != nil {
			st.Error = strings.TrimSpace(out)
			return st
		}
		st.Summary = strings.TrimSpace(out)
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "[") {
				st.Rules = append(st.Rules, line)
			}
			if strings.HasPrefix(line, "Status: active") {
				st.Active = true
			}
		}
		return st
	}

	if _, err := exec.LookPath("firewall-cmd"); err == nil {
		st.Backend = "firewalld"
		out, err := runCmd(5*time.Second, "firewall-cmd", "--state")
		if err != nil {
			st.Error = strings.TrimSpace(out)
			return st
		}
		st.Active = strings.TrimSpace(out) == "running"
		if st.Active {
			detail, _ := runCmd(5*time.Second, "firewall-cmd", "--list-all")
			st.Summary = strings.TrimSpace(detail)
			for _, line := range strings.Split(detail, "\n") {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "ports:") || strings.HasPrefix(line, "services:") || strings.HasPrefix(line, "interfaces:") {
					st.Rules = append(st.Rules, line)
				}
			}
		}
		return st
	}

	// 最后尝试 nftables（只取规则数量，避免返回超大内容）
	if _, err := exec.LookPath("nft"); err == nil {
		st.Backend = "nftables"
		out, err := runCmd(5*time.Second, "nft", "list", "ruleset")
		if err == nil {
			st.Active = strings.Contains(out, "table ")
			count := strings.Count(out, "\n")
			st.Summary = "nftables 规则行数: " + strconv.Itoa(count)
			return st
		}
	}

	st.Backend = "none"
	st.Error = "未检测到 ufw / firewalld / nftables，或当前容器缺少相应命令"
	return st
}

func runCmd(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}
