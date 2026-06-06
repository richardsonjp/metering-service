package bytesize_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"metering-service/pkg/utils/bytesize"
)

func TestHuman(t *testing.T) {
	tests := []struct {
		name string
		in   int64
		want string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 512, "512 B"},
		{"just-below-KiB", 1023, "1023 B"},
		{"one-KiB", bytesize.KiB, "1.0 KiB"},
		{"half-MiB", 512 * bytesize.KiB, "512.0 KiB"},
		{"one-MiB", bytesize.MiB, "1.0 MiB"},
		{"one-GiB", bytesize.GiB, "1.0 GiB"},
		{"one-and-half-GiB", bytesize.GiB + 512*bytesize.MiB, "1.5 GiB"},
		{"unlimited", -1, "unlimited"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, bytesize.Human(tt.in))
		})
	}
}
