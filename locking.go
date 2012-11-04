package locking

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

var AlreadyLocked = errors.New("AlreadyLocked")

type FLock struct {
	fh *os.File
}
type FLocks []FLock

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

func FLockDirs(dirs ...string) (FLocks, error) {
	locks := make([]FLock, 0, len(dirs))
	allright := false
	defer func() {
		if !allright {
			for _, lock := range locks {
				lock.Unlock()
			}
		}
	}()
	var (
		err  error
		ok   bool
		lock FLock
	)
	for _, path := range dirs {
		if lock, err = NewFLock(path); err != nil {
			return nil, err
		}
		if ok, err = lock.TryLock(); err != nil {
			return nil, err
		} else if !ok {
			return nil, AlreadyLocked
		}
		locks = append(locks, lock)
	}
	allright = true
	return FLocks(locks), nil
}

func (locks FLocks) Unlock() {
	for _, lock := range locks {
		lock.Unlock()
	}
}

type DirLock string

// create new directory-based lock (creates a subdir, if not exists, but unlocked first)
// WARNING: no automatic Unlock on exit/panic!
func NewDirLock(path string) (DirLock, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return DirLock(""), err
	}
	if fi.IsDir() {
		path = filepath.Join(path, ".lock")
	} else {
		path = path + ".lock"
	}
	return DirLock(path), nil
}

// locks (creates .lock subdir)
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
	err := os.Mkdir(string(lock), 0600)
	if err == nil {
		return true, nil
	}
	return false, nil
}

func (lock DirLock) Unlock() error {
	return os.Remove(string(lock))
}
