package service

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"upanel/internal/config"
)

const (
	// MinPasswordLength 面板密码最小长度
	MinPasswordLength = 8
	// tokenTTL 登录令牌有效期
	tokenTTL = 24 * time.Hour
)

// AuthService 基于配置文件的单用户认证服务
type AuthService struct {
	mu     sync.Mutex
	secret []byte
}

// JWT 黑名单需要跨 AuthService 实例共享：
// 中间件与 handler 各自持有实例，但登出必须对两者都生效。
var (
	revokedMu     sync.Mutex
	revokedTokens = map[string]time.Time{}
)

// 首次启动密码生成只执行一次，避免多个服务实例并发写入不同密码
var initialPasswordOnce sync.Once
var initialPasswordErr error

// NewAuthService 创建认证服务。
//
// 若配置文件中尚无密码哈希（全新安装且未通过安装脚本设置密码），
// 会生成一个随机密码并写入 data/initial-password.txt（0600），
// 同时打印到日志。绝不再使用任何固定默认密码。
func NewAuthService() *AuthService {
	cfg := config.Current()

	initialPasswordOnce.Do(func() {
		needReset := cfg.PasswordHash == ""
		if !needReset {
			// 兼容旧版本遗留的弱默认密码：检测到后强制改为随机密码
			for _, legacy := range []string{"upanel123", "admin123"} {
				if bcrypt.CompareHashAndPassword([]byte(cfg.PasswordHash), []byte(legacy)) == nil {
					log.Printf("警告: 检测到旧版默认密码 %q，已强制重置为随机密码", legacy)
					needReset = true
					break
				}
			}
		}
		if !needReset {
			return
		}
		password := config.RandomPassword(16)
		hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			initialPasswordErr = fmt.Errorf("生成初始密码失败: %w", err)
			return
		}
		if err := cfg.SetPasswordHash(string(hash)); err != nil {
			initialPasswordErr = fmt.Errorf("写入初始密码失败: %w", err)
			return
		}
		initialPasswordErr = writeInitialPasswordFile(cfg, password)
	})

	if initialPasswordErr != nil {
		log.Printf("警告: %v", initialPasswordErr)
	}

	secret := cfg.JWTSecret
	if secret == "" {
		// 正常情况下 LoadConfig 已经补齐并持久化，这里仅作为兜底。
		secret = config.RandomPassword(48)
		cfg.JWTSecret = secret
		_ = cfg.Save()
	}
	return &AuthService{secret: []byte(secret)}
}

func writeInitialPasswordFile(cfg *config.Config, password string) error {
	path := cfg.InitialPasswordFile()
	content := fmt.Sprintf(`UPanel 初始管理员凭据
======================
用户名: %s
密码:   %s

登录地址: http://<服务器IP>:%s%s/
生成时间: %s

安全提示:
1. 首次登录后请在「设置 → 账号」中修改密码。
2. 修改成功后本文件不会自动删除，请手动删除: rm -f %s
3. 如已通过安装脚本设置密码，则不会生成本文件。
`, cfg.Username, password, cfg.Port, cfg.GetSecurityEntry(), time.Now().Format(time.RFC3339), path)

	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("写入初始密码文件失败: %w", err)
	}
	_ = os.Chmod(path, 0600)
	log.Printf("════════════════════════════════════════════════════════")
	log.Printf("  首次启动：已生成随机管理员密码")
	log.Printf("  用户名: %s", cfg.Username)
	log.Printf("  密码:   %s", password)
	log.Printf("  凭据文件: %s", path)
	log.Printf("  请登录后立即修改密码，并删除该文件")
	log.Printf("════════════════════════════════════════════════════════")
	return nil
}

// Login 校验用户名密码，返回 JWT
func (s *AuthService) Login(username, password string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := config.Current()
	if username == "" || password == "" {
		return "", errors.New("用户名和密码不能为空")
	}
	if subtle.ConstantTimeCompare([]byte(username), []byte(cfg.Username)) != 1 {
		time.Sleep(150 * time.Millisecond)
		return "", errors.New("用户名或密码错误")
	}

	hash := cfg.PasswordHash
	if hash == "" {
		return "", errors.New("面板尚未初始化密码，请执行安装脚本或使用 up 命令设置")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		time.Sleep(150 * time.Millisecond)
		return "", errors.New("用户名或密码错误")
	}

	return s.issueToken(username, hash)
}

