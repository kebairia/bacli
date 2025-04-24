package main

import "github.com/kebairia/backup/internal/operations"

func main() {
	if err := operations.PerformAllBackups(); err != nil {
		panic(err)
	}
}
