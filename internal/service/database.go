package service

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

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
	Password  string    `json:"password,omitempty"`
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
	cfg := config.Current()
	if err := os.MkdirAll(cfg.DatabasePath, 0755); err != nil {
		return nil, fmt.Errorf("创建数据库目录失败: %w", err)
	}
	s := &DatabaseService{cfg: cfg}
	rt, err := NewRuntime()
	if err != nil {
		return s, err
	}
	s.rt = rt
	return s, nil
}

func (s *DatabaseService) dockerReady() error {
	if s.rt == nil {
		return fmt.Errorf("Docker 未连接，无法执行该操作")
	}
	return nil
}

func (s *DatabaseService) Close() error {
	if s.rt == nil {
		return nil
	}
	return s.rt.Close()
}

var dbNameRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,31}$`)
var dbIdentifierRe = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,63}$`)
var dbVersionRe = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{0,31}$`)

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
		if strings.HasSuffix(version, "-alpine") {
			return "redis:" + version
		}
		return "redis:" + version + "-alpine"
	default:
		return "mysql:" + version
	}
}

// allocPort 选定可用端口：优先使用请求端口；被占用时向上顺延。
// 同时检查宿主机监听端口和已有容器映射，避免容器启动后端口冲突。
func (s *DatabaseService) allocPort(want int) (int, error) {
	if want > 0 {
		if want > 65535 {
			return 0, fmt.Errorf("端口必须是 1-65535 之间的数字")
		}
		if !s.rt.HostPortOccupied(want) {
			return want, nil
		}
	}
	start := want
	if start <= 0 {
		start = 3306
	}
	for p := start; p < start+200 && p <= 65535; p++ {
		if !s.rt.HostPortOccupied(p) {
			return p, nil
		}
	}
	return 0, fmt.Errorf("在 %d ~ %d 范围内找不到可用端口", start, start+200)
}

// CreateDatabase 创建并启动数据库
func (s *DatabaseService) CreateDatabase(req *CreateDatabaseRequest) (*Database, error) {
	if err := s.dockerReady(); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("数据库名称不能为空")
	}
	if !dbNameRe.MatchString(name) {
		return nil, fmt.Errorf("数据库名称只能包含字母、数字、- 和 _，且必须以字母开头（最多 32 位）")
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

	version := strings.TrimSpace(req.Version)
	if version != "" && !dbVersionRe.MatchString(version) {
		return nil, fmt.Errorf("版本号包含非法字符")
	}

	if _, err := os.Stat(s.metaPath(name)); err == nil {
		return nil, fmt.Errorf("数据库 %s 已存在", name)
	}

	_, dataDir, defPort, defUser, defVer := defaults(dbType)
	if version == "" {
		version = defVer
	}
	port, err := s.allocPort(firstPositive(req.Port, defPort))
	if err != nil {
		return nil, err
	}

	password := strings.TrimSpace(req.Password)
	if password == "" {
		password = config.RandomPassword(20)
	}
	if len(password) < 8 || len(password) > 128 {
		return nil, fmt.Errorf("数据库密码长度必须在 8-128 位之间")
	}
	if strings.ContainsAny(password, "\r\n\x00") {
		return nil, fmt.Errorf("数据库密码包含非法字符")
	}

	dbName := strings.TrimSpace(req.Database)
	if dbName == "" {
		// 实例名允许连字符，但 SQL 标识符统一用下划线
		dbName = strings.ReplaceAll(name, "-", "_")
	}
	if !dbIdentifierRe.MatchString(dbName) {
		return nil, fmt.Errorf("数据库名只能包含字母、数字和下划线，且必须以字母开头")
	}

	image := s.imageFor(dbType, version)
	dbPath := filepath.Join(s.cfg.DatabasePath, name)
	dbPathExisted := dirExists(dbPath)
	dataPath := filepath.Join(dbPath, "data")
	if err := os.MkdirAll(dataPath, 0755); err != nil {
		return nil, fmt.Errorf("创建数据目录失败: %w", err)
	}
	switch dbType {
	case "mysql":
		prepareMySQLDir(dataPath)
	case "postgres":
		preparePostgresDir(dataPath)
	case "redis":
		prepareRedisDir(dataPath)
	}

	var env []string
	var cmd []string
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
		// 官方 redis 镜像不识别 REDIS_PASSWORD，必须显式通过命令行启用认证。
		cmd = []string{"redis-server", "--requirepass", password, "--appendonly", "yes"}
	}

	if err := s.rt.PullImage(image, nil); err != nil {
		if !dbPathExisted {
			_ = os.RemoveAll(dbPath)
		}
		return nil, err
	}

	spec := ContainerSpec{
		Name:  s.containerName(name),
		Image: image,
		Env:   env,
		Cmd:   cmd,
		Volumes: map[string]string{
			dataPath: dataDir,
		},
		Ports: map[string]string{
			fmt.Sprintf("%d/tcp", defPort): fmt.Sprintf("%d", port),
		},
		// 数据库默认只监听 127.0.0.1，避免 MySQL/Redis 直接暴露到公网。
		// 确实需要远程访问时，请通过 SSH 隧道或自行调整防火墙/端口映射。
		BindLocalhost: true,
		Labels:        map[string]string{"managed-by": "upanel", "upanel-db": name},
	}

	if _, err := s.rt.CreateAndStart(spec); err != nil {
		if !dbPathExisted {
			_ = os.RemoveAll(dbPath)
		}
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
		_ = s.rt.Remove(spec.Name)
		if !dbPathExisted {
			_ = os.RemoveAll(dbPath)
		}
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
	if !dbNameRe.MatchString(name) {
		return ""
	}
	return filepath.Join(s.cfg.DatabasePath, name, "db.json")
}

func (s *DatabaseService) saveMeta(db *Database) error {
	data, err := json.MarshalIndent(db, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(s.cfg.DatabasePath, db.Name), 0755); err != nil {
		return err
	}
	return writeFileAtomic(s.metaPath(db.Name), data, 0600)
}

func (s *DatabaseService) loadMeta(name string) (*Database, error) {
	p := s.metaPath(name)
	if p == "" {
		return nil, fmt.Errorf("数据库名称不合法")
	}
	data, err := os.ReadFile(p)
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
	list := make([]Database, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		d, err := s.loadMeta(e.Name())
		if err != nil {
			continue
		}
		d.Status = s.statusOf(d)
		d.Password = "" // 列表接口不返回密码，凭据通过连接信息接口单独获取
		list = append(list, *d)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return list, nil
}

func (s *DatabaseService) statusOf(d *Database) string {
	if s.rt == nil {
		return "docker_unavailable"
	}
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
	d.Password = ""
	return d, nil
}

// StartDatabase 启动
func (s *DatabaseService) StartDatabase(name string) error {
	if err := s.dockerReady(); err != nil {
		return err
	}
	d, err := s.loadMeta(name)
	if err != nil {
		return fmt.Errorf("数据库 %s 不存在", name)
	}
	if s.rt.ContainerState(d.Container) == "" {
		return fmt.Errorf("容器 %s 已不存在，请删除后重新创建", d.Container)
	}
	return s.rt.Start(d.Container)
}

// StopDatabase 停止
func (s *DatabaseService) StopDatabase(name string) error {
	if err := s.dockerReady(); err != nil {
		return err
	}
	d, err := s.loadMeta(name)
	if err != nil {
		return fmt.Errorf("数据库 %s 不存在", name)
	}
	if s.rt.ContainerState(d.Container) == "" {
		return fmt.Errorf("容器 %s 已不存在", d.Container)
	}
	return s.rt.Stop(d.Container)
}

// RestartDatabase 重启
func (s *DatabaseService) RestartDatabase(name string) error {
	if err := s.StopDatabase(name); err != nil {
		return err
	}
	time.Sleep(time.Second)
	return s.StartDatabase(name)
}

// DeleteDatabase 删除（keepData=true 时保留数据目录）
func (s *DatabaseService) DeleteDatabase(name string, keepData bool) error {
	if err := s.dockerReady(); err != nil {
		return err
	}
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
	pass := url.QueryEscape(d.Password)
	info := map[string]string{
		"host":     "127.0.0.1",
		"port":     fmt.Sprintf("%d", d.Port),
		"username": d.Username,
		"password": d.Password,
		"database": d.Database,
		"type":     d.Type,
		"status":   s.statusOf(d),
	}
	switch d.Type {
	case "mysql":
		info["dsn"] = fmt.Sprintf("mysql://%s:%s@127.0.0.1:%d/%s", d.Username, pass, d.Port, d.Database)
	case "postgres":
		info["dsn"] = fmt.Sprintf("postgres://%s:%s@127.0.0.1:%d/%s?sslmode=disable", d.Username, pass, d.Port, d.Database)
	default:
		info["dsn"] = fmt.Sprintf("redis://:%s@127.0.0.1:%d/0", pass, d.Port)
	}
	info["hint"] = "数据库仅监听 127.0.0.1，远程访问请使用 SSH 隧道：ssh -L " +
		fmt.Sprintf("%d:127.0.0.1:%d", d.Port, d.Port) + " root@服务器IP"
	return info, nil
}

// Logs 获取数据库容器日志
func (s *DatabaseService) Logs(name string, tail int) (string, error) {
	if err := s.dockerReady(); err != nil {
		return "", err
	}
	d, err := s.loadMeta(name)
	if err != nil {
		return "", fmt.Errorf("数据库 %s 不存在", name)
	}
	return s.rt.Logs(d.Container, tail)
}

// ValidateDatabaseName 供 handler 校验路径中的数据库名
func ValidateDatabaseName(name string) error {
	if !dbNameRe.MatchString(name) {
		return fmt.Errorf("数据库名称只能包含字母、数字、- 和 _，且必须以字母开头")
	}
	return nil
}
