package logger

import (
	"fmt"
	"os"

	"github.com/withmandala/go-log"
)

var logger = log.New(os.Stderr)

func init() {
	logger.WithDebug()
}

func Fatal(source string, msg string) {
	tolog := fmt.Sprintf("[%s] %s", source, msg)
	logger.Fatal(tolog)
}

func Error(source string, msg string) {
	tolog := fmt.Sprintf("[%s] %s", source, msg)
	logger.Error(tolog)
}

func Warn(source string, msg string) {
	tolog := fmt.Sprintf("[%s] %s", source, msg)
	logger.Warn(tolog)
}

func Info(source string, msg string) {
	tolog := fmt.Sprintf("[%s] %s", source, msg)
	logger.Info(tolog)
}

func Debug(source string, msg string) {
	tolog := fmt.Sprintf("[%s] %s", source, msg)
	logger.Debug(tolog)
}

func Trace(source string, msg string) {
	tolog := fmt.Sprintf("[%s] %s", source, msg)
	logger.Trace(tolog)
}
