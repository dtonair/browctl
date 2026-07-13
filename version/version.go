package version

// These values can be overridden at build time with -ldflags, for example:
//
//	-X github.com/dt/browctl/version.Version=0.1.0
//	-X github.com/dt/browctl/version.Commit=$(git rev-parse --short HEAD)
//	-X github.com/dt/browctl/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)
var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func Get() Info {
	return Info{Version: Version, Commit: Commit, Date: Date}
}
