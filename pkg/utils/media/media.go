package media

import (
	"strings"

	"github.com/gabriel-vasile/mimetype"
)

const SniffLen = 3072

func Detect(head []byte) string {
	return mimetype.Detect(head).String()
}

func IsImageOrVideo(mime string) bool {
	return strings.HasPrefix(mime, "image/") || strings.HasPrefix(mime, "video/")
}