func (s *AuthService) issueToken(username, passwordHash string) (string, error) {
	now := time.Now()
	sessionVersion := config.Current().SessionVersion
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":      username,
		"username": username,
		"iat":      now.Unix(),
		"nbf":      now.Unix(),
		"exp":      now.Add(tokenTTL).Unix(),
		// pv 与当前密码哈希绑定：修改密码后所有旧令牌立即失效
		"pv":  tokenFingerprint(passwordHash),
		"jti": randomID(),
		"sv":  sessionVersion,
	})
	return token.SignedString(s.secret)
}

func tokenFingerprint(passwordHash string) string {
	sum := sha256.Sum256([]byte(passwordHash))
	return hex.EncodeToString(sum[:8])
}

// ValidateToken 校验 JWT 并返回用户名
func (s *AuthService) ValidateToken(tokenString string) (string, error) {
	cfg := config.Current()

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired(), jwt.WithLeeway(30*time.Second))
	if err != nil {
		return "", err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok || !token.Valid {
		return "", errors.New("invalid token")
	}
	username, _ := claims["username"].(string)
	if username == "" || subtle.ConstantTimeCompare([]byte(username), []byte(cfg.Username)) != 1 {
		return "", errors.New("token user mismatch")
	}
	pv, _ := claims["pv"].(string)
	if pv == "" || cfg.PasswordHash == "" || pv != tokenFingerprint(cfg.PasswordHash) {
		return "", errors.New("token revoked")
	}
	if jti, _ := claims["jti"].(string); jti != "" {
		revokedMu.Lock()
		if until, ok := revokedTokens[jti]; ok {
			if time.Now().Before(until) {
				revokedMu.Unlock()
				return "", errors.New("token logged out")
			}
			delete(revokedTokens, jti)
		}
		revokedMu.Unlock()
	}
	if exp, ok := claims["exp"].(float64); !ok || time.Now().After(time.Unix(int64(exp), 0).Add(30*time.Second)) {
		return "", errors.New("token expired")
	}
	if sv, ok := claims["sv"].(float64); ok && int64(sv) != cfg.SessionVersion {
		return "", errors.New("session revoked")
	}
	return username, nil
}

// RevokeAllSessions 递增持久化的会话版本，登出后所有旧令牌（包括重启后）都失效。
func (s *AuthService) RevokeAllSessions() error {
	cfg := config.Current()
	return cfg.SaveWith(func(c *config.Config) {
		c.SessionVersion++
	})
}

// RevokeToken 将当前令牌加入黑名单（登出后立即失效）
func (s *AuthService) RevokeToken(tokenString string) error {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, errors.New("unexpected signing method")
		}
		return s.secret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return errors.New("invalid token")
	}
	jti, _ := claims["jti"].(string)
	if jti == "" {
		return nil
	}
	exp, _ := claims["exp"].(float64)
	until := time.Now().Add(tokenTTL)
	if exp > 0 {
		until = time.Unix(int64(exp), 0)
	}
	revokedMu.Lock()
	defer revokedMu.Unlock()
	// 控制黑名单大小，清理已过期条目
	if len(revokedTokens) > 2000 {
		now := time.Now()
		for id, t := range revokedTokens {
			if now.After(t) {
				delete(revokedTokens, id)
			}
		}
	}
	revokedTokens[jti] = until
	return nil
}

// ChangePassword 修改密码（需校验旧密码）
func (s *AuthService) ChangePassword(username, oldPassword, newPassword string) error {
	cfg := config.Current()
	if subtle.ConstantTimeCompare([]byte(username), []byte(cfg.Username)) != 1 {
		return errors.New("用户不存在")
	}
	if err := validatePassword(newPassword); err != nil {
		return err
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
	if err := cfg.SetPasswordHash(string(hash)); err != nil {
		return err
	}
	// 修改成功后清理首次启动密码文件，避免明文长期留存
	_ = os.Remove(cfg.InitialPasswordFile())
	return nil
}

// SetPassword 强制设置密码（供 CLI / 安装脚本调用，无需旧密码）
func (s *AuthService) SetPassword(newPassword string) error {
	if err := validatePassword(newPassword); err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := config.Current().SetPasswordHash(string(hash)); err != nil {
		return err
	}
	_ = os.Remove(config.Current().InitialPasswordFile())
	return nil
}

func validatePassword(p string) error {
	p = strings.TrimSpace(p)
	if len(p) < MinPasswordLength {
		return fmt.Errorf("密码长度不能少于 %d 位", MinPasswordLength)
	}
	if len(p) > 128 {
		return errors.New("密码过长（最多 128 位）")
	}
	var hasLetter, hasDigit bool
	for _, r := range p {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
			hasLetter = true
		case r >= '0' && r <= '9':
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return errors.New("密码必须同时包含字母和数字")
	}
	return nil
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
