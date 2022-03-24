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
	// We don't know which crypto gen to use to map plaintext filename to ciphertext filename
	// so we can only do trail and error.
	data, info, err := es.backend.GetObject(es.encryptFilename1(file))
	if err != nil {
		data, info, err = es.backend.GetObject(es.encryptFilename2(file))
		if err != nil {
			return nil, ObjectInfo{}, err
		}
	}

	var plaintext []byte
	if isV2Crypto(data, info.Key) {
		plaintext, _ = es.decrypt2(data)
	} else {
		plaintext = es.decrypt1(data)
	}

	if filename, _, err := es.decryptFilename(info.Key); err != nil {
		return nil, ObjectInfo{}, NewErrorE(http.StatusInternalServerError, err)
	} else {
		info.Key = filename
	}

	return plaintext, info, nil
}

func (es *EncryptedStore) PutObject(ctx context.Context, file string, data []byte) (ObjectInfo, *Error) {
	encdata := es.encrypt2(data, nil) // generate new nonce for every write
	encfile := es.encryptFilename2(file)

	info, err := es.backend.PutObject(ctx, encfile, encdata)
	if err != nil {
		return ObjectInfo{}, err
	}
	info.Key = file
	return info, nil
}

func (es EncryptedStore) DeleteObject(ctx context.Context, file string) *Error {
	// We don't know which crypto gen to use to map plaintext filename to ciphertext filename
	// so we can only do trail and error.
	err := es.backend.DeleteObject(ctx, es.encryptFilename1(file))
	if err != nil {
		return es.backend.DeleteObject(ctx, es.encryptFilename2(file))
	}
	return nil
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
			key, _, err := es.decryptFilename(info.Key)
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

// encryptFilename1
func (es *EncryptedStore) encryptFilename1(plaintext string) string {
	key := []byte(plaintext)
	es.cipher.Encrypt(key, key)
	return base32.StdEncoding.EncodeToString(key)
}

func (es *EncryptedStore) encryptFilename2(plaintext string) string {
	key := []byte(plaintext)
	nonce := key[0:es.aesgcm.NonceSize()]
	ciphertext := es.encrypt2(nonce, key)
	return base32.StdEncoding.EncodeToString(ciphertext)
}

func (es *EncryptedStore) decryptFilename(ciphertext string) (_ string, _ []byte, err error) {
	// guard against Decrypt throwing a panic
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%w: %v", ErrDecrypt, r)
		}
	}()

	decodedCiphertext, err := base32.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", nil, fmt.Errorf("base32 decode: %q: %w", ciphertext, err)
	}

	var plaintext []byte
	var nonce []byte
	if isV2Crypto(nil, ciphertext) {
		plaintext, plaintext = es.decrypt2(decodedCiphertext)
	} else {
		plaintext = es.decrypt1(decodedCiphertext)
	}

	return string(plaintext), nonce, nil
}

func (es *EncryptedStore) decrypt1(data []byte) []byte {
	es.cipher.Decrypt(data, data)
	return data
}

func (es *EncryptedStore) decrypt2(data []byte) ([]byte, []byte) {
	offset := len(encstoreV2Header)
	nonce := data[offset : offset+es.aesgcm.NonceSize()]
	offset += es.aesgcm.NonceSize()
	ciphertext := data[offset:]

	plaintext, err := es.aesgcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(fmt.Errorf("encrypted store: decrypt: %w", err))
	}
	return nonce, plaintext
}

func (es *EncryptedStore) encrypt1(data []byte) []byte {
	es.cipher.Encrypt(data, data)
	return data
}

func (es *EncryptedStore) encrypt2(data []byte, nonce []byte) []byte {
	if nonce == nil {
		nonce = make([]byte, es.aesgcm.NonceSize())
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			panic(fmt.Errorf("encrypted store: could not generate nonce: %w", err))
		}
	}

	// ciphertext reuses data slice
	ciphertext := es.aesgcm.Seal(data[:0], nonce, data, nil)

	// header||nonce||ciphertext
	blob := encstoreV2Header[:]
	blob = append(blob, nonce...)
	blob = append(blob, ciphertext...)
	return blob
}

// ReEncryptObject will fetch an object and re-encrypt it with v2 scheme if
// determined to be old crypto scheme. The the object key must be encrypted.
// Will remove the passed key if re-encryption (to a new filename) succeeds.
func (es *EncryptedStore) ReEncryptObject(ctx context.Context, encObjectKey string) error {
	ciphertext, _, err := es.backend.GetObject(encObjectKey)
	if err != nil {
		return fmt.Errorf("get %q: %w", encObjectKey, err)
	}

	if isV2Crypto(ciphertext, encObjectKey) {
		return fmt.Errorf("already encrypted: %q", encObjectKey)
	}

	filename, _, lerr := es.decryptFilename(encObjectKey)
	if lerr != nil {
		return fmt.Errorf("%q: %w", encObjectKey, lerr)
	}

	plaintext := es.decrypt1(ciphertext)
	encfile := es.encryptFilename2(filename)
	encdata := es.encrypt2(plaintext, nil)

	if _, err := es.backend.PutObject(ctx, encfile, encdata); err != nil {
		return fmt.Errorf("writing new %q: %w", encfile, err)
	}

	// since re-encrypted object will have a differet filename, we must cleanup old object
	if err := es.backend.DeleteObject(ctx, encObjectKey); err != nil {
		return fmt.Errorf("removing old %q: %w", encObjectKey, err)
	}

	return nil
}

func isV2Crypto(data []byte, filename string) bool {
	return bytes.HasPrefix([]byte(filename), encstoreV2Header[:]) &&
		bytes.HasPrefix(data, encstoreV2Header[:])
}
