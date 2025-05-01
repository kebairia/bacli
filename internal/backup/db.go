package backup

type Database interface {
	GetName() string
	GetEngine() string
	GetPath() string
	Backup() (backupPath string, err error)
	Restore(filename string) error
}
