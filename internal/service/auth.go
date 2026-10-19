package service

import (
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"upanel/internal/config"
)

// DefaultPassword 首次安装且未设置密码时的兜底密码
const DefaultPassword = "upanel123"

// AuthService 基于配置文件的单用户认证服务
type AuthService struct {
	mu     sync.Mutex
	secret []byte
}

// NewAuthService 创建认证服务。
// 若配置文件中尚无密码哈希（全新安装），则写入默认密码并落盘。
func NewAuthService() *AuthService {
	cfg := config.Current()
	if cfg.PasswordHash == "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(DefaultPassword), bcrypt.DefaultCost)
		if err == nil {
			_ = cfg.SetPasswordHash(string(hash))
		}
	}
	secret := cfg.JWTSecret
	if secret == "" {
		secret = "upanel-default-secret"
	}
	return &AuthService{secret: []byte(secret)}
}

// Login 校验用户名密码，返回 JWT
func (s *AuthService) Login(username, password string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := config.Current()
	if username == "" || password == "" {
		return "", errors.New("用户名和密码不能为空")
	}
	if username != cfg.Username {
		return "", errors.New("用户名或密码错误")
	}

	hash := cfg.PasswordHash
	if hash == "" {
		return "", errors.New("面板尚未初始化密码，请执行安装脚本或使用 up 命令设置")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return "", errors.New("用户名或密码错误")
	}

	return s.issueToken(username)
}

func (s *AuthService) issueToken(username string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})
	return token.SignedString(s.secret)
}

// ValidateToken 校验 JWT 并返回用户名
func (s *AuthService) ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	})
	if err != nil {
		return "", err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if username, ok := claims["username"].(string); ok {
			return username, nil
		}
	}
	return "", errors.New("invalid token")
}

// ChangePassword 修改密码（需校验旧密码）
func (s *AuthService) ChangePassword(username, oldPassword, newPassword string) error {
	cfg := config.Current()
	if username != cfg.Username {
		return errors.New("用户不存在")
	}
	if len(newPassword) < 6 {
		return errors.New("新密码长度不能少于 6 位")
	}
	if cfg.PasswordHash != "" {
		if err := bcrypt.CompareHashAndPassword([]byte(cfg.PasswordHash), []byte(oldPassword)); err != nil {
			return errors.New("旧密码错误")
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return cfg.SetPasswordHash(string(hash))
}

// SetPassword 强制设置密码（供 CLI / 安装脚本调用，无需旧密码）
func (s *AuthService) SetPassword(newPassword string) error {
	if len(newPassword) < 6 {
		return errors.New("密码长度不能少于 6 位")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return config.Current().SetPasswordHash(string(hash))
}

// VerifyPassword 校验明文密码是否匹配
func (s *AuthService) VerifyPassword(password string) bool {
	cfg := config.Current()
	if cfg.PasswordHash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(cfg.PasswordHash), []byte(password)) == nil
}

// Username 当前管理员用户名
func (s *AuthService) Username() string {
	return config.Current().Username
}

