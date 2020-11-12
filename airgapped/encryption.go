package airgapped

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"math"

	"golang.org/x/crypto/scrypt"
)

var N = int(math.Pow(2, 16))

func encrypt(key, salt, data []byte) ([]byte, error) {
	derivedKey, err := scrypt.Key(key, salt, N, 8, 1, 32)
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func decrypt(key, salt, data []byte) ([]byte, error) {
	derivedKey, err := scrypt.Key(key, salt, N, 8, 1, 32)
	if err != nil {
		return nil, err
	}

	c, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("invalid data length")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	decryptedData, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return decryptedData, nil
}
