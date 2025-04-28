package cmd

import (
	"github.com/kebairia/backup/internal/logger"
	"github.com/spf13/cobra"
)

// ConfigFile is the path to the YAML configuration.
var (
	ConfigFile string
	// rootCmd is the base command for bacli.
	rootCmd = &cobra.Command{
		Use:   "bacli",
		Short: "CLI tool for database backup and restore",
		Long: `bacli provides subcommands to back up and restore
databases based on your YAML configuration file.`,
	}
)

// Execute runs the root command.
func Execute() {
	log, err := logger.Init()
	_ = err
	if err := rootCmd.Execute(); err != nil {
		log.Error("Error executing command: %v", err)
	}
}

func init() {
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
}
