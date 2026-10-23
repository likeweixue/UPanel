package service

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/mail"
	"net/smtp"
	"os"
	"path/filepath"
	"strings"
	"time"

	"upanel/internal/config"
)

// EmailConfig SMTP 通知配置
type EmailConfig struct {
	Enabled    bool     `json:"enabled"`
	Host       string   `json:"host"`
	Port       int      `json:"port"`
	Username   string   `json:"username"`
	Password   string   `json:"password,omitempty"`
	From       string   `json:"from"`
	To         []string `json:"to"`
	Encryption string   `json:"encryption"` // none / starttls / ssl
}

// EmailConfigView 返回给前端的配置（不包含密码明文）
type EmailConfigView struct {
	Enabled     bool     `json:"enabled"`
	Host        string   `json:"host"`
	Port        int      `json:"port"`
	Username    string   `json:"username"`
	From        string   `json:"from"`
	To          []string `json:"to"`
	Encryption  string   `json:"encryption"`
	HasPassword bool     `json:"hasPassword"`
}

func emailFilePath() string {
	return filepath.Join(config.Current().DataPath, "email.json")
}

func loadEmailConfig() (*EmailConfig, error) {
	cfg := &EmailConfig{Port: 587, Encryption: "starttls", To: []string{}}
	data, err := os.ReadFile(emailFilePath())
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, err
	}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("邮件配置文件损坏: %w", err)
	}
	if cfg.To == nil {
		cfg.To = []string{}
	}
	return cfg, nil
}

func saveEmailConfig(cfg *EmailConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(emailFilePath()), 0755); err != nil {
		return err
	}
	return writeFileAtomic(emailFilePath(), data, 0600)
}

// GetEmailConfigView 读取脱敏后的邮件配置
func GetEmailConfigView() (*EmailConfigView, error) {
	cfg, err := loadEmailConfig()
	if err != nil {
		return nil, err
	}
	return &EmailConfigView{
		Enabled:     cfg.Enabled,
		Host:        cfg.Host,
		Port:        cfg.Port,
		Username:    cfg.Username,
		From:        cfg.From,
		To:          cfg.To,
		Encryption:  cfg.Encryption,
		HasPassword: cfg.Password != "",
	}, nil
}

func normalizeEmailConfig(cfg *EmailConfig) error {
	cfg.Host = strings.TrimSpace(cfg.Host)
	cfg.Username = strings.TrimSpace(cfg.Username)
	cfg.From = strings.TrimSpace(cfg.From)
	cfg.Encryption = strings.ToLower(strings.TrimSpace(cfg.Encryption))
	if cfg.Encryption == "" {
		cfg.Encryption = "starttls"
	}
	switch cfg.Encryption {
	case "none", "starttls", "ssl":
	default:
		return fmt.Errorf("加密方式必须是 none / starttls / ssl")
	}
	if cfg.Host == "" {
		return fmt.Errorf("SMTP 服务器地址不能为空")
	}
	if strings.ContainsAny(cfg.Host, " \t\r\n/") {
		return fmt.Errorf("SMTP 服务器地址不合法")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("SMTP 端口必须是 1-65535 之间的数字")
	}
	if cfg.From == "" {
		return fmt.Errorf("发件人地址不能为空")
	}
	if _, err := mail.ParseAddress(cfg.From); err != nil {
		return fmt.Errorf("发件人地址无效: %w", err)
	}
	if len(cfg.To) == 0 {
		return fmt.Errorf("至少需要一个收件人")
	}
	if len(cfg.To) > 10 {
		return fmt.Errorf("收件人最多 10 个")
	}
	cleaned := make([]string, 0, len(cfg.To))
	for _, to := range cfg.To {
		to = strings.TrimSpace(to)
		if to == "" {
			continue
		}
		if _, err := mail.ParseAddress(to); err != nil {
			return fmt.Errorf("收件人地址无效: %s", to)
		}
		cleaned = append(cleaned, to)
	}
	cfg.To = cleaned
	if len(cfg.To) == 0 {
		return fmt.Errorf("至少需要一个有效收件人")
	}
	if strings.ContainsAny(cfg.Username+cfg.Password, "\r\n\x00") {
		return fmt.Errorf("账号或密码包含非法字符")
	}
	return nil
}

