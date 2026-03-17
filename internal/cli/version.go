package cli

import (
	"fmt"

	"github.com/Albert-Przybyla/swaggergo/internal/build"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI version info",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf(
			"Version: %s\nCommit: %s\nDate: %s\n",
			build.Version,
			build.Commit,
			build.Date,
		)
	},
}
