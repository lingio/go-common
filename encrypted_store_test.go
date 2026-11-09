package common

import (
	"bytes"
	"testing"
)

func TestEncryptDataLeavesInputUntouched(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	for name, newCrypto := range map[string]func([]byte) (cryptoModule, error){
		"v1": newV1Crypto,
		"v2": newV2Crypto,
	} {
		t.Run(name, func(t *testing.T) {
			c, err := newCrypto(key)
			if err != nil {
				t.Fatal(err)
			}
			data := []byte("some plaintext that is longer than one block")
			orig := bytes.Clone(data)

			enc := c.encryptData(nil, data)
			if !bytes.Equal(data, orig) {
				t.Fatalf("encryptData modified input: %q", data)
			}
			if got := c.decryptData(enc); !bytes.Equal(got, orig) {
				t.Fatalf("round trip: got %q, want %q", got, orig)
			}
		})
	}
}
