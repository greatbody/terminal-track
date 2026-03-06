package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/greatbody/terminal-track/internal/db"
	"github.com/spf13/cobra"
)

var searchCmd = &cobra.Command{
	Use:   "search [pattern]",
	Short: "Search command history",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSearch,
}

var (
	flagSearchDir   string
	flagSearchLimit int
	flagSearchSince string
)

func init() {
	searchCmd.Flags().StringVarP(&flagSearchDir, "dir", "d", "", "Filter by directory")
	searchCmd.Flags().IntVarP(&flagSearchLimit, "limit", "n", 50, "Max results")
	searchCmd.Flags().StringVar(&flagSearchSince, "since", "", "Show commands since (e.g. '1h', '7d', '2024-01-01')")
	rootCmd.AddCommand(searchCmd)
}

func runSearch(cmd *cobra.Command, args []string) error {
	dbPath, err := db.DefaultDBPath()
	if err != nil {
		return err
	}

	d, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer d.Close()

	opts := db.QueryOptions{
		Limit:     flagSearchLimit,
		Directory: flagSearchDir,
	}

	if len(args) > 0 {
		opts.Search = args[0]
	}

	if flagSearchSince != "" {
		since, err := parseSince(flagSearchSince)
		if err != nil {
			return fmt.Errorf("invalid --since: %w", err)
		}
		opts.Since = &since
	}

	records, err := d.Query(opts)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		fmt.Println("No matching commands found.")
		return nil
	}

	for _, r := range records {
		exitStr := ""
		if r.ExitCode != nil {
			if *r.ExitCode == 0 {
				exitStr = " [ok]"
			} else {
				exitStr = fmt.Sprintf(" [exit %d]", *r.ExitCode)
			}
		}
		fmt.Printf("%s  %s  %s%s\n",
			r.Timestamp.Local().Format("2006-01-02 15:04:05"),
			r.Directory,
			r.Command,
			exitStr,
		)
	}
	return nil
}

func parseSince(s string) (time.Time, error) {
	// Try duration-like strings: "1h", "30m", "7d"
	s = strings.TrimSpace(s)
	if len(s) > 1 {
		unit := s[len(s)-1]
		numStr := s[:len(s)-1]
		var dur time.Duration
		switch unit {
		case 'h':
			d, err := time.ParseDuration(numStr + "h")
			if err == nil {
				dur = d
			}
		case 'm':
			d, err := time.ParseDuration(numStr + "m")
			if err == nil {
				dur = d
			}
		case 'd':
			n := 0
			if _, err := fmt.Sscanf(numStr, "%d", &n); err == nil {
				dur = time.Duration(n) * 24 * time.Hour
			}
		case 'w':
			n := 0
			if _, err := fmt.Sscanf(numStr, "%d", &n); err == nil {
				dur = time.Duration(n) * 7 * 24 * time.Hour
			}
		}
		if dur > 0 {
			return time.Now().Add(-dur), nil
		}
	}

	// Try date formats
	for _, layout := range []string{
		"2006-01-02",
		"2006-01-02T15:04:05",
		time.RFC3339,
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse %q as duration or date", s)
}