// UpdateEmailConfig 保存邮件配置。密码留空时保留原密码。
func UpdateEmailConfig(patch *EmailConfig) (*EmailConfigView, error) {
	if patch == nil {
		return nil, fmt.Errorf("配置不能为空")
	}
	old, err := loadEmailConfig()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(patch.Password) == "" {
		patch.Password = old.Password
	}
	if err := normalizeEmailConfig(patch); err != nil {
		return nil, err
	}
	if err := saveEmailConfig(patch); err != nil {
		return nil, err
	}
	return GetEmailConfigView()
}

// SendTestEmail 发送测试邮件。password 为空时使用已保存密码。
func SendTestEmail(patch *EmailConfig) error {
	cfg := patch
	if cfg == nil {
		loaded, err := loadEmailConfig()
		if err != nil {
			return err
		}
		cfg = loaded
	} else if strings.TrimSpace(cfg.Password) == "" {
		old, err := loadEmailConfig()
		if err != nil {
			return err
		}
		cfg.Password = old.Password
	}
	if err := normalizeEmailConfig(cfg); err != nil {
		return err
	}
	subject := "UPanel SMTP 测试邮件"
	body := fmt.Sprintf("这是一封来自 UPanel 的测试邮件。\r\n\r\n发送时间: %s\r\n如果你收到这封邮件，说明 SMTP 配置正确。\r\n",
		time.Now().Format("2006-01-02 15:04:05"))
	return sendMail(cfg, subject, body)
}

func sendMail(cfg *EmailConfig, subject, body string) error {
	from, err := mail.ParseAddress(cfg.From)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	dialer := &net.Dialer{Timeout: 10 * time.Second}

	var conn net.Conn
	if cfg.Encryption == "ssl" {
		conn, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
			ServerName: cfg.Host,
			MinVersion: tls.VersionTLS12,
		})
	} else {
		conn, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return fmt.Errorf("连接 SMTP 服务器失败: %w", err)
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(30 * time.Second))

	client, err := smtp.NewClient(conn, cfg.Host)
	if err != nil {
		return fmt.Errorf("创建 SMTP 客户端失败: %w", err)
	}
	defer client.Close()

	if cfg.Encryption == "starttls" {
		ok, _ := client.Extension("STARTTLS")
		if !ok {
			return fmt.Errorf("SMTP 服务器不支持 STARTTLS，请改用 SSL 或检查服务商配置")
		}
		if err := client.StartTLS(&tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}); err != nil {
			return fmt.Errorf("STARTTLS 失败: %w", err)
		}
	}
	if cfg.Username != "" {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			return fmt.Errorf("SMTP 认证失败: %w", err)
		}
	}
	if err := client.Mail(from.Address); err != nil {
		return fmt.Errorf("设置发件人失败: %w", err)
	}
	for _, to := range cfg.To {
		addr, err := mail.ParseAddress(to)
		if err != nil {
			return err
		}
		if err := client.Rcpt(addr.Address); err != nil {
			return fmt.Errorf("设置收件人失败: %w", err)
		}
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("开始写入邮件失败: %w", err)
	}
	message := "From: " + from.String() + "\r\n" +
		"To: " + strings.Join(cfg.To, ", ") + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"Date: " + time.Now().Format(time.RFC1123Z) + "\r\n" +
		"\r\n" + body + "\r\n"
	if _, err := w.Write([]byte(message)); err != nil {
		_ = w.Close()
		return fmt.Errorf("写入邮件内容失败: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("提交邮件失败: %w", err)
	}
	return client.Quit()
}
