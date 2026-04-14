// Package notify provides desktop notification backends for SWAT.
package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// Notifier sends user-facing notifications.
type Notifier interface {
	Notify(message string) error
}

// New creates a Notifier for the given backend name.
// Supported backends: "desktop" (default).
func New(backend string) (Notifier, error) {
	if backend == "" {
		backend = "desktop"
	}
	backend = strings.ToLower(backend)

	switch backend {
	case "desktop":
		return &DesktopNotifier{}, nil
	default:
		return nil, fmt.Errorf("unknown notify backend: %q", backend)
	}
}

// DesktopNotifier sends native desktop notifications via OS-specific commands.
type DesktopNotifier struct{}

// Notify sends a desktop notification. Uses osascript on macOS,
// notify-send on Linux, and PowerShell toast on Windows.
func (d *DesktopNotifier) Notify(message string) error {
	switch runtime.GOOS {
	case "darwin":
		script := fmt.Sprintf(`display notification %q with title "SWAT"`, message)
		return exec.Command("osascript", "-e", script).Run()
	case "linux":
		return exec.Command("notify-send", "SWAT", message).Run()
	case "windows":
		ps := fmt.Sprintf(
			`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] > $null; `+
				`$xml = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent(0); `+
				`$text = $xml.GetElementsByTagName('text'); `+
				`$text.Item(0).AppendChild($xml.CreateTextNode('%s')) > $null; `+
				`$toast = [Windows.UI.Notifications.ToastNotification]::new($xml); `+
				`[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('SWAT').Show($toast)`,
			strings.ReplaceAll(message, "'", "''"),
		)
		return exec.Command("powershell", "-NoProfile", "-Command", ps).Run()
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}
