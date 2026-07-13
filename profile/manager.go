package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/dtonair/browctl/protocol"
)

const (
	ChromeDataDirName = "chrome-data"
	ProfileFileName   = "profile.json"
	LockFileName      = "profile.lock"
)

var validName = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type Policy struct {
	Capabilities []string `json:"capabilities,omitempty"`
}

type Profile struct {
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	ChromePath string    `json:"chrome_path,omitempty"`
	Policy     Policy    `json:"policy"`
}

type Manager struct {
	profilesDir string
}

func NewManager(profilesDir string) *Manager {
	return &Manager{profilesDir: profilesDir}
}

func (m *Manager) Create(name, chromePath string) (Profile, error) {
	if err := ValidateName(name); err != nil {
		return Profile{}, err
	}
	if err := os.MkdirAll(m.profilesDir, 0o700); err != nil {
		return Profile{}, fmt.Errorf("create profiles dir: %w", err)
	}

	dir := m.ProfileDir(name)
	if _, err := os.Stat(filepath.Join(dir, ProfileFileName)); err == nil {
		return Profile{}, protocol.NewError(protocol.InvalidRequest, "profile already exists", map[string]any{"profile": name})
	} else if err != nil && !os.IsNotExist(err) {
		return Profile{}, fmt.Errorf("stat profile: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(dir, ChromeDataDirName), 0o700); err != nil {
		return Profile{}, fmt.Errorf("create profile dirs: %w", err)
	}
	if err := os.Chmod(dir, 0o700); err != nil {
		return Profile{}, fmt.Errorf("chmod profile dir: %w", err)
	}
	if err := os.Chmod(filepath.Join(dir, ChromeDataDirName), 0o700); err != nil {
		return Profile{}, fmt.Errorf("chmod chrome data dir: %w", err)
	}

	p := Profile{Name: name, CreatedAt: time.Now().UTC(), ChromePath: chromePath, Policy: Policy{}}
	if err := m.writeProfile(p); err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (m *Manager) List() ([]Profile, error) {
	entries, err := os.ReadDir(m.profilesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []Profile{}, nil
		}
		return nil, fmt.Errorf("read profiles dir: %w", err)
	}

	profiles := make([]Profile, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		p, err := m.Inspect(entry.Name())
		if err != nil {
			continue
		}
		profiles = append(profiles, p)
	}
	sort.Slice(profiles, func(i, j int) bool { return profiles[i].Name < profiles[j].Name })
	return profiles, nil
}

func (m *Manager) Inspect(name string) (Profile, error) {
	if err := ValidateName(name); err != nil {
		return Profile{}, err
	}
	data, err := os.ReadFile(filepath.Join(m.ProfileDir(name), ProfileFileName))
	if err != nil {
		if os.IsNotExist(err) {
			return Profile{}, protocol.NewError(protocol.ProfileNotFound, "profile not found", map[string]any{"profile": name})
		}
		return Profile{}, fmt.Errorf("read profile: %w", err)
	}
	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return Profile{}, fmt.Errorf("decode profile: %w", err)
	}
	return p, nil
}

func (m *Manager) Delete(name string) error {
	if _, err := m.Inspect(name); err != nil {
		return err
	}
	if locked, err := m.LockInfo(name); err == nil && locked.PID != 0 && isProcessAlive(locked.PID) {
		return protocol.NewError(protocol.ProfileLocked, "profile is locked", map[string]any{"profile": name, "pid": locked.PID})
	}
	if err := os.RemoveAll(m.ProfileDir(name)); err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}
	return nil
}

func (m *Manager) Acquire(name string, info LockInfo) (*Lock, error) {
	if _, err := m.Inspect(name); err != nil {
		return nil, err
	}
	return acquireLock(m.lockPath(name), info)
}

func (m *Manager) Release(lock *Lock) error {
	if lock == nil {
		return nil
	}
	return lock.Release()
}

func (m *Manager) LockInfo(name string) (LockInfo, error) {
	if err := ValidateName(name); err != nil {
		return LockInfo{}, err
	}
	return readLockInfo(m.lockPath(name))
}

func (m *Manager) ReleaseLockForPID(name string, pid int) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	info, err := readLockInfo(m.lockPath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.PID != pid {
		return protocol.NewError(protocol.ProfileLocked, "lock is owned by another process", map[string]any{"pid": info.PID})
	}
	if err := os.Remove(m.lockPath(name)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("release lock for pid: %w", err)
	}
	return nil
}

func (m *Manager) ProfileDir(name string) string {
	return filepath.Join(m.profilesDir, name)
}

func (m *Manager) ChromeDataDir(name string) string {
	return filepath.Join(m.ProfileDir(name), ChromeDataDirName)
}

func (m *Manager) writeProfile(p Profile) error {
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("encode profile: %w", err)
	}
	path := filepath.Join(m.ProfileDir(p.Name), ProfileFileName)
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write profile: %w", err)
	}
	return nil
}

func (m *Manager) lockPath(name string) string {
	return filepath.Join(m.ProfileDir(name), LockFileName)
}

func ValidateName(name string) error {
	if name == "" {
		return protocol.NewError(protocol.InvalidRequest, "profile name is required", nil)
	}
	if !validName.MatchString(name) || name == "." || name == ".." {
		return protocol.NewError(protocol.InvalidRequest, "invalid profile name", map[string]any{"profile": name})
	}
	return nil
}
