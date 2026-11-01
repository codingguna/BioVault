package security

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	netapi32             = syscall.NewLazyDLL("netapi32.dll")
	procNetUserEnum      = netapi32.NewProc("NetUserEnum")
	procNetApiBufferFree = netapi32.NewProc("NetApiBufferFree")
)

type userInfo1 struct {
	Name        *uint16
	Password    *uint16
	PasswordAge uint32
	Priv        uint32
	HomeDir     *uint16
	Comment     *uint16
	Flags       uint32
	ScriptPath  *uint16
}

const (
	filterNormalAccount = 0x0002
	ufAccountDisable    = 0x0002
)

// GetActiveLocalUsers returns the list of active (enabled) human local Windows accounts on this machine.
func GetActiveLocalUsers() ([]string, error) {
	var bufptr uintptr
	var entriesRead, totalEntries, resumeHandle uint32

	ret, _, _ := procNetUserEnum.Call(
		0,                   // local computer
		1,                   // level 1 (userInfo1)
		filterNormalAccount, // normal user accounts only
		uintptr(unsafe.Pointer(&bufptr)),
		0xFFFFFFFF,          // max length
		uintptr(unsafe.Pointer(&entriesRead)),
		uintptr(unsafe.Pointer(&totalEntries)),
		uintptr(unsafe.Pointer(&resumeHandle)),
	)

	if ret != 0 {
		// Fallback to current username if NetUserEnum fails
		u := os.Getenv("USERNAME")
		if u != "" {
			return []string{u}, nil
		}
		return nil, fmt.Errorf("NetUserEnum failed with code %d", ret)
	}
	defer procNetApiBufferFree.Call(bufptr)

	var users []string
	itemSize := unsafe.Sizeof(userInfo1{})

	for i := uint32(0); i < entriesRead; i++ {
		entry := (*userInfo1)(unsafe.Pointer(bufptr + uintptr(i)*itemSize))
		if (entry.Flags & ufAccountDisable) != 0 {
			continue // Account is disabled
		}
		name := syscall.UTF16ToString((*[512]uint16)(unsafe.Pointer(entry.Name))[:])
		if name == "DefaultAccount" || name == "WDAGUtilityAccount" || name == "Guest" {
			continue
		}
		users = append(users, name)
	}

	if len(users) == 0 {
		u := os.Getenv("USERNAME")
		if u != "" {
			users = append(users, u)
		}
	}

	return users, nil
}
