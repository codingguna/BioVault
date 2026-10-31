package security

import (
	"errors"
	"fmt"
	"syscall"
	"unsafe"
)

var (
	crypt32              = syscall.NewLazyDLL("crypt32.dll")
	kernel32             = syscall.NewLazyDLL("kernel32.dll")
	procCryptProtect     = crypt32.NewProc("CryptProtectData")
	procCryptUnprotect   = crypt32.NewProc("CryptUnprotectData")
	procLocalFree        = kernel32.NewProc("LocalFree")
)

const (
	cryptprotectUIForbidden = 0x1
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

// EncryptTemplate encrypts biometric template bytes at rest using Windows DPAPI.
// Only the current Windows user / system can decrypt it.
func EncryptTemplate(plain []byte) ([]byte, error) {
	if len(plain) == 0 {
		return nil, errors.New("cannot encrypt empty template")
	}

	inBlob := newBlob(plain)
	var outBlob dataBlob

	desc, _ := syscall.UTF16PtrFromString("MFS100 Fingerprint Template")

	r, _, err := procCryptProtect.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		uintptr(unsafe.Pointer(desc)),
		0, // optional entropy
		0, // reserved
		0, // prompt struct
		cryptprotectUIForbidden,
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if r == 0 {
		return nil, fmt.Errorf("CryptProtectData failed: %w", err)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(outBlob.pbData)))

	return outBlob.toBytes(), nil
}

// DecryptTemplate decrypts a DPAPI-protected template blob.
func DecryptTemplate(cipher []byte) ([]byte, error) {
	if len(cipher) == 0 {
		return nil, errors.New("cannot decrypt empty ciphertext")
	}

	inBlob := newBlob(cipher)
	var outBlob dataBlob

	r, _, err := procCryptUnprotect.Call(
		uintptr(unsafe.Pointer(&inBlob)),
		0, // description out
		0, // optional entropy
		0, // reserved
		0, // prompt struct
		cryptprotectUIForbidden,
		uintptr(unsafe.Pointer(&outBlob)),
	)
	if r == 0 {
		return nil, fmt.Errorf("CryptUnprotectData failed: %w", err)
	}
	defer procLocalFree.Call(uintptr(unsafe.Pointer(outBlob.pbData)))

	return outBlob.toBytes(), nil
}
