package security

import (
	"bytes"
	"testing"
)

func TestDPAPIRoundTrip(t *testing.T) {
	orig := []byte("ISO-19794-2-Mantra-Test-Template-Bytes-123456789")
	enc, err := EncryptTemplate(orig)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if bytes.Equal(orig, enc) {
		t.Fatal("Encrypted bytes should not equal original")
	}

	dec, err := DecryptTemplate(enc)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if !bytes.Equal(orig, dec) {
		t.Fatalf("Decrypted mismatch: got %s, want %s", string(dec), string(orig))
	}
}
