package logger

import (
	"go.uber.org/zap"
)

// Log is a global logger instance that other packages can use.
var Log *zap.Logger

func Init() error {
	var err error
	// Log, err = zap.NewProduction()
	Log, err = zap.NewDevelopment() // <-- pretty output with colors
	Log.WithOptions()
	if err != nil {
		return err
	}
	return nil
}

func Cleanup() {
	if Log != nil {
		_ = Log.Sync()
	}
}
