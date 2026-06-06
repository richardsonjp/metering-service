package logs

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiGray   = "\x1b[90m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
	ansiYellow = "\x1b[33m"
	ansiCyan   = "\x1b[36m"
)

const timestampLayout = "2006-01-02 15:04:05.000"

type consoleFormatter struct {
	color bool
}

func (f *consoleFormatter) Format(e *logrus.Entry) ([]byte, error) {
	var b bytes.Buffer

	ts := e.Time.Format(timestampLayout)
	level := fmt.Sprintf("%-5s", levelLabel(e.Level))

	if f.color {
		b.WriteString(ansiGray + ts + ansiReset)
		b.WriteString("  ")
		b.WriteString(levelColor(e.Level) + ansiBold + level + ansiReset)
	} else {
		b.WriteString(ts)
		b.WriteString("  ")
		b.WriteString(level)
	}
	b.WriteString("  ")
	b.WriteString(e.Message)

	for _, k := range sortedKeys(e.Data) {
		b.WriteByte(' ')
		if f.color {
			b.WriteString(ansiCyan + k + ansiReset)
		} else {
			b.WriteString(k)
		}
		b.WriteByte('=')
		b.WriteString(renderValue(e.Data[k]))
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}

func levelLabel(l logrus.Level) string {
	switch l {
	case logrus.TraceLevel:
		return "TRACE"
	case logrus.DebugLevel:
		return "DEBUG"
	case logrus.InfoLevel:
		return "INFO"
	case logrus.WarnLevel:
		return "WARN"
	case logrus.ErrorLevel:
		return "ERROR"
	case logrus.FatalLevel:
		return "FATAL"
	case logrus.PanicLevel:
		return "PANIC"
	default:
		return strings.ToUpper(l.String())
	}
}

func levelColor(l logrus.Level) string {
	switch l {
	case logrus.DebugLevel, logrus.TraceLevel:
		return ansiGray
	case logrus.InfoLevel:
		return ansiGreen
	case logrus.WarnLevel:
		return ansiYellow
	default:
		return ansiRed
	}
}

func sortedKeys(data logrus.Fields) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func renderValue(v interface{}) string {
	s := fmt.Sprintf("%v", v)
	if s == "" || strings.ContainsAny(s, " =\"") {
		return fmt.Sprintf("%q", s)
	}
	return s
}
