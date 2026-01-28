package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"

	"github.com/Okenamay/securawr/internal/version"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of SecuRawr",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("SecuRawr Client\n")
		fmt.Printf("Version: %s\n", version.Version)
		fmt.Printf("Build Date: %s\n", version.BuildDate)
		fmt.Printf("Go Version: %s\n", runtime.Version())
		fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
