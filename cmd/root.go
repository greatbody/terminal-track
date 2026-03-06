package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "tt",
	Short: "terminal-track — record and browse your shell history",
	Long: `terminal-track (tt) transparently records every command you type
across all your terminals and tmux sessions, storing them with
timestamps, working directories, and exit codes.

Use 'tt serve' to browse your history in a web-based timeline.`,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}
