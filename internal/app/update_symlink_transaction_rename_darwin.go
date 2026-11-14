//go:build darwin

package app

import (
	"unsafe"

	"golang.org/x/sys/unix"
)

func renameNoReplaceAt(parentFD int, from, to string) error {
	return renameNoReplaceAcrossAt(parentFD, from, parentFD, to)
}

func renameNoReplaceAcrossAt(fromFD int, from string, toFD int, to string) error {
	fromPointer, err := unix.BytePtrFromString(from)
	if err != nil {
		return err
	}
	toPointer, err := unix.BytePtrFromString(to)
	if err != nil {
		return err
	}
	const renameExclusive = 0x00000004
	_, _, errno := unix.Syscall6(
		unix.SYS_RENAMEATX_NP,
		uintptr(fromFD), uintptr(unsafe.Pointer(fromPointer)),
		uintptr(toFD), uintptr(unsafe.Pointer(toPointer)),
		renameExclusive, 0,
	)
	if errno != 0 {
		return errno
	}
	return nil
}
