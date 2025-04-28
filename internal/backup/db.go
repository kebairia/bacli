package backup

type Database interface {
	GetName() string
	GetEngine() string
	Backup() (backupPath string, err error)
	Restore(filename string) error
}
