package backup

type Database interface {
	Backup() error
	Restore(filename string) error
}
