package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	appDirName    = ".browctl"
	runtimeSubdir = "run"
	profilesDir   = "profiles"
	artifactsDir  = "artifacts"
	daemonSock    = "daemon.sock"
)

type Layout struct {
	HomeDir      string
	BaseDir      string
	RuntimeDir   string
	ProfilesDir  string
	ArtifactsDir string
	DaemonSocket string
}

// Resolve returns browctl's filesystem layout without creating directories.
//
// Runtime dir preference:
//  1. $XDG_RUNTIME_DIR/browctl when XDG_RUNTIME_DIR is set.
//  2. ~/.browctl/run fallback.
func Resolve() (Layout, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Layout{}, fmt.Errorf("resolve home dir: %w", err)
	}
	if home == "" {
		return Layout{}, errors.New("resolve home dir: empty home directory")
	}

	base := filepath.Join(home, appDirName)
	runtime := filepath.Join(base, runtimeSubdir)
	if xdg := os.Getenv("XDG_RUNTIME_DIR"); xdg != "" {
		runtime = filepath.Join(xdg, "browctl")
	}

	return Layout{
		HomeDir:      home,
		BaseDir:      base,
		RuntimeDir:   runtime,
		ProfilesDir:  filepath.Join(base, profilesDir),
		ArtifactsDir: filepath.Join(base, artifactsDir),
		DaemonSocket: filepath.Join(runtime, daemonSock),
	}, nil
}

// Ensure creates the directories needed by browctl with owner-only permissions.
func Ensure() (Layout, error) {
	layout, err := Resolve()
	if err != nil {
		return Layout{}, err
	}

	for _, dir := range []string{layout.BaseDir, layout.RuntimeDir, layout.ProfilesDir, layout.ArtifactsDir} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return Layout{}, fmt.Errorf("create %s: %w", dir, err)
		}
		if err := chmodOwnerOnly(dir); err != nil {
			return Layout{}, err
		}
	}

	return layout, nil
}

func chmodOwnerOnly(dir string) error {
	if err := os.Chmod(dir, 0o700); err != nil {
		return fmt.Errorf("chmod %s: %w", dir, err)
	}
	return nil
}
