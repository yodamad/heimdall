package build

import (
	goversion "github.com/caarlos0/go-version"
)

// vars could be modified with ldflags
var (
	Version = "devel"
	Commit  = ""
	Date    = ""
)

func GetBuildInfos() goversion.Info {
	return goversion.GetVersionInfo(
		func(i *goversion.Info) {
			if Commit != "" {
				i.GitCommit = Commit
			}
			if Date != "" {
				i.BuildDate = Date
			}
			if Version != "devel" {
				i.GitVersion = Version
			}
		},
	)
}

var BuildInfos = GetBuildInfos()
