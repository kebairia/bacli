package cmd

import (
	"fmt"

	"github.com/kebairia/backup/internal/config"
	"github.com/kebairia/backup/internal/operations"
	"github.com/spf13/cobra"
)

const BackupDir = "/home/zakaria/dox/work/emploitic/backup/"

var restoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore all databases based on config",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config to derive default output directory
		var cfg config.Config
		if err := cfg.LoadConfig(ConfigFile); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
		fmt.Printf("Restore databases from ...\n")
		if err := operations.RestoreAll(ConfigFile, BackupDir); err != nil {
			return fmt.Errorf("restore failed: %w", err)
		}
		return nil
	},
}

func init() {
	restoreCmd.Flags().
		StringP("source", "s", "", "path to backup source (defaults to <outuptu_dir>)")
}
