package backup

import "errors"

var ErrTimeout = errors.New("operation timed out")

type Database interface {
	GetName() string
	GetEngine() string
	GetPath() string
	Backup() (backupPath string, err error)
	Restore(filename string) error
}
