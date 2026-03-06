package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/greatbody/terminal-track/internal/hook"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the zsh hook into your .zshrc",
	Long: `Adds a source line to your .zshrc that loads the terminal-track
zsh hook. This enables automatic command recording in all new shells.`,
	RunE: runInstall,
}

var flagUninstall bool

func init() {
	installCmd.Flags().BoolVar(&flagUninstall, "uninstall", false, "Remove the hook from .zshrc")
	rootCmd.AddCommand(installCmd)
}

const hookMarker = "# terminal-track hook"
const hookFileName = "tt-hook.zsh"

func runInstall(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	zshrc := filepath.Join(home, ".zshrc")
	hookDir := filepath.Join(home, ".terminal-track")
	hookPath := filepath.Join(hookDir, hookFileName)
	sourceLine := fmt.Sprintf("%s\n[[ -f %q ]] && source %q\n", hookMarker, hookPath, hookPath)

	if flagUninstall {
		return uninstallHook(zshrc)
	}

	// Write the hook script
	if err := os.MkdirAll(hookDir, 0755); err != nil {
		return fmt.Errorf("create hook dir: %w", err)
	}
	if err := os.WriteFile(hookPath, []byte(hook.ZshHook), 0644); err != nil {
		return fmt.Errorf("write hook script: %w", err)
	}
	fmt.Printf("Wrote hook script to %s\n", hookPath)

	// Check if already installed
	existing, err := os.ReadFile(zshrc)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read .zshrc: %w", err)
	}
	if strings.Contains(string(existing), hookMarker) {
		fmt.Println("Hook already present in .zshrc — nothing to do.")
		return nil
	}

	// Append to .zshrc
	f, err := os.OpenFile(zshrc, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open .zshrc: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString("\n" + sourceLine); err != nil {
		return fmt.Errorf("write to .zshrc: %w", err)
	}

	fmt.Println("Added terminal-track hook to .zshrc")
	fmt.Println("Restart your shell or run: source ~/.zshrc")
	return nil
}

func uninstallHook(zshrc string) error {
	data, err := os.ReadFile(zshrc)
	if err != nil {
		return fmt.Errorf("read .zshrc: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	var filtered []string
	skip := false
	for _, line := range lines {
		if strings.Contains(line, hookMarker) {
			skip = true
			continue
		}
		if skip {
			skip = false
			continue // skip the source line that follows the marker
		}
		filtered = append(filtered, line)
	}

	if err := os.WriteFile(zshrc, []byte(strings.Join(filtered, "\n")), 0644); err != nil {
		return fmt.Errorf("write .zshrc: %w", err)
	}

	fmt.Println("Removed terminal-track hook from .zshrc")
	return nil
}
