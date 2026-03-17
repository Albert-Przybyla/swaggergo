package cli

import (
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate swagger files",
	Run: func(cmd *cobra.Command, args []string) {
		// err := service.GenerateSwagger()
		// if err != nil {
		// 	fmt.Println("Error:", err)
		// }
	},
}
