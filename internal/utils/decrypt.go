package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
)

// Decrypt function to get the original private key
func Decrypt(encryptedData, iv, secretKey string) (string, error) {
	// Convert inputs from hex to byte slices
	encryptedBytes, err := hex.DecodeString(encryptedData)
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	ivBytes, err := hex.DecodeString(iv)
	if err != nil {
		return "", fmt.Errorf("failed to decode IV: %w", err)
	}

	// Convert the secret key to a byte slice
	key := []byte(secretKey)
	if len(key) != 32 {
		return "", errors.New("secret key must be 32 bytes for AES-256")
	}

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Decrypt the data using CBC mode
	mode := cipher.NewCBCDecrypter(block, ivBytes)
	decrypted := make([]byte, len(encryptedBytes))
	mode.CryptBlocks(decrypted, encryptedBytes)

	// Remove padding from decrypted data
	decrypted = removePKCS7Padding(decrypted)

	return string(decrypted), nil
}

// Helper function to remove PKCS7 padding
func removePKCS7Padding(data []byte) []byte {
	paddingLength := int(data[len(data)-1])
	return data[:len(data)-paddingLength]
}
