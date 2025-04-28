package backup

type Database interface {
	GetName() string
	Backup() (backupPath string, err error)
	Restore(filename string) error
}
