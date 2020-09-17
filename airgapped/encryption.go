package airgapped

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"golang.org/x/crypto/scrypt"
	"io"
)

func encrypt(key, data []byte) ([]byte, error) {
	//TODO: salt
	derivedKey, err := scrypt.Key(key, nil, 32768, 8, 1, 32)
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}

	cipherData := make([]byte, aes.BlockSize+len(data))
	iv := cipherData[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(cipherData[aes.BlockSize:], data)
	return cipherData, nil
}

func decrypt(key, data []byte) ([]byte, error) {
	//TODO: salt
	derivedKey, err := scrypt.Key(key, nil, 32768, 8, 1, 32)
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, err
	}

	if len(data) < aes.BlockSize {
		return nil, fmt.Errorf("ciphertext block size is too short")
	}

	iv := data[:aes.BlockSize]
	data = data[aes.BlockSize:]

	stream := cipher.NewCFBDecrypter(block, iv)
	stream.XORKeyStream(data, data)

	return data, nil
}
