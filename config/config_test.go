package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"metering-service/config"
	"metering-service/pkg/utils/bytesize"
)

func TestLoad_Defaults(t *testing.T) {
	for _, key := range []string{"SYSTEM_APP_NAME", "SERVER_ADDR", "SYSTEM_TIME_ZONE", "REQUEST_LIMIT", "STORAGE_LIMIT_BYTES", "MAX_UPLOAD_BYTES", "LOG_LEVEL"} {
		t.Setenv(key, "")
		os.Unsetenv(key)
	}

	cfg := config.Load()
	assert.Equal(t, "metering-service", cfg.System.AppName)
	assert.Equal(t, ":8080", cfg.System.Addr)
	assert.Equal(t, "Asia/Jakarta", cfg.System.TimeZone)
	assert.Equal(t, int64(1000), cfg.Metering.RequestLimit)
	assert.Equal(t, bytesize.GiB, cfg.Metering.StorageLimit)
	assert.Equal(t, bytesize.GiB, cfg.Metering.MaxUploadBytes)
	assert.Equal(t, "info", cfg.Log.Level)
}

func TestLoad_Overrides(t *testing.T) {
	t.Setenv("SERVER_ADDR", ":9999")
	t.Setenv("REQUEST_LIMIT", "5")
	t.Setenv("STORAGE_LIMIT_BYTES", "2048")
	t.Setenv("MAX_UPLOAD_BYTES", "1024")

	cfg := config.Load()
	assert.Equal(t, ":9999", cfg.System.Addr)
	assert.Equal(t, int64(5), cfg.Metering.RequestLimit)
	assert.Equal(t, int64(2048), cfg.Metering.StorageLimit)
	assert.Equal(t, int64(1024), cfg.Metering.MaxUploadBytes)
}

func TestLoad_InvalidIntPanics(t *testing.T) {
	t.Setenv("REQUEST_LIMIT", "not-a-number")
	assert.Panics(t, func() { config.Load() })
}
