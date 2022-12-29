package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"

	"github.com/trunov/go-shortener/internal/app/util"
)

type Encryptor struct {
	key []byte
}

func NewEncryptor(key []byte) *Encryptor {
	return &Encryptor{
		key: key,
	}
}

func (e *Encryptor) Encode(userID []byte) (string, error) {
	c, err := aes.NewCipher(e.key)
	if err != nil {
		return "Cipher", err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "GCM", err
	}

	nonce, err := util.GenerateRandom(gcm.NonceSize())
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}

	out := gcm.Seal(nonce, nonce, userID, nil)

	return base64.StdEncoding.EncodeToString([]byte(out)), nil
}

func (e *Encryptor) Decode(userID string) (string, error) {
	b64Decode, err := base64.StdEncoding.DecodeString(userID)
	if err != nil {
		return "", err
	}

	c, err := aes.NewCipher(e.key)
	if err != nil {
		return "Cipher", err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return "GCM", err
	}

	nonceSize := gcm.NonceSize()
	nonce, b64UserID := b64Decode[:nonceSize], b64Decode[nonceSize:]

	decrypted, err := gcm.Open(nil, nonce, b64UserID, nil)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return "", err
	}

	return string(decrypted), nil
}
