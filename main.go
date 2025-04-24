package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kebairia/backup/internal/operations"
)

func main() {
	configPath := flag.String("c", "./configs/config.yaml", "Path to YAML config file")
	doBackup := flag.Bool("B", false, "Backup all databases")
	// doRestore := flag.Bool("R", false, "Restore all databases")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		flag.PrintDefaults()
	}
	flag.Parse()

	if *doBackup {
		if err := operations.BackupAll(*configPath); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
	}
}
