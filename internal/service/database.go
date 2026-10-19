package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"upanel/internal/config"
)

// Database 数据库实例
type Database struct {
	Name      string    `json:"name"`
	Type      string    `json:"type"` // mysql / postgres / redis
	Version   string    `json:"version"`
	Status    string    `json:"status"`
	Port      int       `json:"port"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Database  string    `json:"database"`
	Container string    `json:"container"`
	Path      string    `json:"path"`
	CreatedAt time.Time `json:"createdAt"`
}

// CreateDatabaseRequest 创建数据库请求
type CreateDatabaseRequest struct {
	Name     string `json:"name" binding:"required"`
	Type     string `json:"type" binding:"required"`
	Version  string `json:"version"`
	Password string `json:"password"`
	Port     int    `json:"port"`
	Database string `json:"database"`
}

// DatabaseService 数据库管理服务
type DatabaseService struct {
	rt  *Runtime
	cfg *config.Config
}

// NewDatabaseService 创建数据库服务
func NewDatabaseService() (*DatabaseService, error) {
	rt, err := NewRuntime()
	if err != nil {
		return nil, err
	}
	cfg := config.Current()
	if err := os.MkdirAll(cfg.DatabasePath, 0755); err != nil {
		rt.Close()
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}
	return &DatabaseService{rt: rt, cfg: cfg}, nil
}

func (s *DatabaseService) Close() error {
	return s.rt.Close()
}

// containerName 数据库容器名
func (s *DatabaseService) containerName(name string) string {
	return "upanel-db-" + Slug(name)
}

// defaults 返回类型相关的默认配置
func defaults(dbType string) (image, dataDir string, defPort int, user, version string) {
	switch strings.ToLower(dbType) {
	case "postgres", "postgresql":
		return "postgres:15", "/var/lib/postgresql/data", 5432, "postgres", "15"
	case "redis":
		return "redis:7-alpine", "/data", 6379, "", "7"
	default: // mysql
		return "mysql:8.0", "/var/lib/mysql", 3306, "root", "8.0"
	}
}

func (s *DatabaseService) imageFor(dbType, version string) string {
	_, _, _, _, defVer := defaults(dbType)
	if version == "" {
		version = defVer
	}
	switch strings.ToLower(dbType) {
	case "postgres", "postgresql":
		return "postgres:" + version
	case "redis":
		return "redis:" + version + "-alpine"
	default:
		return "mysql:" + version
	}
}

// usedHostPorts 收集已被容器占用的宿主机端口
func (s *DatabaseService) usedHostPorts() map[int]bool {
	used := map[int]bool{}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	list, err := s.rt.cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		return used
	}
	for _, c := range list {
		for _, p := range c.Ports {
			if p.PublicPort > 0 {
				used[int(p.PublicPort)] = true
			}
		}
	}
	return used
}

// allocPort 选定可用端口：优先使用请求端口，冲突时向上顺延
func (s *DatabaseService) allocPort(want int) (int, error) {
	used := s.usedHostPorts()
	if want > 0 && !used[want] {
		return want, nil
	}
	start := want
	if start <= 0 {
		start = 3306
	}
	for p := start; p < start+200; p++ {
		if !used[p] {
			return p, nil
		}
	}
	return 0, fmt.Errorf("在 %d ~ %d 范围内找不到可用端口", start, start+200)
}

// CreateDatabase 创建并启动数据库
func (s *DatabaseService) CreateDatabase(req *CreateDatabaseRequest) (*Database, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("数据库名称不能为空")
	}
	if strings.ContainsAny(name, "/ .") {
		return nil, fmt.Errorf("数据库名称不能包含空格、斜杠或点号")
	}

	dbType := strings.ToLower(strings.TrimSpace(req.Type))
	if dbType == "" {
		dbType = "mysql"
	}
	switch dbType {
	case "mysql", "mariadb":
		dbType = "mysql"
	case "postgres", "postgresql":
		dbType = "postgres"
	case "redis":
	default:
		return nil, fmt.Errorf("不支持的数据库类型: %s（可选 mysql / postgres / redis）", req.Type)
	}

	if _, err := os.Stat(s.metaPath(name)); err == nil {
		return nil, fmt.Errorf("数据库 %s 已存在", name)
	}

	_, _, defPort, defUser, _ := defaults(dbType)
	port, err := s.allocPort(firstPositive(req.Port, defPort))
	if err != nil {
		return nil, err
	}

	password := strings.TrimSpace(req.Password)
	if password == "" {
		password = config.RandomPassword(16)
	}
	dbName := strings.TrimSpace(req.Database)
	if dbName == "" {
		dbName = name
	}

	image := s.imageFor(dbType, req.Version)
	version := req.Version
	if version == "" {
		_, _, _, _, v := defaults(dbType)
		version = v
	}

	dbPath := filepath.Join(s.cfg.DatabasePath, name)
	dataPath := filepath.Join(dbPath, "data")
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}

	_, dataDir, _, _, _ := defaults(dbType)

	var env []string
	switch dbType {
	case "mysql":
		env = []string{
			"MYSQL_ROOT_PASSWORD=" + password,
			"MYSQL_DATABASE=" + dbName,
		}
	case "postgres":
		env = []string{
			"POSTGRES_PASSWORD=" + password,
			"POSTGRES_DB=" + dbName,
			"POSTGRES_USER=" + defUser,
		}
	case "redis":
		env = []string{
			"REDIS_PASSWORD=" + password,
		}
	}

	if err := s.rt.PullImage(image, nil); err != nil {
		return nil, err
	}

	spec := ContainerSpec{
		Name:  s.containerName(name),
		Image: image,
		Env:   env,
		Volumes: map[string]string{
			dataPath: dataDir,
		},
		Ports: map[string]string{
			fmt.Sprintf("%d/tcp", defPort): fmt.Sprintf("%d", port),
		},
	}
	if dbType == "redis" && false {
		// redis 官方镜像不支持 REDIS_PASSWORD 环境变量，保持无密码启动
		spec.Cmd = nil
	}

	if _, err := s.rt.CreateAndStart(spec); err != nil {
		return nil, err
	}

	db := &Database{
		Name:      name,
		Type:      dbType,
		Version:   version,
		Status:    "running",
		Port:      port,
		Username:  defUser,
		Password:  password,
		Database:  dbName,
		Container: s.containerName(name),
		Path:      dbPath,
		CreatedAt: time.Now(),
	}
	if err := s.saveMeta(db); err != nil {
		return nil, err
	}
	return db, nil
}

func firstPositive(v, def int) int {
	if v > 0 {
		return v
	}
	return def
}

func (s *DatabaseService) metaPath(name string) string {
	return filepath.Join(s.cfg.DatabasePath, name, "db.json")
}

func (s *DatabaseService) saveMeta(db *Database) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.metaPath(db.Name), data, 0600)
}

func (s *DatabaseService) loadMeta(name string) (*Database, error) {
	data, err := os.ReadFile(s.metaPath(name))
	if err != nil {
		return nil, err
	}
	var d Database
	if err := json.Unmarshal(data, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// ListDatabases 列出所有数据库（带实时状态）
func (s *DatabaseService) ListDatabases() ([]Database, error) {
	entries, err := os.ReadDir(s.cfg.DatabasePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []Database{}, nil
		}
		return nil, err
	}
	var list []Database
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		d, err := s.loadMeta(e.Name())
		if err != nil {
			continue
		}
		d.Status = s.statusOf(d)
		list = append(list, *d)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list, nil
}

func (s *DatabaseService) statusOf(d *Database) string {
	st := s.rt.ContainerState(d.Container)
	if st == "running" {
		return "running"
	}
	if st == "" {
		return "missing"
	}
	return "stopped"
}

// GetDatabase 获取单个数据库
func (s *DatabaseService) GetDatabase(name string) (*Database, error) {
	d, err := s.loadMeta(name)
	if err != nil {
		return nil, fmt.Errorf("数据库 %s 不存在", name)
	}
	d.Status = s.statusOf(d)
	return d, nil
}

// StartDatabase 启动
func (s *DatabaseService) StartDatabase(name string) error {
	d, err := s.loadMeta(name)
	if err != nil {
		return fmt.Errorf("数据库 %s 不存在", name)
	}
	if s.rt.ContainerState(d.Container) == "" {
		return fmt.Errorf("容器 %s 已不存在，请重新创建", d.Container)
	}
	return s.rt.Start(d.Container)
}

// StopDatabase 停止
func (s *DatabaseService) StopDatabase(name string) error {
	d, err := s.loadMeta(name)
	if err != nil {
		return fmt.Errorf("数据库 %s 不存在", name)
	}
	return s.rt.Stop(d.Container)
}

// RestartDatabase 重启
func (s *DatabaseService) RestartDatabase(name string) error {
	if err := s.StopDatabase(name); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return s.StartDatabase(name)
}

// DeleteDatabase 删除（keepData=true 时保留数据目录）
func (s *DatabaseService) DeleteDatabase(name string, keepData bool) error {
	d, err := s.loadMeta(name)
	if err != nil {
		return fmt.Errorf("数据库 %s 不存在", name)
	}
	if err := s.rt.Remove(d.Container); err != nil {
		return fmt.Errorf("删除容器失败: %w", err)
	}
	if keepData {
		return os.Remove(s.metaPath(name))
	}
	return os.RemoveAll(filepath.Join(s.cfg.DatabasePath, name))
}

// ConnectionInfo 连接信息
func (s *DatabaseService) ConnectionInfo(name string) (map[string]string, error) {
	d, err := s.loadMeta(name)
	if err != nil {
		return nil, fmt.Errorf("数据库 %s 不存在", name)
	}
	info := map[string]string{
		"host":     "127.0.0.1",
		"port":     fmt.Sprintf("%d", d.Port),
		"username": d.Username,
		"password": d.Password,
		"database": d.Database,
		"type":     d.Type,
		"status":   s.statusOf(d),
	}
	if d.Type == "mysql" {
		info["dsn"] = fmt.Sprintf("mysql://%s:%s@127.0.0.1:%d/%s", d.Username, d.Password, d.Port, d.Database)
	} else if d.Type == "postgres" {
		info["dsn"] = fmt.Sprintf("postgres://%s:%s@127.0.0.1:%d/%s", d.Username, d.Password, d.Port, d.Database)
	} else {
		info["dsn"] = fmt.Sprintf("redis://127.0.0.1:%d", d.Port)
	}
	return info, nil
}

// Logs 获取数据库容器日志
func (s *DatabaseService) Logs(name string, tail int) (string, error) {
	d, err := s.loadMeta(name)
	if err != nil {
		return "", fmt.Errorf("数据库 %s 不存在", name)
	}
	if tail <= 0 {
		tail = 200
	}
	ctx := context.Background()
	opts := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%d", tail),
	}
	info, err := s.rt.FindContainer(d.Container)
	if err != nil {
		return "", err
	}
	rc, err := s.rt.cli.ContainerLogs(ctx, info.ID, opts)
	if err != nil {
		return "", err
	}
	defer rc.Close()
	buf := make([]byte, 0)
	tmp := make([]byte, 4096)
	for {
		n, err := rc.Read(tmp)
		buf = append(buf, tmp[:n]...)
		if err != nil {
			break
		}
	}
	return string(buf), nil
}

var _ = strings.TrimSpace
