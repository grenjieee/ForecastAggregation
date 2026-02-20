package kalshi

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"
)

// SignRequest 使用 RSA 私钥对 Kalshi 请求进行签名
// 消息格式: timestamp + method + path（path 不含 query）
func SignRequest(privateKeyPEM, timestamp, method, path string) (string, error) {
	path = strings.Split(path, "?")[0]
	message := timestamp + method + path

	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return "", fmt.Errorf("无法解析 PEM 私钥")
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		key2, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return "", fmt.Errorf("解析私钥失败: %w", err)
		}
		var ok bool
		key, ok = key2.(*rsa.PrivateKey)
		if !ok {
			return "", fmt.Errorf("私钥类型不是 RSA")
		}
	}

	hashed := sha256.Sum256([]byte(message))
	signature, err := rsa.SignPSS(rand.Reader, key, crypto.SHA256, hashed[:], &rsa.PSSOptions{
		SaltLength: rsa.PSSSaltLengthEqualsHash,
	})
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}
