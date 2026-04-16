package service

import (
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	users      map[string]string
	jwtSecret  []byte
}

// 替换下面这行为你刚才生成的哈希值
const adminPasswordHash = "$2a$10$KRDc8YeI5cqbCJomWW44Z.i3ZDNp70SvfVPM3f3Cwe.EGvj1ga4fa"

func NewAuthService() *AuthService {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "upanel-default-secret-key-2024"
	}
	
	return &AuthService{
		users: map[string]string{
			"admin": adminPasswordHash,
		},
		jwtSecret: []byte(secret),
	}
}

func (s *AuthService) Login(username, password string) (string, error) {
	hashedPassword, exists := s.users[username]
	if !exists {
		return "", errors.New("用户不存在")
	}
	
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		return "", errors.New("密码错误")
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
		"iat":      time.Now().Unix(),
	})
	
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}
	
	return tokenString, nil
}

func (s *AuthService) ValidateToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	
	if err != nil {
		return "", err
	}
	
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		username, ok := claims["username"].(string)
		if !ok {
			return "", errors.New("invalid token claims")
		}
		return username, nil
	}
	
	return "", errors.New("invalid token")
}

func (s *AuthService) ChangePassword(username, oldPassword, newPassword string) error {
	hashedPassword, exists := s.users[username]
	if !exists {
		return errors.New("用户不存在")
	}
	
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(oldPassword))
	if err != nil {
		return errors.New("旧密码错误")
	}
	
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	s.users[username] = string(newHashedPassword)
	return nil
}
