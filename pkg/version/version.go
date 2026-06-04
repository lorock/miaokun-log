package version

var (
	Version   = "0.5.1"
	BuildDate = ""
	GitCommit = ""
)

type Info struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	GitCommit string `json:"git_commit"`
}

func GetInfo() Info {
	return Info{
		Version:   Version,
		BuildDate: BuildDate,
		GitCommit: GitCommit,
	}
}
