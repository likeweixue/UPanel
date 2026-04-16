package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type Website struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Domain     string    `json:"domain"`
	Status     string    `json:"status"`
	Port       int       `json:"port"`
	PHPVersion string    `json:"php_version"`
	CreatedAt  time.Time `json:"created_at"`
	Path       string    `json:"path"`
}

type CreateWebsiteRequest struct {
	Domain     string `json:"domain" binding:"required"`
	PHPVersion string `json:"php_version"`
	CreateDB   bool   `json:"create_db"`
	DBName     string `json:"db_name"`
	DBPassword string `json:"db_password"`
}

type WebsiteService struct {
	cli      *client.Client
	dataPath string
}

func NewWebsiteService() (*WebsiteService, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	
	// 网站数据目录
	dataPath := "/Users/machangsheng/Downloads/Upanel/www"
	os.MkdirAll(dataPath, 0755)
	
	return &WebsiteService{
		cli:      cli,
		dataPath: dataPath,
	}, nil
}

func (s *WebsiteService) Close() error {
	return s.cli.Close()
}

// 创建网站
func (s *WebsiteService) CreateWebsite(req *CreateWebsiteRequest) (*Website, error) {
	// 1. 创建网站目录
	sitePath := filepath.Join(s.dataPath, req.Domain)
	htmlPath := filepath.Join(sitePath, "html")
	
	if err := os.MkdirAll(htmlPath, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %v", err)
	}
	
	// 2. 创建测试页面
	indexContent := fmt.Sprintf(`<?php
echo "<h1>Welcome to %s</h1>";
echo "<p>Powered by UPanel</p>";
echo "<p>PHP Version: " . phpversion() . "</p>";
echo "<p>Server Time: " . date('Y-m-d H:i:s') . "</p>";
?>
`, req.Domain)
	
	if err := os.WriteFile(filepath.Join(htmlPath, "index.php"), []byte(indexContent), 0644); err != nil {
		return nil, err
	}
	
	// 3. 创建 Nginx 配置文件
	nginxConfig := s.generateNginxConfig(req.Domain)
	nginxConfPath := filepath.Join(sitePath, "nginx.conf")
	if err := os.WriteFile(nginxConfPath, []byte(nginxConfig), 0644); err != nil {
		return nil, err
	}
	
	// 4. 创建 docker-compose.yml
	composeContent := s.generateDockerCompose(req, sitePath)
	composePath := filepath.Join(sitePath, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return nil, err
	}
	
	// 5. 先创建网络（如果不存在）
	networkName := "upanel_network"
	_, err := s.cli.NetworkInspect(context.Background(), networkName, types.NetworkInspectOptions{})
	if err != nil {
		_, err = s.cli.NetworkCreate(context.Background(), networkName, types.NetworkCreate{
			Driver: "bridge",
		})
		if err != nil {
			return nil, fmt.Errorf("创建网络失败: %v", err)
		}
	}
	
	// 6. 启动容器
	cmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d")
	cmd.Dir = sitePath
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("启动容器失败: %v", err)
	}
	
	// 7. 获取容器信息
	containers, err := s.cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}
	
	var containerID string
	for _, c := range containers {
		for _, name := range c.Names {
			if strings.Contains(name, req.Domain) && strings.Contains(name, "nginx") {
				containerID = c.ID[:12]
				break
			}
		}
	}
	
	return &Website{
		ID:         containerID,
		Name:       req.Domain,
		Domain:     req.Domain,
		Status:     "running",
		Port:       80,
		PHPVersion: req.PHPVersion,
		CreatedAt:  time.Now(),
		Path:       sitePath,
	}, nil
}

// 生成 Nginx 配置
func (s *WebsiteService) generateNginxConfig(domain string) string {
	return fmt.Sprintf(`server {
    listen 80;
    server_name %s;
    root /var/www/html;
    index index.php index.html;

    location / {
        try_files $uri $uri/ /index.php?$args;
    }

    location ~ \.php$ {
        fastcgi_pass php:9000;
        fastcgi_index index.php;
        fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;
        include fastcgi_params;
    }

    location ~ /\.ht {
        deny all;
    }
}
`, domain)
}

