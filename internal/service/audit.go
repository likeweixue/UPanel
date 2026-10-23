package service

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"upanel/internal/config"
)

// LoginAttempt 一次登录尝试的审计记录
type LoginAttempt struct {
	Time      time.Time `json:"time"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent"`
	Username  string    `json:"username"`
	Success   bool      `json:"success"`
	Reason    string    `json:"reason,omitempty"`
}

var (
	auditMu      sync.Mutex
	auditEntries = make([]LoginAttempt, 0, 256)
	auditOnce    sync.Once
)

const auditMaxEntries = 500

// RecordLoginAttempt 记录登录成功/失败，并追加到 data/logs/auth.log。
func RecordLoginAttempt(ip, userAgent, username string, success bool, reason string) {
	auditOnce.Do(func() {
		loadAuditFromDisk()
	})

	entry := LoginAttempt{
		Time:      time.Now(),
		IP:        truncate(ip, 64),
		UserAgent: truncate(userAgent, 300),
		Username:  truncate(username, 64),
		Success:   success,
		Reason:    truncate(reason, 200),
	}

	auditMu.Lock()
	auditEntries = append(auditEntries, entry)
	if len(auditEntries) > auditMaxEntries {
		auditEntries = append([]LoginAttempt{}, auditEntries[len(auditEntries)-auditMaxEntries:]...)
	}
	auditMu.Unlock()

	appendAuditToDisk(entry)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// RecentLoginAttempts 返回最近的登录记录（倒序）
func RecentLoginAttempts(limit int) []LoginAttempt {
	auditOnce.Do(func() {
		loadAuditFromDisk()
	})
	if limit <= 0 || limit > auditMaxEntries {
		limit = 100
	}
	auditMu.Lock()
	defer auditMu.Unlock()
	n := len(auditEntries)
	if n == 0 {
		return []LoginAttempt{}
	}
	start := n - limit
	if start < 0 {
		start = 0
	}
	out := make([]LoginAttempt, 0, n-start)
	for i := n - 1; i >= start; i-- {
		out = append(out, auditEntries[i])
	}
	return out
}

// LoginStats 返回窗口内的失败/成功次数以及最近一次失败时间
func LoginStats(window time.Duration) (failures int, successes int, lastFailure *time.Time, lastSuccess *time.Time) {
	auditOnce.Do(func() {
		loadAuditFromDisk()
	})
	cutoff := time.Now().Add(-window)
	auditMu.Lock()
	defer auditMu.Unlock()
	for _, e := range auditEntries {
		if e.Time.Before(cutoff) {
			continue
		}
		if e.Success {
			successes++
			t := e.Time
			lastSuccess = &t
		} else {
			failures++
			t := e.Time
			lastFailure = &t
		}
	}
	return
}

func auditLogPath() string {
	return filepath.Join(config.Current().DataPath, "logs", "auth.log")
}

func loadAuditFromDisk() {
	path := auditLogPath()
	info, err := os.Stat(path)
	if err != nil || info.Size() > 4<<20 {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := splitLines(string(data))
	start := len(lines) - auditMaxEntries
	if start < 0 {
		start = 0
	}
	for _, line := range lines[start:] {
		if line == "" {
			continue
		}
		var e LoginAttempt
		if err := json.Unmarshal([]byte(line), &e); err == nil {
			auditEntries = append(auditEntries, e)
		}
	}
	if len(auditEntries) > auditMaxEntries {
		auditEntries = auditEntries[len(auditEntries)-auditMaxEntries:]
	}
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

func appendAuditToDisk(entry LoginAttempt) {
	path := auditLogPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return
	}
	// 简单轮转：超过 4MB 时保留最近一半
	if info, err := os.Stat(path); err == nil && info.Size() > 4<<20 {
		if data, err := os.ReadFile(path); err == nil {
			half := len(data) / 2
			_ = os.WriteFile(path, data[half:], 0600)
		}
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
}
