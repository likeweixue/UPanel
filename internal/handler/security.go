package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"upanel/internal/config"
	"upanel/internal/service"
)

// SecurityHandler 安全中心接口
type SecurityHandler struct{}

func NewSecurityHandler() *SecurityHandler { return &SecurityHandler{} }

type securityOverview struct {
	SecurityEntry     string                 `json:"securityEntry"`
	BindIP            string                 `json:"bindIP"`
	PanelPort         string                 `json:"panelPort"`
	TotalSites        int                    `json:"totalSites"`
	HTTPSSites        int                    `json:"httpsSites"`
	InitialPassword   bool                   `json:"initialPasswordExists"`
	LoginFailures24h  int                    `json:"loginFailures24h"`
	LoginSuccesses24h int                    `json:"loginSuccesses24h"`
	LastFailure       *time.Time             `json:"lastFailure,omitempty"`
	LastSuccess       *time.Time             `json:"lastSuccess,omitempty"`
	Warnings          []string               `json:"warnings"`
	Firewall          service.FirewallStatus `json:"firewall"`
	SSH               service.SSHStatus      `json:"ssh"`
}

// Overview 安全概览
func (h *SecurityHandler) Overview(c *gin.Context) {
	cfg := config.Current()
	overview := securityOverview{
		SecurityEntry: cfg.GetSecurityEntry(),
		BindIP:        cfg.BindIP,
		PanelPort:     cfg.Port,
		Warnings:      []string{},
		Firewall:      service.GetFirewallStatus(),
		SSH:           service.GetSSHStatus(),
	}

	// 站点统计（直接读取 site.json，Docker 不可用时也能展示）
	if entries, err := os.ReadDir(cfg.WebsitePath); err == nil {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			data, err := os.ReadFile(filepath.Join(cfg.WebsitePath, e.Name(), "site.json"))
			if err != nil {
				continue
			}
			var site service.Website
			if err := json.Unmarshal(data, &site); err != nil {
				continue
			}
			overview.TotalSites++
			if site.SSLEnabled {
				overview.HTTPSSites++
			}
		}
	}

	if _, err := os.Stat(cfg.InitialPasswordFile()); err == nil {
		overview.InitialPassword = true
	}
	overview.LoginFailures24h, overview.LoginSuccesses24h, overview.LastFailure, overview.LastSuccess =
		service.LoginStats(24 * time.Hour)

	if overview.SecurityEntry == "" {
		overview.Warnings = append(overview.Warnings, "未启用安全入口，知道面板地址的人都可以直接访问登录页")
	}
	if cfg.BindIP == "0.0.0.0" || cfg.BindIP == "::" || cfg.BindIP == "" {
		overview.Warnings = append(overview.Warnings, "面板监听所有网卡，建议用防火墙限制面板端口的来源 IP")
	}
	if overview.InitialPassword {
		overview.Warnings = append(overview.Warnings, "首次启动生成的密码文件仍然存在，登录后请删除 data/initial-password.txt")
	}
	if overview.TotalSites > 0 && overview.HTTPSSites == 0 {
		overview.Warnings = append(overview.Warnings, "已有站点但没有任何站点启用 HTTPS")
	}
	if !overview.Firewall.Active {
		overview.Warnings = append(overview.Warnings, "未检测到已启用的防火墙（ufw/firewalld），请确认云安全组规则")
	}
	if overview.LoginFailures24h >= 5 {
		overview.Warnings = append(overview.Warnings, "最近 24 小时登录失败次数较多，请检查是否遭受暴力破解")
	}
	for _, w := range overview.SSH.Warnings {
		overview.Warnings = append(overview.Warnings, "SSH: "+w)
	}

	c.JSON(http.StatusOK, gin.H{"data": overview})
}

// Audit 登录审计
func (h *SecurityHandler) Audit(c *gin.Context) {
	limit := 100
	if v := c.Query("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	failures, successes, lastFailure, lastSuccess := service.LoginStats(24 * time.Hour)
	c.JSON(http.StatusOK, gin.H{
		"data": service.RecentLoginAttempts(limit),
		"stats": gin.H{
			"failures24h":  failures,
			"successes24h": successes,
			"lastFailure":  lastFailure,
			"lastSuccess":  lastSuccess,
		},
	})
}
