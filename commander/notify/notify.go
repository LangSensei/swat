// Package notify provides notification targets for SWAT.
package notify

import (
	"fmt"
	"strings"
)

// Notifier sends user-facing notifications.
type Notifier interface {
	Notify(opID string, message string) error
}

// New creates a Notifier for the given target name.
// Supported targets: "desktop" (default), "openclaw".
func New(target string) (Notifier, error) {
	if target == "" {
		target = "desktop"
	}
	target = strings.ToLower(target)

	switch target {
	case "desktop":
		return &DesktopNotifier{}, nil
	case "openclaw":
		return newOpenClawNotifier()
	default:
		return nil, fmt.Errorf("unknown notify target: %q", target)
	}
}
