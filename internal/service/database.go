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

type Database struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"` // mysql, postgres
	Version    string    `json:"version"`
	Status     string    `json:"status"`
	Port       int       `json:"port"`
	Username   string    `json:"username"`
	Password   string    `json:"password"`
	CreatedAt  time.Time `json:"created_at"`
	Path       string    `json:"path"`
}

type CreateDatabaseRequest struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"` // mysql, postgres
	Version  string `json:"version"`
	Password string `json:"password"`
	Port     int    `json:"port"`
}

type DatabaseService struct {
	cli      *client.Client
	dataPath string
}

func NewDatabaseService() (*DatabaseService, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	
	// 数据库数据目录
	dataPath := "/Users/machangsheng/Downloads/Upanel/databases"
	os.MkdirAll(dataPath, 0755)
	
	return &DatabaseService{
		cli:      cli,
		dataPath: dataPath,
	}, nil
}

func (s *DatabaseService) Close() error {
	return s.cli.Close()
}

// 创建数据库
func (s *DatabaseService) CreateDatabase(req *CreateDatabaseRequest) (*Database, error) {
	// 设置默认值
	if req.Version == "" {
		if req.Type == "mysql" {
			req.Version = "8.0"
		} else {
			req.Version = "15"
		}
	}
	
	if req.Password == "" {
		req.Password = generateRandomPassword()
	}
	
	if req.Port == 0 {
		if req.Type == "mysql" {
			req.Port = 3306
		} else {
			req.Port = 5432
		}
	}
	
	// 创建数据目录
	dbPath := filepath.Join(s.dataPath, req.Name)
	if err := os.MkdirAll(dbPath, 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %v", err)
	}
	
	// 创建 docker-compose.yml
	composeContent := s.generateDockerCompose(req, dbPath)
	composePath := filepath.Join(dbPath, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		return nil, err
	}
	
	// 创建网络（如果不存在）
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
	
	// 启动容器
	cmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d")
	cmd.Dir = dbPath
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("启动容器失败: %v", err)
	}
	
	// 获取容器信息
	containers, err := s.cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}
	
	var containerID string
	for _, c := range containers {
		for _, name := range c.Names {
			if strings.Contains(name, req.Name) {
				containerID = c.ID[:12]
				break
			}
		}
	}
	
	// 获取实际端口映射
	actualPort := req.Port
	for _, c := range containers {
		for _, name := range c.Names {
			if strings.Contains(name, req.Name) && len(c.Ports) > 0 {
				actualPort = int(c.Ports[0].PublicPort)
				break
			}
		}
	}
	
	return &Database{
		ID:        containerID,
		Name:      req.Name,
		Type:      req.Type,
		Version:   req.Version,
		Status:    "running",
		Port:      actualPort,
		Username:  "root",
		Password:  req.Password,
		CreatedAt: time.Now(),
		Path:      dbPath,
	}, nil
}

// 生成 docker-compose.yml
func (s *DatabaseService) generateDockerCompose(req *CreateDatabaseRequest, dbPath string) string {
	var image string
	var envVars string
	var defaultPort int
	
	switch req.Type {
	case "mysql":
		image = fmt.Sprintf("mysql:%s", req.Version)
		envVars = fmt.Sprintf(`
      MYSQL_ROOT_PASSWORD: %s
      MYSQL_DATABASE: app_db`, req.Password)
		defaultPort = 3306
	case "postgres":
		image = fmt.Sprintf("postgres:%s", req.Version)
		envVars = fmt.Sprintf(`
      POSTGRES_PASSWORD: %s
      POSTGRES_DB: app_db
      POSTGRES_USER: root`, req.Password)
		defaultPort = 5432
	default:
		image = "mysql:8.0"
		envVars = fmt.Sprintf(`MYSQL_ROOT_PASSWORD: %s`, req.Password)
		defaultPort = 3306
	}
	
	return fmt.Sprintf(`version: '3.8'

services:
  database:
    image: %s
    container_name: upanel_db_%s
    ports:
      - "%d"
    environment:
      %s
    volumes:
      - ./data:/var/lib/mysql
    networks:
      - upanel_network
    restart: unless-stopped

networks:
  upanel_network:
    external: true
`, image, req.Name, defaultPort, envVars)
}

// 获取数据库列表
func (s *DatabaseService) ListDatabases() ([]Database, error) {
	entries, err := os.ReadDir(s.dataPath)
	if err != nil {
		return nil, err
	}
	
	var databases []Database
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
		containerName := "upanel_db_" + entry.Name()
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
		
		databases = append(databases, Database{
			Name:      entry.Name(),
			Type:      "mysql", // 可以从配置文件读取
			Status:    status,
			CreatedAt: time.Now(),
			Path:      filepath.Join(s.dataPath, entry.Name()),
		})
	}
	
	return databases, nil
}

// 启动数据库
func (s *DatabaseService) StartDatabase(name string) error {
	dbPath := filepath.Join(s.dataPath, name)
	composePath := filepath.Join(dbPath, "docker-compose.yml")
	
	cmd := exec.Command("docker", "compose", "-f", composePath, "up", "-d")
	cmd.Dir = dbPath
	return cmd.Run()
}

// 停止数据库
func (s *DatabaseService) StopDatabase(name string) error {
	dbPath := filepath.Join(s.dataPath, name)
	composePath := filepath.Join(dbPath, "docker-compose.yml")
	
	cmd := exec.Command("docker", "compose", "-f", composePath, "down")
	cmd.Dir = dbPath
	return cmd.Run()
}

// 重启数据库
func (s *DatabaseService) RestartDatabase(name string) error {
	s.StopDatabase(name)
	time.Sleep(2 * time.Second)
	return s.StartDatabase(name)
}

// 删除数据库
func (s *DatabaseService) DeleteDatabase(name string) error {
	// 先停止容器
	s.StopDatabase(name)
	
	// 删除目录
	dbPath := filepath.Join(s.dataPath, name)
	return os.RemoveAll(dbPath)
}

// 获取连接信息
func (s *DatabaseService) GetConnectionInfo(name string) (map[string]string, error) {
	containerName := "upanel_db_" + name
	containers, err := s.cli.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return nil, err
	}
	
	var port int
	for _, c := range containers {
		for _, n := range c.Names {
			if n == "/"+containerName && c.State == "running" {
				if len(c.Ports) > 0 {
					port = int(c.Ports[0].PublicPort)
				}
				break
			}
		}
	}
	
	info := map[string]string{
		"host":     "localhost",
		"port":     fmt.Sprintf("%d", port),
		"username": "root",
		"password": "password123", // 实际应该从配置文件读取
		"database": "app_db",
	}
	
	return info, nil
}

// 生成随机密码
func generateRandomPassword() string {
	return fmt.Sprintf("Upanel%d%d%d", time.Now().UnixNano()%10000, time.Now().UnixNano()%10000, time.Now().UnixNano()%10000)
}
