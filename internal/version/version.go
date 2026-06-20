package version

const (
	name    = "lyrike-studio-tui"
	current = "0.1.0-dev"
)

func Label() string {
	return name + " " + current
}
