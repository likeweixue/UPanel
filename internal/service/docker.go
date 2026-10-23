package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/docker/docker/client"
)

// DockerConfig 面板展示 / 编辑的 Docker 守护进程配置子集。
// 注意：daemon.json 的真实字段是 kebab-case，这里做一层映射，
// 并且在保存时合并而不是覆盖整个文件。
type DockerConfig struct {
	RegistryMirrors        []string `json:"registry_mirrors"`
	InsecureRegistries     []string `json:"insecure_registries,omitempty"`
	MaxConcurrentDownloads int      `json:"max_concurrent_downloads,omitempty"`
	LogDriver              string   `json:"log-driver,omitempty"`
}

type DockerService struct {
	configPath string
}

func NewDockerService() *DockerService {
	configPath := ""
	if runtime.GOOS == "linux" {
		configPath = "/etc/docker/daemon.json"
	} else if runtime.GOOS == "darwin" {
		homeDir, _ := os.UserHomeDir()
		configPath = filepath.Join(homeDir, ".docker", "daemon.json")
	}
	return &DockerService{configPath: configPath}
}

func (s *DockerService) readRaw() (map[string]interface{}, error) {
	raw := map[string]interface{}{}
	if s.configPath == "" {
		return raw, fmt.Errorf("当前操作系统不支持编辑 Docker 配置")
	}
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return raw, nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return raw, nil
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("现有 %s 不是合法 JSON，为避免破坏配置已中止: %w", s.configPath, err)
	}
	return raw, nil
}

// GetConfig 获取当前 Docker 配置
func (s *DockerService) GetConfig() (*DockerConfig, error) {
	raw, err := s.readRaw()
	if err != nil {
		return nil, err
	}
	config := &DockerConfig{RegistryMirrors: []string{}}
	if v, ok := raw["registry-mirrors"].([]interface{}); ok {
		for _, item := range v {
			if str, ok := item.(string); ok {
				config.RegistryMirrors = append(config.RegistryMirrors, str)
			}
		}
	}
	if v, ok := raw["insecure-registries"].([]interface{}); ok {
		for _, item := range v {
			if str, ok := item.(string); ok {
				config.InsecureRegistries = append(config.InsecureRegistries, str)
			}
		}
	}
	if v, ok := raw["max-concurrent-downloads"].(float64); ok {
		config.MaxConcurrentDownloads = int(v)
	}
	if v, ok := raw["log-driver"].(string); ok {
		config.LogDriver = v
	}
	return config, nil
}

func validateRegistryList(list []string, label string) ([]string, error) {
	if len(list) > 10 {
		return nil, fmt.Errorf("%s 最多 10 个", label)
	}
	out := make([]string, 0, len(list))
	for _, item := range list {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if strings.ContainsAny(item, " \t\r\n\"'") {
			return nil, fmt.Errorf("%s 包含非法字符: %s", label, item)
		}
		if label == "镜像加速地址" && !strings.HasPrefix(item, "http://") && !strings.HasPrefix(item, "https://") {
			return nil, fmt.Errorf("镜像加速地址必须以 http:// 或 https:// 开头: %s", item)
		}
		out = append(out, strings.TrimRight(item, "/"))
	}
	return out, nil
}

// SaveConfig 合并保存 Docker 配置，保留 daemon.json 中的其它字段。
func (s *DockerService) SaveConfig(config *DockerConfig) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}
	if s.configPath == "" {
		return fmt.Errorf("当前操作系统不支持编辑 Docker 配置")
	}
	mirrors, err := validateRegistryList(config.RegistryMirrors, "镜像加速地址")
	if err != nil {
		return err
	}
	insecure, err := validateRegistryList(config.InsecureRegistries, "非安全仓库")
	if err != nil {
		return err
	}
	if config.MaxConcurrentDownloads < 0 || config.MaxConcurrentDownloads > 64 {
		return fmt.Errorf("最大并发下载数必须在 0-64 之间")
	}
	if config.LogDriver != "" {
		switch config.LogDriver {
		case "json-file", "local", "journald", "syslog", "none":
		default:
			return fmt.Errorf("不支持的日志驱动: %s", config.LogDriver)
		}
	}

	raw, err := s.readRaw()
	if err != nil {
		return err
	}
	if mirrors == nil {
		mirrors = []string{}
	}
	raw["registry-mirrors"] = mirrors
	if len(insecure) > 0 {
		raw["insecure-registries"] = insecure
	} else {
		delete(raw, "insecure-registries")
	}
	if config.MaxConcurrentDownloads > 0 {
		raw["max-concurrent-downloads"] = config.MaxConcurrentDownloads
	} else {
		delete(raw, "max-concurrent-downloads")
	}
	if config.LogDriver != "" {
		raw["log-driver"] = config.LogDriver
	} else {
		delete(raw, "log-driver")
	}

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.configPath), 0755); err != nil {
		return err
	}
	// 备份原文件，便于用户回滚
	if old, err := os.ReadFile(s.configPath); err == nil {
		_ = os.WriteFile(s.configPath+".upanel.bak", old, 0600)
	}
	return writeFileAtomic(s.configPath, append(data, '\n'), 0644)
}

// RestartDocker 重启 Docker 服务
func (s *DockerService) RestartDocker() error {
	switch runtime.GOOS {
	case "linux":
		out, err := exec.Command("systemctl", "restart", "docker").CombinedOutput()
		if err != nil {
			return fmt.Errorf("重启 Docker 失败: %v %s", err, strings.TrimSpace(string(out)))
		}
		return nil
	case "darwin":
		_ = exec.Command("osascript", "-e", `quit app "Docker"`).Run()
		if err := exec.Command("open", "-a", "Docker").Run(); err != nil {
			return fmt.Errorf("启动 Docker Desktop 失败: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("当前操作系统不支持从面板重启 Docker")
	}
}

// Version 返回 Docker 守护进程版本
func (s *DockerService) Version() (string, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return "", fmt.Errorf("无法连接 Docker: %w", err)
	}
	defer cli.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	version, err := cli.ServerVersion(ctx)
	if err != nil {
		return "", fmt.Errorf("Docker 守护进程无响应: %w", err)
	}
	return version.Version, nil
}

// GetMirrors 获取常用镜像加速源
func (s *DockerService) GetMirrors() []map[string]string {
	return []map[string]string{
		{"name": "阿里云", "url": "https://your-id.mirror.aliyuncs.com", "description": "需要注册阿里云账号获取专属加速地址"},
		{"name": "中科大", "url": "https://docker.mirrors.ustc.edu.cn", "description": "中国科学技术大学镜像站"},
		{"name": "网易", "url": "https://hub-mirror.c.163.com", "description": "网易镜像站"},
		{"name": "百度云", "url": "https://mirror.baidubce.com", "description": "百度云镜像站"},
		{"name": "1Panel", "url": "https://docker.1panel.live", "description": "1Panel 提供的镜像加速"},
	}
}
