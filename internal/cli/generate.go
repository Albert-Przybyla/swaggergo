package cli

import (
	"fmt"

	"github.com/Albert-Przybyla/swaggergo/internal/service"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate swagger files",
	Run: func(cmd *cobra.Command, args []string) {
		cfgPath, _ := cmd.Flags().GetString("config")
		outputOverride, _ := cmd.Flags().GetString("output")
		verbose, _ := cmd.Flags().GetBool("verbose")

		err := service.Generate(&service.GenerateOpts{
			CfgPath:        cfgPath,
			OutputOverride: outputOverride,
			Verbose:        verbose,
		})
		if err != nil {
			fmt.Println("Error:", err)
		}
	},
}
