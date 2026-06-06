package logs

import (
	"strings"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func entry(level logrus.Level, msg string, data logrus.Fields) *logrus.Entry {
	e := logrus.NewEntry(logrus.New())
	e.Time = time.Date(2026, 6, 6, 15, 7, 52, 123_000_000, time.UTC)
	e.Level = level
	e.Message = msg
	e.Data = data
	return e
}

func TestConsoleFormatter_TimestampFirstNoColor(t *testing.T) {
	f := &consoleFormatter{color: false}
	out, err := f.Format(entry(logrus.InfoLevel, "server starting", logrus.Fields{"addr": ":8080"}))
	require.NoError(t, err)
	line := string(out)

	assert.True(t, strings.HasPrefix(line, "2026-06-06 15:07:52.123"), "timestamp first: %q", line)
	assert.Contains(t, line, "INFO")
	assert.Contains(t, line, "server starting")
	assert.Contains(t, line, "addr=:8080")
	assert.NotContains(t, line, "\x1b", "no ANSI color when color is off")
	assert.True(t, strings.HasSuffix(line, "\n"))
}

func TestConsoleFormatter_ColorAddsAnsi(t *testing.T) {
	f := &consoleFormatter{color: true}
	out, err := f.Format(entry(logrus.WarnLevel, "heads up", logrus.Fields{}))
	require.NoError(t, err)
	assert.Contains(t, string(out), "\x1b[", "ANSI escape present when color is on")
}

func TestRenderValue_QuotesWhenNeeded(t *testing.T) {
	assert.Equal(t, "plain", renderValue("plain"))
	assert.Equal(t, `"has space"`, renderValue("has space"))
	assert.Equal(t, `""`, renderValue(""))
	assert.Equal(t, "200", renderValue(200))
}

func TestLevelColor(t *testing.T) {
	assert.Equal(t, ansiGray, levelColor(logrus.DebugLevel))
	assert.Equal(t, ansiGreen, levelColor(logrus.InfoLevel))
	assert.Equal(t, ansiYellow, levelColor(logrus.WarnLevel))
	assert.Equal(t, ansiRed, levelColor(logrus.ErrorLevel))
}

func TestConsoleFormatter_DebugColorAndFields(t *testing.T) {
	f := &consoleFormatter{color: true}
	out, err := f.Format(entry(logrus.DebugLevel, "dbg", logrus.Fields{"a": 1, "b": "two words"}))
	require.NoError(t, err)
	line := string(out)
	assert.Contains(t, line, "DEBUG")
	assert.Contains(t, line, ansiCyan, "field keys are colored")
	assert.Contains(t, line, `"two words"`, "spaced value is quoted")
}
