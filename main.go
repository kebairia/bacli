package main

import (
	"fmt"
	"log"

	"github.com/kebairia/backup/internal/backup"
	"github.com/kebairia/backup/internal/operations"
)

func main() {
	mongo := backup.NewMongoDBInstance(
		"root",
		"example",
		"kvmcli",
		"localhost",
		"27017",
		"./backups",
	)
	postgres := backup.NewPostgresInstance(
		"admin",
		"example",
		"mydb",
		"localhost",
		"5432",
		"./backups",
	)
	if err := operations.PerformBackup(postgres); err != nil {
		log.Fatalf("PostgreSQL backup error: %v", err)
	}

	if err := operations.PerformBackup(mongo); err != nil {
		log.Fatalf("MongoDB backup error: %v", err)
	}
	fmt.Println("All backups")
	dbs := []backup.Database{mongo, postgres}
	if err := operations.PerformAllBackups(dbs); err != nil {
		log.Fatalf("backup failed : %v", err)
	}
}