// 生成 docker-compose.yml
func (s *WebsiteService) generateDockerCompose(req *CreateWebsiteRequest, sitePath string) string {
	phpVersion := "8.2"
	if req.PHPVersion != "" {
		phpVersion = req.PHPVersion
	}
	
	compose := fmt.Sprintf(`version: '3.8'

services:
  nginx:
    image: nginx:alpine
    container_name: upanel_nginx_%s
    ports:
      - "80"
    volumes:
      - %s/html:/var/www/html
      - %s/nginx.conf:/etc/nginx/conf.d/default.conf
    networks:
      - upanel_network
    restart: unless-stopped

  php:
    image: php:%s-fpm
    container_name: upanel_php_%s
    volumes:
      - %s/html:/var/www/html
    networks:
      - upanel_network
    restart: unless-stopped
`, req.Domain, sitePath, sitePath, phpVersion, req.Domain, sitePath)
	
	// 如果需要 MySQL
	if req.CreateDB {
		dbName := req.DBName
		if dbName == "" {
			dbName = strings.ReplaceAll(req.Domain, ".", "_")
		}
		dbPassword := req.DBPassword
		if dbPassword == "" {
			dbPassword = "password123"
		}
		
		compose += fmt.Sprintf(`
  mysql:
    image: mysql:8.0
    container_name: upanel_mysql_%s
    environment:
      MYSQL_ROOT_PASSWORD: %s
      MYSQL_DATABASE: %s
    volumes:
      - %s/mysql:/var/lib/mysql
    networks:
      - upanel_network
    restart: unless-stopped
`, req.Domain, dbPassword, dbName, sitePath)
	}
	
	compose += `
networks:
  upanel_network:
    external: true
`
	
	return compose
}

// 获取网站列表
func (s *WebsiteService) ListWebsites() ([]Website, error) {
	entries, err := os.ReadDir(s.dataPath)
	if err != nil {
		return nil, err
	}
	
	var websites []Website
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		
		// 检查是否有 docker-compose.yml
		composePath := filepath.Join(s.dataPath, entry.Name(), "docker-compose.yml")
		if _, err := os.Stat(composePath); os.IsNotExist(err) {
			continue
		}
		
		// 获取容器状态
		status := "unknown"
		containerName := "upanel_nginx_" + entry.Name()
		containers, _ := s.cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
		for _, c := range containers {
			for _, name := range c.Names {
				if name == "/"+containerName {
					if c.State == "running" {
						status = "running"
					} else {
						status = "stopped"
					}
					break
				}
			}
		}
		
		websites = append(websites, Website{
			Name:       entry.Name(),
			Domain:     entry.Name(),
			Status:     status,
			PHPVersion: "8.2",
			CreatedAt:  time.Now(),
			Path:       filepath.Join(s.dataPath, entry.Name()),
		})
	}
	
	return websites, nil
}

// 启动网站
func (s *WebsiteService) StartWebsite(domain string) error {
	sitePath := filepath.Join(s.dataPath, domain)
	composePath := filepath.Join(sitePath, "docker-compose.yml")
	
	cmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d")
	cmd.Dir = sitePath
	return cmd.Run()
}

// 停止网站
func (s *WebsiteService) StopWebsite(domain string) error {
	sitePath := filepath.Join(s.dataPath, domain)
	composePath := filepath.Join(sitePath, "docker-compose.yml")
	
	cmd := exec.Command("docker", "compose", "-f", composePath, "down")
	cmd.Dir = sitePath
	return cmd.Run()
}

// 删除网站
func (s *WebsiteService) DeleteWebsite(domain string) error {
	// 先停止容器
	s.StopWebsite(domain)
	
	// 删除目录
	sitePath := filepath.Join(s.dataPath, domain)
	return os.RemoveAll(sitePath)
}

// 获取网站访问地址
func (s *WebsiteService) GetWebsiteURL(domain string) (string, error) {
	containerName := "upanel_nginx_" + domain
	containers, err := s.cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return "", err
	}
	
	for _, c := range containers {
		for _, name := range c.Names {
			if name == "/"+containerName && c.State == "running" {
				// 获取端口映射
				for _, port := range c.Ports {
					if port.PrivatePort == 80 && port.PublicPort > 0 {
						return fmt.Sprintf("http://localhost:%d", port.PublicPort), nil
					}
				}
			}
		}
	}
	
	return fmt.Sprintf("http://%s (请检查容器是否运行)", domain), nil
}
