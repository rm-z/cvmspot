package utils

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
)

// GenerateKey 生成RSA密钥对
func GenerateKey(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, &privateKey.PublicKey, nil
}

// Encrypt RSA加密
func Encrypt(plainText []byte, pubKey *rsa.PublicKey) ([]byte, error) {
	return rsa.EncryptPKCS1v15(rand.Reader, pubKey, plainText)
}

// Decrypt RSA解密
func Decrypt(cipherText []byte, privKey *rsa.PrivateKey) ([]byte, error) {
	return rsa.DecryptPKCS1v15(rand.Reader, privKey, cipherText)
}

// ExportPublicKey 导出公钥为PEM格式
func ExportPublicKey(pubKey *rsa.PublicKey) ([]byte, error) {
	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return nil, err
	}
	pubKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubKeyBytes,
	})
	return pubKeyPem, nil
}

// ExportPrivateKey 导出私钥为PEM格式
func ExportPrivateKey(privKey *rsa.PrivateKey) []byte {
	privKeyBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privKeyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})
	return privKeyPem
}

// ParsePublicKey 从PEM格式解析公钥
func ParsePublicKey(pubKeyPem []byte) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(pubKeyPem)
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return pub.(*rsa.PublicKey), nil
}

// ParsePrivateKey 从PEM格式解析私钥
func ParsePrivateKey(privKeyPem []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(privKeyPem)
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the private key")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

// ExampleUsage 展示RSA加解密使用示例
func ExampleUsage() {
	// 1. 生成2048位RSA密钥对
	privateKey, publicKey, err := GenerateKey(2048)
	if err != nil {
		panic(err)
	}

	// 2. 准备要加密的数据
	originalData := []byte("这是一段需要加密的敏感数据")

	// 3. 使用公钥加密
	encryptedData, err := Encrypt(originalData, publicKey)
	if err != nil {
		panic(err)
	}

	// 4. 使用私钥解密
	decryptedData, err := Decrypt(encryptedData, privateKey)
	if err != nil {
		panic(err)
	}

	// 5. 验证解密结果
	if string(decryptedData) != string(originalData) {
		panic("解密数据与原始数据不匹配")
	}

	// 6. 导出公钥和私钥
	pubPEM, err := ExportPublicKey(publicKey)
	if err != nil {
		panic(err)
	}
	privPEM := ExportPrivateKey(privateKey)

	// 7. 从PEM重新导入密钥
	importedPubKey, err := ParsePublicKey(pubPEM)
	if err != nil {
		panic(err)
	}
	importedPrivKey, err := ParsePrivateKey(privPEM)
	if err != nil {
		panic(err)
	}

	// 8. 使用导入的密钥再次验证
	reEncrypted, err := Encrypt(originalData, importedPubKey)
	if err != nil {
		panic(err)
	}
	reDecrypted, err := Decrypt(reEncrypted, importedPrivKey)
	if err != nil {
		panic(err)
	}

	if string(reDecrypted) != string(originalData) {
		panic("导入密钥后解密失败")
	}
}
