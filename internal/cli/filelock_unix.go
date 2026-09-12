//go:build !windows

package cli

import (
	"os"

	"golang.org/x/sys/unix"
)

func lockIndexFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}

func unlockIndexFile(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}
