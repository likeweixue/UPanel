package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	// 服务器配置
	Port    string `json:"port"`
	Mode    string `json:"mode"` // debug, release, test
	
	// 认证配置
	JWTSecret string `json:"jwt_secret"`
	
	// 数据库配置（可选，用于存储用户数据）
	DBPath string `json:"db_path"`
	
	// 路径配置
	StaticPath string `json:"static_path"`
	DataPath   string `json:"data_path"`
	
	// 安全配置
	SecurityEntry string `json:"security_entry"`
}

var AppConfig *Config

// 加载配置
func LoadConfig() error {
	// 加载 .env 文件（如果存在）
	_ = godotenv.Load()
	
	AppConfig = &Config{
		Port:          getEnv("PANEL_PORT", "8080"),
		Mode:          getEnv("GIN_MODE", "release"),
		JWTSecret:     getEnv("JWT_SECRET", "upanel-default-secret-key-2024"),
		DBPath:        getEnv("DB_PATH", "./data/upanel.db"),
		StaticPath:    getEnv("STATIC_PATH", "./web"),
		DataPath:      getEnv("DATA_PATH", "./data"),
		SecurityEntry: getEnv("PANEL_ENTRY", ""),
	}
	
	return nil
}

// 获取环境变量，带默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// 获取端口（整数）
func (c *Config) GetPort() int {
	port, _ := strconv.Atoi(c.Port)
	if port == 0 {
		return 8080
	}
	return port
}

// 获取安全入口路径
func (c *Config) GetSecurityEntry() string {
	entry := c.SecurityEntry
	if entry == "" {
		return ""
	}
	// 确保入口路径以 / 开头
	if !strings.HasPrefix(entry, "/") {
		entry = "/" + entry
	}
	return entry
}
