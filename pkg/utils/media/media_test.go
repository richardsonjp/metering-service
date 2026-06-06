package media_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"metering-service/pkg/utils/media"
)

func TestDetectAndIsImageOrVideo(t *testing.T) {
	tests := []struct {
		name    string
		head    []byte
		allowed bool
	}{
		{"png", []byte("\x89PNG\r\n\x1a\n"), true},
		{"jpeg", []byte{0xFF, 0xD8, 0xFF, 0xE0}, true},
		{"gif", []byte("GIF89a"), true},
		{"mp4", []byte{0x00, 0x00, 0x00, 0x10, 'f', 't', 'y', 'p', 'm', 'p', '4', '2', 0, 0, 0, 0}, true},
		{"mov-quicktime", []byte{0x00, 0x00, 0x00, 0x14, 'f', 't', 'y', 'p', 'q', 't', ' ', ' ', 0, 0, 0, 0, 'q', 't', ' ', ' '}, true},
		{"mkv", append([]byte{0x1A, 0x45, 0xDF, 0xA3, 0x42, 0x82, 0x88}, []byte("matroska")...), true},
		{"plain-text", []byte("just some plain text, not media"), false},
		{"pdf", []byte("%PDF-1.7"), false},
		{"empty", []byte{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mime := media.Detect(tt.head)
			assert.Equal(t, tt.allowed, media.IsImageOrVideo(mime), "mime=%q", mime)
		})
	}
}

func TestIsImageOrVideo(t *testing.T) {
	assert.True(t, media.IsImageOrVideo("image/png"))
	assert.True(t, media.IsImageOrVideo("video/mp4"))
	assert.False(t, media.IsImageOrVideo("application/octet-stream"))
	assert.False(t, media.IsImageOrVideo("text/plain; charset=utf-8"))
}
