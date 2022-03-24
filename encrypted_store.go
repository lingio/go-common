package common

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"io"
	"net/http"
)

var ErrDecrypt = errors.New("encrypted store: decryption error")

var encstoreV1Header = []byte{}               // there was no header in version 1 :(
var encstoreV2Header = [...]byte{'v', 2, '/'} // magic byte sequence for detecting encryption store scheme

// EncryptedStore
type EncryptedStore struct {
	backend LingioStore

	cipher cipher.Block // v1: ciphertext[16b]||plaintext
	aesgcm cipher.AEAD  // v2: header[3b]||nonce[12b]||ciphertext
}

// NewEncryptedStore
func NewEncryptedStore(backend LingioStore, cipherKey string) (*EncryptedStore, error) {
	if len(cipherKey) != 32 {
		return nil, errors.New("cipherKey must be 32 chars")
	}

	block, err := aes.NewCipher([]byte(cipherKey))
	if err != nil {
		return nil, fmt.Errorf("cipher: %w", err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("aes gcm: %w", err)
	}

	return &EncryptedStore{
		backend: backend,
		cipher:  block,
		aesgcm:  aesgcm,
	}, nil
}

func (es *EncryptedStore) GetObject(file string) ([]byte, ObjectInfo, *Error) {
	data, info, err := es.backend.GetObject(es.encryptFilename(file))
	if err != nil {
		return nil, ObjectInfo{}, err
	}

	if filename, err := es.decryptFilename(info.Key); err != nil {
		return nil, ObjectInfo{}, NewErrorE(http.StatusInternalServerError, err)
	} else {
		info.Key = filename
	}
	plaintext := es.decrypt(data)

	return plaintext, info, nil
}

func (es *EncryptedStore) PutObject(ctx context.Context, file string, data []byte) (ObjectInfo, *Error) {
	encdata := es.encrypt(data)
	encfile := es.encryptFilename(file)

	info, err := es.backend.PutObject(ctx, encfile, encdata)
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

func (es EncryptedStore) StoreName() string {
	return es.backend.StoreName()
}

func (es *EncryptedStore) encryptFilename(file string) string {
	tmp := es.encrypt([]byte(file))
	return base32.StdEncoding.EncodeToString(tmp)
}

func (es *EncryptedStore) decryptFilename(file string) (_ string, err error) {
	// guard against Decrypt throwing a panic
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrDecrypt, r)
		}
	}()
	tmp, err := base32.StdEncoding.DecodeString(file)
	if err != nil {
		return "", fmt.Errorf("base32 decode: %q: %w", file, err)
	}
	tmp = es.decrypt(tmp)
	return string(tmp), nil
}

func (es *EncryptedStore) decrypt(data []byte) []byte {
	var plaintext []byte
	if bytes.HasPrefix(data, encstoreV2Header[:]) {
		offset := len(encstoreV2Header)
		nonce := data[offset : offset+es.aesgcm.NonceSize()]
		offset += es.aesgcm.NonceSize()
		ciphertext := data[offset:]

		var err error
		plaintext, err = es.aesgcm.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			panic(fmt.Errorf("encrypted store: decrypt: %w", err))
		}
	} else {
		// fallback to v1 scheme which only encrypted the first block (16 bytes)
		es.cipher.Decrypt(data, data)
		plaintext = data
	}
	return plaintext
}

func (es *EncryptedStore) encrypt(data []byte) []byte {
	nonce := make([]byte, es.aesgcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		panic(fmt.Errorf("encrypted store: could not generate nonce: %w", err))
	}

	// ciphertext reuses data slice
	ciphertext := es.aesgcm.Seal(data[:0], nonce, data, nil)

	// header||nonce||ciphertext
	blob := encstoreV2Header[:]
	blob = append(blob, nonce...)
	blob = append(blob, ciphertext...)
	return blob
}
