package helpers

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
)

// LoadPublicKey loads a public key from file
func LoadPublicKey(filename string) (*rsa.PublicKey, error) {
	if filename == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		publicKey, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, errors.New("not an RSA public key")
		}
		return publicKey, nil
	}

	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	return publicKey, nil
}

// LoadPrivateKey loads a private key from file
func LoadPrivateKey(filename string) (*rsa.PrivateKey, error) {
	if filename == "" {
		return nil, nil
	}

	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %w", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("failed to parse PEM block containing the private key")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return privateKey, nil
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: unsupported format. PKCS#1 error: %v, PKCS#8 error: %v", err, err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("private key is not an RSA key")
	}

	return rsaKey, nil
}

// EncryptData encrypts data using hybrid encryption (AES for data, RSA for AES key)
func EncryptData(data []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	if publicKey == nil {
		return data, nil
	}

	aesKey := make([]byte, 32)
	if _, err := rand.Read(aesKey); err != nil {
		return nil, fmt.Errorf("failed to generate AES key: %w", err)
	}

	encryptedData, err := encryptAES(data, aesKey)
	if err != nil {
		return nil, fmt.Errorf("AES encryption failed: %w", err)
	}

	encryptedKey, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey, aesKey)
	if err != nil {
		return nil, fmt.Errorf("RSA encryption failed: %w", err)
	}

	result := make([]byte, len(encryptedKey)+len(encryptedData))
	copy(result[:len(encryptedKey)], encryptedKey)
	copy(result[len(encryptedKey):], encryptedData)

	return result, nil
}

// DecryptData decrypts data using hybrid decryption
func DecryptData(data []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	if privateKey == nil {
		return data, nil
	}

	keySize := privateKey.Size()
	if len(data) < keySize {
		return nil, errors.New("encrypted data too short")
	}

	encryptedKey := data[:keySize]
	encryptedData := data[keySize:]

	aesKey, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, encryptedKey)
	if err != nil {
		return nil, fmt.Errorf("RSA decryption failed: %w", err)
	}

	decryptedData, err := decryptAES(encryptedData, aesKey)
	if err != nil {
		return nil, fmt.Errorf("AES decryption failed: %w", err)
	}

	return decryptedData, nil
}

// encryptAES encrypts data using AES-GCM
func encryptAES(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// decryptAES decrypts data using AES-GCM
func decryptAES(data, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}
