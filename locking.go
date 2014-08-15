// Copyright 2013 Tamás Gulácsi. All rights reserved.
// Use of this source code is governed by an Apache 2.0
// license that can be found in the LICENSE file.

// Package locking contains file- and network (port) locking primitives
package locking

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"syscall"
	"time"
)

// AlreadyLocked is an error
var AlreadyLocked = errors.New("AlreadyLocked")

// FLock is a file-based lock
type FLock struct {
	fh *os.File
}

// NewFLock creates new Flock-based lock (unlocked first)
func NewFLock(path string) (FLock, error) {
	fh, err := os.Open(path)
	if err != nil {
		return FLock{}, err
	}
	return FLock{fh: fh}, nil
}

// Lock acquires the lock, blocking
func (lock FLock) Lock() error {
	return syscall.Flock(int(lock.fh.Fd()), syscall.LOCK_EX)
}

// TryLock acquires the lock, non-blocking
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

// Unlock releases the lock
func (lock FLock) Unlock() error {
	lock.fh.Close()
	return syscall.Flock(int(lock.fh.Fd()), syscall.LOCK_UN)
}

// FLocks is an array of FLocks, Unlockable at once
type FLocks []FLock

// FLockDirs returns FLocks for each directory
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

// Unlock all locks
func (locks FLocks) Unlock() {
	for _, lock := range locks {
		lock.Unlock()
	}
}

// DirLock is a directory lock
type DirLock string

// NewDirLock create new directory-based lock
// (creates a subdir, if not exists, but unlocked first)
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

// Lock locks (creates .lock subdir)
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
}

// TryLock acquires the lock, non-blocking
func (lock DirLock) TryLock() (bool, error) {
	err := os.Mkdir(string(lock), 0600)
	if err == nil {
		return true, nil
	}
	return false, nil
}

// Unlock releases the directory lock
func (lock DirLock) Unlock() error {
	return os.Remove(string(lock))
}

// PortLock is a locker which locks by binding to a port on the loopback IPv4 interface
type PortLock struct {
	hostport string
	ln       net.Listener
}

// NewPortLock returns a lock for port
func NewPortLock(port int) *PortLock {
	return &PortLock{hostport: net.JoinHostPort("127.0.0.1", strconv.Itoa(port))}
}

// Lock locks on port
func (p *PortLock) Lock() {
	t := 1 * time.Second
	for {
		if l, err := net.Listen("tcp", p.hostport); err == nil {
			p.ln = l	// thanks to zhangpy
			return
		}
		//log.Printf("spinning lock on %s (%s)", p.hostport, err)
		time.Sleep(t)
		t = time.Duration(float32(t) * 1.2)
	}
}

// Unlock unlocks the port lock
func (p *PortLock) Unlock() {
	if p.ln != nil {
		p.ln.Close()
	}
}
