package browser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadDevToolsActivePort(t *testing.T) {
	path := filepath.Join(t.TempDir(), devToolsActivePortFile)
	if err := os.WriteFile(path, []byte("4242\n/devtools/browser/abc\n"), 0o600); err != nil {
		t.Fatalf("write DevToolsActivePort: %v", err)
	}
	port, wsPath, err := readDevToolsActivePort(path)
	if err != nil {
		t.Fatalf("readDevToolsActivePort() error = %v", err)
	}
	if port != 4242 || wsPath != "/devtools/browser/abc" {
		t.Fatalf("port/wsPath = %d/%q, want 4242 /devtools/browser/abc", port, wsPath)
	}
}

func TestReadDevToolsActivePortRejectsInvalidData(t *testing.T) {
	tests := map[string]string{
		"missing ws":    "4242\n",
		"bad port":      "nope\n/devtools/browser/abc\n",
		"relative path": "4242\ndevtools/browser/abc\n",
	}
	for name, data := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), devToolsActivePortFile)
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatalf("write DevToolsActivePort: %v", err)
			}
			if _, _, err := readDevToolsActivePort(path); err == nil {
				t.Fatal("readDevToolsActivePort() error = nil, want error")
			}
		})
	}
}
