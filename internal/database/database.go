package database

import "errors"

var (
	ErrTimeout       = errors.New("operation timed out")
	ErrBackupFailed  = errors.New("backup failed")
	ErrRestoreFailed = errors.New("restore failed")
)

type Database interface {
	GetName() string
	GetEngine() string
	GetPath() string
	Backup() (backupPath string, err error)
	Restore(filename string) error
}
