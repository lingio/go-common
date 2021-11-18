package common

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base32"
	"errors"
	"fmt"
	"net/http"
)

var ErrDecrypt = errors.New("encrypted_store: decryption error")

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

func (es *EncryptedStore) GetObject(file string) ([]byte, ObjectInfo, *Error) {
	data, info, err := es.backend.GetObject(es.encryptFilename(file))
	if err != nil {
		return nil, ObjectInfo{}, err
	}
	es.cipher.Decrypt(data, data)
	if filename, err := es.decryptFilename(info.Key); err != nil {
		return nil, ObjectInfo{}, NewErrorE(http.StatusInternalServerError, err)
	} else {
		info.Key = filename
	}
	return data, info, nil
}

func (es *EncryptedStore) PutObject(ctx context.Context, file string, data []byte) (ObjectInfo, *Error) {
	es.cipher.Encrypt(data, data)
	info, err := es.backend.PutObject(ctx, es.encryptFilename(file), data)
	if err != nil {
		return ObjectInfo{}, err
	}
	info.Key = file
	return info, nil
}

func (es EncryptedStore) DeleteObject(ctx context.Context, file string) *Error {
	return es.backend.DeleteObject(ctx, es.encryptFilename(file))
}

// ListObjects will list all decryptable objects.
func (es EncryptedStore) ListObjects(ctx context.Context) <-chan ObjectInfo {
	listing := es.backend.ListObjects(ctx)
	objects := make(chan ObjectInfo, 10)
	go func() {
		defer close(objects)
		for info := range listing {
			// If backing store contains objects encrypted with another key or
			// scheme, we should silently ignore them for now since the channel
			// consumer cannot do anything worthwhile with that object anyway.
			key, err := es.decryptFilename(info.Key)
			if err != nil {
				continue
			}
			info.Key = key
			objects <- info
		}
	}()
	return objects
}

func (es *EncryptedStore) encryptFilename(file string) string {
	tmp := []byte(file)
	if len(tmp) < 16 {
		panic(fmt.Errorf("%s: file length < 16 bytes", file))
	}
	es.cipher.Encrypt(tmp, tmp)
	return base32.StdEncoding.EncodeToString(tmp)
}

func (es *EncryptedStore) decryptFilename(file string) (_ string, err error) {
	// guard against Decrypt throwing a panic
	defer func() {
		if r := recover(); r != nil {
			err = ErrDecrypt
		}
	}()
	tmp, err := base32.StdEncoding.DecodeString(file)
	if err != nil {
		return "", fmt.Errorf("%s: %w", file, err)
	}
	es.cipher.Decrypt(tmp, tmp)
	return string(tmp), nil
}
