package cmd

import (
	"fmt"
	"os"

	"github.com/kebairia/backup/internal/operations"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Backup all databases as per config",
	Run: func(cmd *cobra.Command, args []string) {
		if ConfigFile == "" {
			fmt.Errorf("Config file is required (-f flag)")
		}
		if err := operations.BackupAll(ConfigFile, "/home/zakaria/dox/work/emploitic/bacli"); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Bind the config file flag to the global variable.
	backupCmd.Flags().
		StringVarP(&ConfigFile, "config", "c", "./configs/config.yaml", "path to YAML config file")
}
