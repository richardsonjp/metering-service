package bytesize

import "fmt"

const (
	B   int64 = 1
	KiB       = 1024 * B
	MiB       = 1024 * KiB
	GiB       = 1024 * MiB
	TiB       = 1024 * GiB
)

func Human(b int64) string {
	if b < 0 {
		return "unlimited"
	}
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(b)/float64(div), "KMGTPE"[exp])
}
