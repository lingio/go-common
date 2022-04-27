package common

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"net/http"
)

var ErrDecrypt = errors.New("encrypted store: decryption error")

// EncryptedStore
type EncryptedStore struct {
	backend LingioStore
	crypto  cryptoModule
}

// cryptoModule is a simple wrapper for AEAD crypto.
type cryptoModule interface {
	encryptFilename(plaintext string) string
	decryptFilename(ciphertext string) (string, error)

	encryptData(nonce, data []byte) []byte
	decryptData(ciphertext []byte) []byte
}

// NewEncryptedStore initializes a lingio store with secure v2 crypto.
func NewEncryptedStore(backend LingioStore, cipherKey string) (*EncryptedStore, error) {
	if len(cipherKey) != 32 {
		return nil, errors.New("encrypted store: cipherKey must be 32 chars")
	}

	cm, err := newV2Crypto([]byte(cipherKey))
	if err != nil {
		return nil, fmt.Errorf("encrypted store: v2 crypto: %w", err)
	}

	return &EncryptedStore{
		backend: backend,
		crypto:  cm,
	}, nil
}

// NewInsecureEncryptedStore initializes a lingio store with insecure v1 crypto.
func NewInsecureEncryptedStore(backend LingioStore, cipherKey string) (*EncryptedStore, error) {
	if len(cipherKey) != 32 {
		return nil, errors.New("encrypted store: cipherKey must be 32 chars")
	}

	cm, err := newV1Crypto([]byte(cipherKey))
	if err != nil {
		return nil, fmt.Errorf("encrypted store: v1 crypto: %w", err)
	}

	return &EncryptedStore{
		backend: backend,
		crypto:  cm,
	}, nil
}

func (es *EncryptedStore) GetObject(file string) ([]byte, ObjectInfo, *Error) {
	// We don't know which crypto gen to use to map plaintext filename to ciphertext filename
	// so we can only do trail and error.
	data, info, lerr := es.backend.GetObject(es.crypto.encryptFilename(file))
	if lerr != nil {
		return nil, ObjectInfo{}, lerr
	}

	var err error
	plaintext := es.crypto.decryptData(data)
	info.Key, err = es.crypto.decryptFilename(info.Key)
	if err != nil {
		return nil, ObjectInfo{}, NewErrorE(http.StatusInternalServerError, err)
	}

	return plaintext, info, nil
}

func (es *EncryptedStore) PutObject(ctx context.Context, file string, data []byte) (ObjectInfo, *Error) {
	encdata := es.crypto.encryptData(nil, data) // generate new nonce for every write
	encfile := es.crypto.encryptFilename(file)

	info, err := es.backend.PutObject(ctx, encfile, encdata)
	if err != nil {
		return ObjectInfo{}, err
	}
	info.Key = file
	return info, nil
}

func (es EncryptedStore) DeleteObject(ctx context.Context, file string) *Error {
	return es.backend.DeleteObject(ctx, es.crypto.encryptFilename(file))
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
			key, err := es.crypto.decryptFilename(info.Key)
			if err != nil {
				continue
			}
			info.Key = key
			objects <- info
		}
	}()
	return objects
}

func (es EncryptedStore) StoreName() string {
	return es.backend.StoreName()
}

// v1Crypto implements partial object+filename encryption.
type v1Crypto struct {
	cipher cipher.Block
}

func newV1Crypto(key []byte) (cryptoModule, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}
	return v1Crypto{block}, nil
}

func (c v1Crypto) encryptFilename(plaintext string) string {
	ciphertext := []byte(plaintext)
	c.cipher.Encrypt(ciphertext, ciphertext)
	return base32.StdEncoding.EncodeToString(ciphertext)
}

func (c v1Crypto) decryptFilename(ciphertext string) (string, error) {
	decodedCiphertext, err := base32.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base32 decode: %q: %w", ciphertext, err)
	}
	return string(c.decryptData(decodedCiphertext)), nil
}

func (c v1Crypto) encryptData(nonce, data []byte) []byte {
	// nonce is not used
	c.cipher.Encrypt(data, data)
	return data
}

func (c v1Crypto) decryptData(data []byte) []byte {
	c.cipher.Decrypt(data, data)
	return data
}

// v2Crypto implements full object+filename encryption.
type v2Crypto struct {
	aesgcm cipher.AEAD
}

func newV2Crypto(key []byte) (cryptoModule, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes gcm: %w", err)
	}

	return v2Crypto{aesgcm}, nil
}

func (c v2Crypto) encryptFilename(plaintext string) string {
	key := []byte(plaintext)

	// deterministic nonce so we can find encoded ciphertext in object store in GetObject
	nonce := fnv.New128a().Sum(key) // fnv128a has decent avalance properties and is fast
	nonce = nonce[:c.aesgcm.NonceSize()]

	ciphertext := c.encryptData(nonce, key)
	return base32.StdEncoding.EncodeToString(ciphertext)
}

func (c v2Crypto) decryptFilename(ciphertext string) (string, error) {
	decodedCiphertext, err := base32.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("base32 decode: %q: %w", ciphertext, err)
	}
	return string(c.decryptData(decodedCiphertext)), nil
}

func (c v2Crypto) encryptData(nonce, data []byte) []byte {
	if nonce == nil {
		nonce = make([]byte, c.aesgcm.NonceSize())
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			panic(fmt.Errorf("encrypted store: could not generate nonce: %w", err))
		}
	}

	// ciphertext reuses data slice
	ciphertext := c.aesgcm.Seal(data[:0], nonce, data, nil)

	// nonce||ciphertext
	var blob []byte
	blob = append(blob, nonce...)
	blob = append(blob, ciphertext...)
	return blob
}

func (c v2Crypto) decryptData(data []byte) []byte {
	nonce := data[:c.aesgcm.NonceSize()]
	ciphertext := data[c.aesgcm.NonceSize():]

	plaintext, err := c.aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(fmt.Errorf("encrypted store: decrypt: %w", err))
	}
	return plaintext
}
