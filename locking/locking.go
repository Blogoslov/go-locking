package locking

import (
	"os"
	"syscall"
)

type FLock struct {
	fh *os.File
}

//create new Flock-based lock (unlocked first)
func NewFLock(path string) (FLock, error) {
	fh, err := os.Open(path)
	if err != nil {
		return FLock{}, err
	}
	return FLock{fh: fh}, nil
}

// acquire the lock, blocking
func (lock FLock) Lock() error {
	return syscall.Flock(int(lock.fh.Fd()), syscall.LOCK_EX)
}

// acquire the lock, non-blocking
func (lock FLock) TryLock() (bool, error) {
	err := syscall.Flock(int(lock.fh.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	switch err {
	case nil:
		return true, nil
	case syscall.EWOULDBLOCK:
		return false, nil
	}
	return false, err
}

func (lock FLock) Unlock() error {
	lock.fh.Close()
	return syscall.Flock(int(lock.fh.Fd()), syscall.LOCK_UN)
}
