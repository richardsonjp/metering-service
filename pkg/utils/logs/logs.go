package logs

import (
	"os"
	"runtime/debug"

	"github.com/mattn/go-isatty"
	"github.com/sirupsen/logrus"
)

type Fields = logrus.Fields

var Log *logrus.Logger

func Init(level string) {
	l := logrus.New()
	l.SetOutput(os.Stdout)
	l.SetFormatter(&consoleFormatter{color: isTerminal(os.Stdout)})
	if lvl, err := logrus.ParseLevel(level); err == nil {
		l.SetLevel(lvl)
	}
	Log = l
}

func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}

func ensure() {
	if Log == nil {
		Init("info")
	}
}

func Info(msg string, f Fields) { ensure(); Log.WithFields(f).Info(msg) }

func Warn(msg string, f Fields) { ensure(); Log.WithFields(f).Warn(msg) }

func Error(msg string, f Fields) { ensure(); Log.WithFields(f).Error(msg) }

func Panic(rec interface{}, f Fields) {
	ensure()
	if f == nil {
		f = Fields{}
	}
	f["panic"] = rec
	f["stack"] = string(debug.Stack())
	Log.WithFields(f).Error("PANIC")
}
