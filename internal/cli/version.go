package cli

import (
	"fmt"
	"runtime/debug"

	"github.com/Albert-Przybyla/swaggergo/internal/build"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version info",
	Run: func(cmd *cobra.Command, args []string) {
		version := build.Version
		commit := build.Commit
		date := build.Date

		if info, ok := debug.ReadBuildInfo(); ok {
			if version == "dev" {
				version = info.Main.Version
			}

			for _, s := range info.Settings {
				switch s.Key {
				case "vcs.revision":
					if commit == "none" {
						commit = s.Value
					}
				case "vcs.time":
					if date == "unknown" {
						date = s.Value
					}
				}
			}
		}

		fmt.Printf(
			"Version: %s\nCommit: %s\nDate: %s\n",
			version,
			commit,
			date,
		)
	},
}
