package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/greatbody/terminal-track/internal/db"
	"github.com/spf13/cobra"
)

var recordCmd = &cobra.Command{
	Use:    "record",
	Short:  "Record a command (called by the zsh hook)",
	Hidden: true,
	RunE:   runRecord,
}

var (
	flagCmd       string
	flagDir       string
	flagExitCode  int
	flagSession   string
	flagTimestamp string
)

func init() {
	recordCmd.Flags().StringVar(&flagCmd, "cmd", "", "The command that was executed")
	recordCmd.Flags().StringVar(&flagDir, "dir", "", "Working directory")
	recordCmd.Flags().IntVar(&flagExitCode, "exit-code", -1, "Exit code (-1 means not captured)")
	recordCmd.Flags().StringVar(&flagSession, "session", "", "Session ID")
	recordCmd.Flags().StringVar(&flagTimestamp, "timestamp", "", "Timestamp (RFC3339)")

	recordCmd.MarkFlagRequired("cmd")
	recordCmd.MarkFlagRequired("dir")

	rootCmd.AddCommand(recordCmd)
}

func runRecord(cmd *cobra.Command, args []string) error {
	if flagCmd == "" {
		return nil
	}

	dbPath, err := db.DefaultDBPath()
	if err != nil {
		return err
	}

	d, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer d.Close()

	ts := time.Now().UTC()
	if flagTimestamp != "" {
		// Try multiple formats
		for _, layout := range []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02T15:04:05Z",
			"2006-01-02T15:04:05",
		} {
			if parsed, err := time.Parse(layout, flagTimestamp); err == nil {
				ts = parsed
				break
			}
		}
	}

	hostname, _ := os.Hostname()

	rec := db.Record{
		Timestamp: ts,
		Command:   flagCmd,
		Directory: flagDir,
		SessionID: flagSession,
		Hostname:  hostname,
	}

	if flagExitCode >= 0 {
		rec.ExitCode = &flagExitCode
	}

	return d.Insert(rec)
}
