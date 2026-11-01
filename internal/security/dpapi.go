package security

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows/registry"
)

var (
	crypt32            = syscall.NewLazyDLL("crypt32.dll")
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	procCryptUnprotect = crypt32.NewProc("CryptUnprotectData")
	procLocalFree      = kernel32.NewProc("LocalFree")
)

const (
	cryptprotectUIForbidden  = 0x1
	cryptprotectLocalMachine = 0x4
)

type dataBlob struct {
	cbData uint32
	pbData *byte
}

func newBlob(d []byte) dataBlob {
	if len(d) == 0 {
		return dataBlob{}
	}
	return dataBlob{
		cbData: uint32(len(d)),
		pbData: &d[0],
	}
}

func (b *dataBlob) toBytes() []byte {
	if b.cbData == 0 || b.pbData == nil {
		return nil
	}
	out := make([]byte, b.cbData)
	copy(out, unsafe.Slice(b.pbData, b.cbData))
	return out
}

var aesPrefix = []byte("MFS100_GCM_v1:")

func getMachineKey() []byte {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\Cryptography`, registry.QUERY_VALUE)
	var guid string
	if err == nil {
		guid, _, _ = k.GetStringValue("MachineGuid")
		k.Close()
	}
	if guid == "" {
		guid = "c7e60385-8a00-4813-ad1e-0371dc52650a"
	}
	h := sha256.Sum256([]byte("MFS100_SECURE_VAULT_KEY_" + guid))
	return h[:]
}

// EncryptTemplate encrypts biometric template bytes or passwords using AES-256-GCM keyed to this machine.
// This allows both interactive users and background services (SYSTEM) to decrypt reliably across all sessions.
func EncryptTemplate(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return nil, errors.New("cannot encrypt empty data")
	}

	key := getMachineKey()
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	sealed := gcm.Seal(nonce, nonce, plain, nil)
	out := make([]byte, len(aesPrefix)+len(sealed))
	copy(out, aesPrefix)
	copy(out[len(aesPrefix):], sealed)
	return out, nil
}

// DecryptTemplate decrypts data using AES-256-GCM, with backward-compatible DPAPI fallback.
func DecryptTemplate(cipherData []byte) ([]byte, error) {
	if len(cipherData) == 0 {
		return nil, errors.New("cannot decrypt empty ciphertext")
	}

	// 1. Check for AES-GCM prefix
	if bytes.HasPrefix(cipherData, aesPrefix) {
		raw := cipherData[len(aesPrefix):]
		key := getMachineKey()
		block, err := aes.NewCipher(key)
		if err != nil {
			return nil, err
		}
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, err
		}
		nonceSize := gcm.NonceSize()
		if len(raw) < nonceSize {
			return nil, errors.New("malformed ciphertext")
		}
		nonce, ciphertext := raw[:nonceSize], raw[nonceSize:]
		return gcm.Open(nil, nonce, ciphertext, nil)
	}

	// 2. Backward compatibility: Fallback to DPAPI
	return decryptDPAPI(cipherData)
}

func decryptDPAPI(cipher []byte) ([]byte, error) {
	inBlob := newBlob(cipher)
	var outBlob dataBlob

	// Try with machine-level flag
	r, _, _ := procCryptUnprotect.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, 0, 0, 0,
		cryptprotectUIForbidden|cryptprotectLocalMachine,
		uintptr(unsafe.Pointer(&outBlob)),
	)

	// Fallback to user-level flag if machine-level failed
	if r == 0 {
		r, _, _ = procCryptUnprotect.Call(
			uintptr(unsafe.Pointer(&inBlob)),
			0, 0, 0, 0,
			cryptprotectUIForbidden,
			uintptr(unsafe.Pointer(&outBlob)),
		)
	}

	if r == 0 {
		return nil, fmt.Errorf("DPAPI decryption failed")
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(outBlob.pbData)))

	return outBlob.toBytes(), nil
}
