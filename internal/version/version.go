package version

var (
	Version = "dev"
	Commit  = "unknown"
	Date    = "unknown"
)

func Full() string {
	if Commit == "unknown" {
		return Version
	}

	return Version + " (" + Commit + ", " + Date + ")"
}
