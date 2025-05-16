package database

// This get retreived from a config file

type MySQLInstance struct {
	User       string
	Password   string
	DBName     string
	Host       string
	Port       string
	OutputPath string
}

func (instance *MySQLInstance) Backup() error {
	return nil
}
