package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/luma-cli/lumer-cli/internal/output"
)

var rootCmd = &cobra.Command{
	Use:   "luma-cli",
	Short: "Luma CLI — AI-powered video creation toolkit",
	Version: version,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if jsonFlag, _ := cmd.Flags().GetBool("json"); jsonFlag {
			runtimeOpts.JSON = true
		}
		setupNotices()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().Bool("json", false, "JSON output mode")
}

// Execute runs the root command and returns the OS exit code.
func Execute(args []string) int {
	// Strip argv[0] (program name or path)
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		args = args[1:]
	}
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		if runtimeOpts.JSON {
			return output.WriteError(err)
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}
