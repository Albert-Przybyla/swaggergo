package cli

import (
	"fmt"

	"github.com/Albert-Przybyla/swaggergo/internal/service"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize swagger config",
	Run: func(cmd *cobra.Command, args []string) {
		err := service.InitProject()
		if err != nil {
			fmt.Println("Error:", err)
		}
	},
}
