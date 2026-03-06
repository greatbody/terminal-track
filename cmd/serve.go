package cmd

import (
	"fmt"
	"net/http"

	"github.com/greatbody/terminal-track/internal/db"
	"github.com/greatbody/terminal-track/internal/web"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the web timeline UI",
	RunE:  runServe,
}

var flagPort int

func init() {
	serveCmd.Flags().IntVarP(&flagPort, "port", "p", 8080, "Port to listen on")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	dbPath, err := db.DefaultDBPath()
	if err != nil {
		return err
	}

	d, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open db: %w", err)
	}
	defer d.Close()

	srv, err := web.New(d)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	addr := fmt.Sprintf(":%d", flagPort)
	fmt.Printf("terminal-track web UI: http://localhost%s\n", addr)
	return http.ListenAndServe(addr, srv.Handler())
}
