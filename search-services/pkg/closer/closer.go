// Package closer helps close objects and log it
package closer

import (
	"io"
	"log/slog"
)

func CloseOrLog(c io.Closer) {
	if c == nil {
		return
	}

	if err := c.Close(); err != nil {
		slog.Error("failed to close", "err", err)
	}
}

func CloseOrIgnore(c io.Closer) {
	if c == nil {
		return
	}
	_ = c.Close()
}
