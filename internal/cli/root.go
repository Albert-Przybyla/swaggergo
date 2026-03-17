package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "swaggergo",
	Short: "Swagger generator CLI",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(generateCmd)
	generateCmd.Flags().StringP("config", "c", ".swaggergo.yaml", "config file path")
	generateCmd.Flags().StringP("output", "o", "", "output directory (overrides config)")
	generateCmd.Flags().BoolP("verbose", "v", false, "verbose output")
	rootCmd.AddCommand(versionCmd)
}
