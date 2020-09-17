package airgapped

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"golang.org/x/crypto/scrypt"
	"io"
)

type EncryptionError string

func (e EncryptionError) Error() string {
	return fmt.Sprintf("failed to encrypt data: %v", e)
}

type DecryptionError string

func (e DecryptionError) Error() string {
	return fmt.Sprintf("failed to decrypt data: %v", e)
}

func encrypt(key, data []byte) ([]byte, error) {
	//TODO: salt
	derivedKey, err := scrypt.Key(key, nil, 32768, 8, 1, 32)
	if err != nil {
		return nil, EncryptionError(err.Error())
	}

	c, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, EncryptionError(err.Error())
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, EncryptionError(err.Error())
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, EncryptionError(err.Error())
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func decrypt(key, data []byte) ([]byte, error) {
	//TODO: salt
	derivedKey, err := scrypt.Key(key, nil, 32768, 8, 1, 32)
	if err != nil {
		return nil, DecryptionError(err.Error())
	}

	c, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, DecryptionError(err.Error())
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, DecryptionError(err.Error())
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, DecryptionError("invalid data length")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	decryptedData, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, DecryptionError(err.Error())
	}

	return decryptedData, nil
}
