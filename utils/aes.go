package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
)

// GenerateAESKey 生成指定长度的AES密钥
func GenerateAESKey(keySize int) ([]byte, error) {
	if keySize != 16 && keySize != 24 && keySize != 32 {
		return nil, errors.New("invalid key size, must be 16, 24 or 32 bytes")
	}

	key := make([]byte, keySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

// EncryptAESCBC AES-CBC模式加密
func EncryptAESCBC(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// 填充数据
	plaintext = pkcs7Pad(plaintext, aes.BlockSize)

	// 生成IV
	ciphertext := make([]byte, aes.BlockSize+len(plaintext))
	iv := ciphertext[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// 加密
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext[aes.BlockSize:], plaintext)

	return ciphertext, nil
}

// DecryptAESCBC AES-CBC模式解密
func DecryptAESCBC(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	if len(ciphertext) < aes.BlockSize {
		return nil, errors.New("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// 解密
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// 去除填充
	return pkcs7Unpad(ciphertext)
}

// EncryptAESGCM AES-GCM模式加密
func EncryptAESGCM(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// DecryptAESGCM AES-GCM模式解密
func DecryptAESGCM(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// pkcs7Pad 数据填充
func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

// pkcs7Unpad 去除填充
func pkcs7Unpad(data []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data")
	}

	padding := int(data[len(data)-1])
	if padding > len(data) {
		return nil, errors.New("invalid padding")
	}

	for i := len(data) - padding; i < len(data); i++ {
		if int(data[i]) != padding {
			return nil, errors.New("invalid padding")
		}
	}

	return data[:len(data)-padding], nil
}

// ExampleUsage AES加解密使用示例
func ExampleAesUsage() {
	// 1. 生成256位(32字节)AES密钥
	key, err := GenerateAESKey(32)
	if err != nil {
		panic(err)
	}

	// 2. 准备要加密的数据
	originalData := []byte("这是一段需要AES加密的敏感数据")

	// 3. CBC模式加密
	cbcEncrypted, err := EncryptAESCBC(originalData, key)
	if err != nil {
		panic(err)
	}

	// 4. CBC模式解密
	cbcDecrypted, err := DecryptAESCBC(cbcEncrypted, key)
	if err != nil {
		panic(err)
	}

	// 5. 验证CBC解密结果
	if string(cbcDecrypted) != string(originalData) {
		panic("CBC模式解密失败")
	}

	// 6. GCM模式加密
	gcmEncrypted, err := EncryptAESGCM(originalData, key)
	if err != nil {
		panic(err)
	}

	// 7. GCM模式解密
	gcmDecrypted, err := DecryptAESGCM(gcmEncrypted, key)
	if err != nil {
		panic(err)
	}

	// 8. 验证GCM解密结果
	if string(gcmDecrypted) != string(originalData) {
		panic("GCM模式解密失败")
	}

	// 9. 打印结果
	println("原始数据:", string(originalData))
	println("CBC加密结果:", base64.StdEncoding.EncodeToString(cbcEncrypted))
	println("GCM加密结果:", base64.StdEncoding.EncodeToString(gcmEncrypted))
}
