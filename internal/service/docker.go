package service

import (
	"encoding/json"
	"os"
	"os/exec"
	"runtime"
)

type DockerConfig struct {
	RegistryMirrors []string `json:"registry_mirrors"`
	InsecureRegistries []string `json:"insecure_registries,omitempty"`
	MaxConcurrentDownloads int `json:"max_concurrent_downloads,omitempty"`
	LogDriver string `json:"log-driver,omitempty"`
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
		configPath = homeDir + "/.docker/daemon.json"
	}
	
	return &DockerService{
		configPath: configPath,
	}
}

// 获取当前 Docker 配置
func (s *DockerService) GetConfig() (*DockerConfig, error) {
	config := &DockerConfig{
		RegistryMirrors: []string{},
	}
	
	data, err := os.ReadFile(s.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, err
	}
	
	if err := json.Unmarshal(data, config); err != nil {
		return nil, err
	}
	
	return config, nil
}

// 保存 Docker 配置
func (s *DockerService) SaveConfig(config *DockerConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(s.configPath, data, 0644)
}

// 重启 Docker 服务
func (s *DockerService) RestartDocker() error {
	var cmd *exec.Cmd
	if runtime.GOOS == "linux" {
		cmd = exec.Command("systemctl", "restart", "docker")
	} else if runtime.GOOS == "darwin" {
		// macOS 通过 osascript 重启 Docker Desktop
		cmd = exec.Command("osascript", "-e", `quit app "Docker"`)
		cmd.Run()
		cmd = exec.Command("open", "-a", "Docker")
	}
	return cmd.Run()
}

// 获取常用镜像加速源
func (s *DockerService) GetMirrors() []map[string]string {
	return []map[string]string{
		{"name": "阿里云", "url": "https://your-id.mirror.aliyuncs.com", "description": "需要注册阿里云账号获取专属加速地址"},
		{"name": "中科大", "url": "https://docker.mirrors.ustc.edu.cn", "description": "中国科学技术大学镜像站"},
		{"name": "网易", "url": "https://hub-mirror.c.163.com", "description": "网易镜像站"},
		{"name": "百度云", "url": "https://mirror.baidubce.com", "description": "百度云镜像站"},
		{"name": "Docker 中国", "url": "https://registry.docker-cn.com", "description": "Docker 官方中国镜像"},
		{"name": "1Panel", "url": "https://docker.1panel.live", "description": "1Panel 提供的镜像加速"},
	}
}
