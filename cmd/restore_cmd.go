package cmd

import (
	"fmt"

	"github.com/kebairia/backup/internal/config"
	"github.com/kebairia/backup/internal/operations"
	"github.com/spf13/cobra"
)

const BackupDir = "/home/zakaria/dox/work/emploitic/bacli/"

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore all databases based on config",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config to derive default output directory
		var cfg config.Config
		if err := cfg.LoadConfig(ConfigFile); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if err := operations.RestoreAll(ConfigFile); err != nil {
		}
		return nil
	},
}

func init() {
	restoreCmd.Flags().
		StringP("source", "s", "", "path to backup source (defaults to <outuptu_dir>)")
}
