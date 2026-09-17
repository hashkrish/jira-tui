// Command jira-tui is a terminal UI client for Jira.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/krishnan/jira-tui/internal/config"
	"github.com/krishnan/jira-tui/internal/jiraclient"
	"github.com/krishnan/jira-tui/internal/ui"
	"github.com/krishnan/jira-tui/internal/ui/components/issuelist"
)

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "jira-tui",
		Short: "Terminal UI client for Jira",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI()
		},
	}

	root.AddCommand(newAuthCmd())
	return root
}

func newAuthCmd() *cobra.Command {
	authCmd := &cobra.Command{
		Use:   "auth",
		Short: "Manage and verify Jira credentials",
	}

	authCmd.AddCommand(&cobra.Command{
		Use:   "test",
		Short: "Verify credentials by calling /rest/api/3/myself",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAuthTest()
		},
	})

	return authCmd
}

func runAuthTest() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config (run setup first): %w", err)
	}

	client := jiraclient.New(cfg)
	me, err := client.Myself()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Printf("Authenticated as %s <%s> against %s\n", me.DisplayName, me.EmailAddress, cfg.BaseURL)
	return nil
}

func runTUI() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config, run `jira-tui auth test` for details: %w", err)
	}

	client := jiraclient.New(cfg)
	me, err := client.Myself()
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	home := issuelist.New(client, issuelist.DefaultJQL)

	app := ui.New(client, home, me.DisplayName, cfg.BaseURL)
	program := tea.NewProgram(app, tea.WithAltScreen())
	_, err = program.Run()
	return err
}
