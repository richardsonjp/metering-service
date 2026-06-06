package logs_test

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"metering-service/pkg/utils/logs"
)

func TestInitSetsLevel(t *testing.T) {
	logs.Init("warn")
	require.NotNil(t, logs.Log)
	assert.Equal(t, logrus.WarnLevel, logs.Log.GetLevel())
}

func TestInitUnknownLevelKeepsDefault(t *testing.T) {
	logs.Init("not-a-level")
	require.NotNil(t, logs.Log)
	assert.Equal(t, logrus.InfoLevel, logs.Log.GetLevel())
}

func TestHelpersDoNotPanic(t *testing.T) {
	logs.Init("debug")
	assert.NotPanics(t, func() {
		logs.Info("info", logs.Fields{"k": "v"})
		logs.Warn("warn", logs.Fields{"k": "v"})
		logs.Error("error", logs.Fields{"k": "v"})
		logs.Panic("boom", nil)
	})
}

func TestHelpersLazyInitWhenLogNil(t *testing.T) {
	logs.Log = nil
	assert.NotPanics(t, func() { logs.Info("info", logs.Fields{}) })
	assert.NotNil(t, logs.Log, "ensure() should have initialized the logger")
}
