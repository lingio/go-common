package common

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base32"
	"errors"
	"fmt"
)

// EncryptedStore
type EncryptedStore struct {
	backend LingioStore
	// AES block cipher is safe for concurrent use
	cipher cipher.Block
}

// NewEncryptedStore
func NewEncryptedStore(backend LingioStore, cipherKey string) (*EncryptedStore, error) {
	if cipherKey == "" {
		return nil, errors.New("cipherKey must not be empty")
	}
	if len(cipherKey) != 32 {
		return nil, errors.New("cipherKey must be 32 chars")
	}

	cipher, err := aes.NewCipher([]byte(cipherKey))
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}

	return &EncryptedStore{
		backend: backend,
		cipher:  cipher,
	}, nil
}

func (es *EncryptedStore) GetObject(file string) ([]byte, ObjectInfo, error) {
	data, info, err := es.backend.GetObject(es.encryptFilename(file))
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	es.cipher.Decrypt(data, data)
	info.Key = file
	return data, info, nil
}

func (es *EncryptedStore) PutObject(file string, data []byte) (ObjectInfo, error) {
	es.cipher.Encrypt(data, data)
	info, err := es.backend.PutObject(es.encryptFilename(file), data)
	if err != nil {
		return ObjectInfo{}, err
	}
	info.Key = file
	return info, nil
}

func (es EncryptedStore) DeleteObject(file string) error {
	return es.backend.DeleteObject(es.encryptFilename(file))
}

func (es EncryptedStore) ListObjects() <-chan ObjectInfo {
	listing := es.backend.ListObjects()
	objects := make(chan ObjectInfo, 10)
	go func() {
		defer close(objects)
		for info := range listing {
			info.Key = es.decryptFilename(info.Key)
			objects <- info
		}
	}()
	return objects
}

func (es *EncryptedStore) encryptFilename(file string) string {
	tmp := []byte(file)
	es.cipher.Encrypt(tmp, tmp)
	return base32.StdEncoding.EncodeToString(tmp)
}

func (es *EncryptedStore) decryptFilename(file string) string {
	tmp, err := base32.StdEncoding.DecodeString(file)
	if err != nil {
		panic(err)
	}
	es.cipher.Decrypt(tmp, tmp)
	return string(tmp)
}