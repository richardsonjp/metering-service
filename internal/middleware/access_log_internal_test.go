package middleware

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSummarizeBody(t *testing.T) {
	assert.Equal(t, "", summarizeBody(nil, ""), "empty body")
	assert.Equal(t, `{"a":1}`, summarizeBody([]byte(`{"a":1}`), "application/json"), "json kept")

	assert.Equal(t, "<3 bytes>", summarizeBody([]byte{0xff, 0xfe, 0xfd}, "application/octet-stream"))

	big := strings.Repeat("a", maxBodyLog+10)
	got := summarizeBody([]byte(big), "text/plain")
	assert.True(t, strings.HasSuffix(got, "…(truncated)"))
	assert.Equal(t, maxBodyLog+len("…(truncated)"), len(got))
}

func TestIsTextual(t *testing.T) {
	assert.True(t, isTextual("application/json; charset=utf-8", []byte("{}")))
	assert.True(t, isTextual("text/plain", []byte("hi")))
	assert.True(t, isTextual("", []byte("plain utf8 fallback")))
	assert.False(t, isTextual("", []byte{0xff, 0xfe}))
}
