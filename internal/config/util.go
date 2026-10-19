package config

import (
	"crypto/rand"
	"math/big"
)

const randChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// randomString 生成指定长度的随机字符串
func randomString(n int) string {
	out := make([]byte, n)
	max := big.NewInt(int64(len(randChars)))
	for i := 0; i < n; i++ {
		v, err := rand.Int(rand.Reader, max)
		if err != nil {
			out[i] = randChars[i%len(randChars)]
			continue
		}
		out[i] = randChars[v.Int64()]
	}
	return string(out)
}

// RandomPassword 生成可读的随机密码（用于数据库等初始化凭据）
func RandomPassword(n int) string {
	if n < 8 {
		n = 12
	}
	return randomString(n)
}
