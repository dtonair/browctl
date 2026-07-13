package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"syscall"
	"time"

	"github.com/dt/browctl/protocol"
)

type LockInfo struct {
	PID       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
	Endpoint  string    `json:"endpoint,omitempty"`
}

type Lock struct {
	path string
	info LockInfo
}

func (l *Lock) Info() LockInfo {
	if l == nil {
		return LockInfo{}
	}
	return l.info
}

func (l *Lock) Update(info LockInfo) error {
	if l == nil || l.path == "" {
		return nil
	}
	if info.PID == 0 {
		info.PID = os.Getpid()
	}
	if info.StartedAt.IsZero() {
		info.StartedAt = time.Now().UTC()
	} else {
		info.StartedAt = info.StartedAt.UTC()
	}
	current, err := readLockInfo(l.path)
	if err != nil {
		return err
	}
	if current.PID != l.info.PID || !current.StartedAt.Equal(l.info.StartedAt) {
		return protocol.NewError(protocol.ProfileLocked, "lock is owned by another process", map[string]any{"pid": current.PID})
	}
	file, err := os.OpenFile(l.path, os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("open lock for update: %w", err)
	}
	encErr := json.NewEncoder(file).Encode(info)
	closeErr := file.Close()
	if encErr != nil {
		return fmt.Errorf("write lock update: %w", encErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close lock update: %w", closeErr)
	}
	l.info = info
	return nil
}

func (l *Lock) Release() error {
	if l == nil || l.path == "" {
		return nil
	}
	current, err := readLockInfo(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if current.PID != l.info.PID || !current.StartedAt.Equal(l.info.StartedAt) {
		return protocol.NewError(protocol.ProfileLocked, "lock is owned by another process", map[string]any{"pid": current.PID})
	}
	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("release lock: %w", err)
	}
	return nil
}

func acquireLock(path string, info LockInfo) (*Lock, error) {
	if info.PID == 0 {
		info.PID = os.Getpid()
	}
	if info.StartedAt.IsZero() {
		info.StartedAt = time.Now().UTC()
	} else {
		info.StartedAt = info.StartedAt.UTC()
	}

	for {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err == nil {
			encErr := json.NewEncoder(file).Encode(info)
			closeErr := file.Close()
			if encErr != nil {
				_ = os.Remove(path)
				return nil, fmt.Errorf("write lock: %w", encErr)
			}
			if closeErr != nil {
				_ = os.Remove(path)
				return nil, fmt.Errorf("close lock: %w", closeErr)
			}
			return &Lock{path: path, info: info}, nil
		}
		if !errors.Is(err, os.ErrExist) {
			return nil, fmt.Errorf("create lock: %w", err)
		}

		current, readErr := readLockInfo(path)
		if readErr != nil {
			if os.IsNotExist(readErr) {
				continue
			}
			return nil, readErr
		}
		if current.PID != 0 && isProcessAlive(current.PID) {
			return nil, protocol.NewError(protocol.ProfileLocked, "profile is locked", map[string]any{"pid": current.PID, "started_at": current.StartedAt})
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("remove stale lock: %w", err)
		}
	}
}

func readLockInfo(path string) (LockInfo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LockInfo{}, err
	}
	var info LockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return LockInfo{}, fmt.Errorf("decode lock: %w", err)
	}
	return info, nil
}

func isProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil || errors.Is(err, syscall.EPERM)
}
