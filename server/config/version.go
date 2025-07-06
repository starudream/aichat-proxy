package config

import (
	"bytes"
	"fmt"
	"runtime"
	"runtime/debug"
	"sync"
	"text/tabwriter"
	"time"
)

type Version struct {
	GitVersion   string `json:"gitVersion"`
	GitCommit    string `json:"gitCommit"`
	GitTreeState string `json:"gitTreeState"`
	BuildDate    string `json:"buildDate"`
	GoVersion    string `json:"goVersion"`
	Compiler     string `json:"compiler"`
	Platform     string `json:"platform"`
}

func (v Version) String() string {
	b := &bytes.Buffer{}
	w := tabwriter.NewWriter(b, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "GitVersion:\t%s\n", v.GitVersion)
	_, _ = fmt.Fprintf(w, "GitCommit:\t%s\n", v.GitCommit)
	_, _ = fmt.Fprintf(w, "GitTreeState:\t%s\n", v.GitTreeState)
	_, _ = fmt.Fprintf(w, "BuildDate:\t%s\n", v.BuildDate)
	_, _ = fmt.Fprintf(w, "GoVersion:\t%s\n", v.GoVersion)
	_, _ = fmt.Fprintf(w, "Compiler:\t%s\n", v.Compiler)
	_, _ = fmt.Fprintf(w, "Platform:\t%s\n", v.Platform)
	_ = w.Flush()
	return b.String()
}

var (
	// build by -ldflags
	gitVersion string

	version     Version
	versionOnce sync.Once
)

func GetVersion() Version {
	versionOnce.Do(func() {
		bi, ok := debug.ReadBuildInfo()
		if !ok || bi == nil {
			return
		}

		stm := map[string]string{}
		for _, s := range bi.Settings {
			stm[s.Key] = s.Value
		}

		if gitVersion != "" {
			version.GitVersion = gitVersion
		} else if bi.Main.Version != "(devel)" {
			version.GitVersion = bi.Main.Version
		} else {
			version.GitVersion = "v0.0.0"
		}
		version.GitCommit = stm["vcs.revision"]
		version.GitTreeState = "clean"
		if stm["vcs.modified"] == "true" {
			version.GitTreeState = "dirty"
		}

		t, err := time.Parse(time.RFC3339Nano, stm["vcs.time"])
		if err == nil {
			version.BuildDate = t.Local().Format(time.RFC3339Nano)
		}

		version.GoVersion = runtime.Version()
		version.Compiler = runtime.Compiler
		version.Platform = runtime.GOOS + "/" + runtime.GOARCH
	})
	return version
}
