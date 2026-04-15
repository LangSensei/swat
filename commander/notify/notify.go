// Package notify provides notification backends for SWAT.
package notify

import (
	"fmt"
	"strings"
)

// Notifier sends user-facing notifications.
type Notifier interface {
	Notify(message string) error
}

// New creates a Notifier for the given backend name.
// Supported backends: "desktop" (default), "openclaw".
func New(backend string) (Notifier, error) {
	if backend == "" {
		backend = "desktop"
	}
	backend = strings.ToLower(backend)

	switch backend {
	case "desktop":
		return &DesktopNotifier{}, nil
	case "openclaw":
		return newOpenClawNotifier()
	default:
		return nil, fmt.Errorf("unknown notify backend: %q", backend)
	}
}
