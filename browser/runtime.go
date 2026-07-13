package browser

import "time"

type State string

const (
	StateRunning State = "running"
	StateStopped State = "stopped"
	StateCrashed State = "crashed"
)

type Runtime struct {
	Profile    string    `json:"profile"`
	PID        int       `json:"pid"`
	Port       int       `json:"port"`
	Endpoint   string    `json:"endpoint"`
	WSEndpoint string    `json:"ws_endpoint"`
	ChromePath string    `json:"chrome_path"`
	StartedAt  time.Time `json:"started_at"`
	State      State     `json:"state"`
}
