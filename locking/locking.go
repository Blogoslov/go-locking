package locking

import (
	"os"
	"syscall"
	"time"
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

type DirLock string

// create new directory-based lock (creates a subdir, if not exists, but unlocked first)
func NewDirLock(path, name string) (DirLock, error) {
	return DirLock(path + "/.lock-" + name), nil
}

func (lock DirLock) Lock() error {
	var (
		ok  bool
		err error
	)
	for {
		if ok, err = lock.TryLock(); ok && err == nil {
			return nil
		}
		if err != nil {
			return err
		}
		time.Sleep(1)
	}
	panic("unreachable")
}

// acquire the lock, non-blocking
func (lock DirLock) TryLock() (bool, error) {
	err := os.Mkdir(string(lock), 0400)
	if err == nil {
		return true, nil
	}
	return false, nil
}

func (lock DirLock) Unlock() error {
	return os.Remove(string(lock))
}
